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

package org.apache.openwhisk.core

import akka.Done
import akka.actor.ActorSystem
import akka.kafka.ProducerSettings
import akka.kafka.scaladsl.Producer
import akka.stream.ActorMaterializer
import akka.stream.scaladsl.Source
import com.google.common.base.{Stopwatch, Throwables}
import org.apache.kafka.clients.producer.ProducerRecord
import org.apache.kafka.common.serialization.StringSerializer
import org.apache.openwhisk.common.{Counter, Logging, TransactionId}
import spray.json._

import scala.collection.concurrent.TrieMap
import scala.concurrent.{ExecutionContext, Future}
import scala.util.{Failure, Success}

case class LoadGeneratorConfig(port: Int)

trait MessageGenerator {
  def next(index: Int)(implicit tid: TransactionId): JsObject
}

case class LoadGenerator(config: LoadGeneratorConfig, count: Int, topic: String, generator: MessageGenerator)(
  implicit system: ActorSystem,
  materializer: ActorMaterializer,
  logging: Logging,
  tid: TransactionId) {
  import LoadGenerator._

  private implicit val ec: ExecutionContext = system.dispatcher
  private val w = Stopwatch.createStarted()

  val id: Long = idCounter.next()
  val progressCounter: Counter = new Counter

  def status: String = s"$id - Sent ${progressCounter.cur} messages to $topic since $w"

  logging.info(this, s"Starting producing $count messages for topic $topic")
  generators.put(id, this)

  val done: Future[Done] = Source(1 to count)
    .map(i => generator.next(i).compactPrint)
    .wireTap(_ => progressCounter.next())
    .map(value => new ProducerRecord[String, String](topic, value))
    .runWith(Producer.plainSink(producerSettings()))

  done.onComplete(_ => generators.remove(id))

  done.onComplete {
    case Success(_) => logging.info(this, s"Successfully published $count messages to $topic in $w")
    case Failure(t) => logging.warn(this, "Failed to produce all the messages " + Throwables.getStackTraceAsString(t))
  }

  private def producerSettings(): ProducerSettings[String, String] = {
    ProducerSettings(system, new StringSerializer, new StringSerializer)
  }
}

object LoadGenerator {
  val idCounter = new Counter
  val generators = new TrieMap[Long, LoadGenerator]()
}
