/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package whisk.core.database

import java.security.MessageDigest

import akka.event.Logging.ErrorLevel
import akka.stream.SinkShape
import akka.stream.scaladsl.{Broadcast, Flow, GraphDSL, Keep, Sink}
import akka.util.ByteString
import spray.json.DefaultJsonProtocol._
import spray.json.{JsObject, RootJsonFormat}
import whisk.common.{Logging, StartMarker, TransactionId}
import whisk.core.entity.{DocInfo, DocRevision, DocumentReader, WhiskDocument}

import scala.concurrent.{ExecutionContext, Future}

private[database] object StoreUtils {

  def reportFailure[T](f: Future[T], start: StartMarker, failureMessage: Throwable => String)(
    implicit transid: TransactionId,
    logging: Logging,
    ec: ExecutionContext): Future[T] = {
    f.onFailure({
      case _: ArtifactStoreException => // These failures are intentional and shouldn't trigger the catcher.
      case x                         => transid.failed(this, start, failureMessage(x), ErrorLevel)
    })
    f
  }

  def checkDocHasRevision(doc: DocInfo): Unit = {
    require(doc != null, "doc undefined")
    require(doc.rev.rev != null, "doc revision must be specified")
  }

  def deserialize[A <: DocumentAbstraction, DocumentAbstraction](doc: DocInfo, js: JsObject)(
    implicit docReader: DocumentReader,
    ma: Manifest[A],
    jsonFormat: RootJsonFormat[DocumentAbstraction]): A = {
    val asFormat = try {
      docReader.read(ma, js)
    } catch {
      case _: Exception => jsonFormat.read(js)
    }

    if (asFormat.getClass != ma.runtimeClass) {
      throw DocumentTypeMismatchException(
        s"document type ${asFormat.getClass} did not match expected type ${ma.runtimeClass}.")
    }

    val deserialized = asFormat.asInstanceOf[A]

    val responseRev = js.fields("_rev").convertTo[String]
    assert(doc.rev.rev == null || doc.rev.rev == responseRev, "Returned revision should match original argument")
    // FIXME remove mutability from appropriate classes now that it is no longer required by GSON.
    deserialized.asInstanceOf[WhiskDocument].revision(DocRevision(responseRev))
  }

  def combinedSink[T](dest: Sink[ByteString, T]): Sink[ByteString, (Future[String], Future[Long], T)] = {
    Sink.fromGraph(GraphDSL.create(digestSink(), lengthSink(), dest)((_, _, _)) {
      implicit builder => (dgs, ls, dests) =>
        import GraphDSL.Implicits._

        val bcast = builder.add(Broadcast[ByteString](3))

        bcast ~> dgs.in
        bcast ~> ls.in
        bcast ~> dests.in

        SinkShape(bcast.in)
    })
  }

  private def digestSink(): Sink[ByteString, Future[String]] = {
    Flow[ByteString]
      .fold(emptyDigest())((digest, bytes) => { digest.update(bytes.toArray); digest })
      .map(md => md.digest().map("%02X" format _).mkString)
      .toMat(Sink.head)(Keep.right)
  }

  private def lengthSink(): Sink[ByteString, Future[Long]] = {
    Sink.fold[Long, ByteString](0)((length, bytes) => length + bytes.size)
  }

  private def emptyDigest() = MessageDigest.getInstance("SHA-256")
}
