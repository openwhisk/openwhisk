/*
 * Copyright 2015-2016 IBM Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package whisk.core.database.test

import scala.concurrent.Await
import scala.concurrent.Future
import scala.concurrent.Promise
import scala.concurrent.duration.DurationDouble
import scala.concurrent.duration.DurationInt
import scala.util._

import org.junit.runner.RunWith
import org.scalatest.BeforeAndAfterAll
import org.scalatest.FlatSpec
import org.scalatest.Matchers
import org.scalatest.concurrent.PatienceConfiguration.Timeout
import org.scalatest.concurrent.ScalaFutures
import org.scalatest.junit.JUnitRunner

import akka.actor.Props
import akka.http.scaladsl.model._
import akka.stream.scaladsl._
import akka.util.ByteString
import common.WskActorSystem
import spray.json._
import spray.json.DefaultJsonProtocol._
import whisk.core.WhiskConfig
import whisk.core.WhiskConfig._
import whisk.test.http.RESTProxy

@RunWith(classOf[JUnitRunner])
class CouchDbRestClientTests extends FlatSpec
    with Matchers
    with ScalaFutures
    with BeforeAndAfterAll
    with WskActorSystem
    with DbUtils {

    override implicit val patienceConfig = PatienceConfig(timeout = 10.seconds, interval = 0.5.seconds)

    val config = new WhiskConfig(Map(
        dbProvider -> null,
        dbProtocol -> null,
        dbUsername -> null,
        dbPassword -> null,
        dbHost -> null,
        dbPort -> null))

    // We assume this DB does not exist.
    val dbName = s"whisk_test_db_${Random.nextInt().abs.toInt}"

    val client = new ExtendedCouchDbRestClient(config.dbProtocol, config.dbHost, config.dbPort.toInt, config.dbUsername, config.dbPassword, dbName)

    override def beforeAll() {
        super.beforeAll()
        whenReady(client.createDb()) { r =>
            assert(r.isRight)
        }
    }

    override def afterAll() {
        whenReady(client.deleteDb()) { r =>
            assert(r.isRight)
        }
        super.afterAll()
    }

    def checkInstanceInfoResponse(response: Either[StatusCode, JsObject]): Unit = response match {
        case Right(obj) =>
            assert(obj.fields.contains("couchdb"), "response object doesn't contain 'couchdb'")

        case Left(code) =>
            assert(false, s"unsuccessful response (code ${code.intValue})")
    }

    behavior of "CouchDbRestClient"

    it should "successfully access the DB instance info" in {
        assume(config.dbProvider == "Cloudant" || config.dbProvider == "CouchDB")
        val f = client.instanceInfo()
        whenReady(f) { e => checkInstanceInfoResponse(e) }
    }

    ignore /* it */ should "successfully access the DB despite transient connection failures" in {
        assume(config.dbProvider == "Cloudant" || config.dbProvider == "CouchDB")

        val dbAuthority = Uri.Authority(
            host = Uri.Host(config.dbHost),
            port = config.dbPort.toInt)

        val proxyPort = 15975
        val proxyActor = actorSystem.actorOf(Props(new RESTProxy("0.0.0.0", proxyPort)(dbAuthority, config.dbProtocol == "https")))

        val proxiedClient = new ExtendedCouchDbRestClient("http", "localhost", proxyPort, config.dbUsername, config.dbPassword, dbName)

        // sprays the client with requests, makes sure they are all answered
        // despite temporary connection failure.
        val numRequests = 30
        val timeSpan = 5.seconds
        val delta = timeSpan / numRequests

        val promises = Vector.fill(numRequests)(Promise[Either[StatusCode, JsObject]])

        for (i <- 0 until numRequests) {
            actorSystem.scheduler.scheduleOnce(delta * (i + 1)) {
                proxiedClient.instanceInfo().andThen({ case r => promises(i).tryComplete(r) })
            }
        }

        // Mayhem! Havoc!
        actorSystem.scheduler.scheduleOnce(2.5.seconds, proxyActor, RESTProxy.UnbindFor(1.second))

        // What a type!
        val futures: Vector[Future[Try[Either[StatusCode, JsObject]]]] =
            promises.map(_.future.map(e => Success(e)).recover { case t: Throwable => Failure(t) })

        whenReady(Future.sequence(futures), Timeout(timeSpan * 2)) { results =>
            // We check that the first result was OK
            // (i.e. the service worked before the disruption)
            results.head.toOption shouldBe defined
            checkInstanceInfoResponse(results.head.get)

            // We check that the last result was OK
            // (i.e. the service worked again after the disruption)
            results.last.toOption shouldBe defined
            checkInstanceInfoResponse(results.last.get)

            // We check that there was at least one error
            // (i.e. we did manage to unbind for a while)
            results.find(_.isFailure) shouldBe defined
        }
    }

    it should "upload then download an attachment" in {
        assume(config.dbProvider == "Cloudant" || config.dbProvider == "CouchDB")

        val docId = "some_doc"
        val doc = JsObject("greeting" -> JsString("hello"))
        val attachmentName = "misc"
        val attachmentType = ContentTypes.`text/plain(UTF-8)`
        val attachment = ("""
            | This could have been poetry.
            | But it isn't.
        """).stripMargin

        val attachmentSource = Source.single(ByteString.fromString(attachment))

        val retrievalSink = Sink.fold[String, ByteString]("")((s, bs) => s + bs.decodeString("UTF-8"))

        val insertAndRetrieveResult: Future[(ContentType, String)] = for (
            docResponse <- client.putDoc(docId, doc);
            Right(d) = docResponse;
            rev1 = d.fields("rev").convertTo[String];
            attResponse <- client.putAttachment(docId, rev1, attachmentName, attachmentType, attachmentSource);
            Right(a) = attResponse;
            rev2 = a.fields("rev").convertTo[String];
            retResponse <- client.getAttachment[String](docId, rev2, attachmentName, retrievalSink);
            Right(pair) = retResponse
        ) yield pair

        whenReady(insertAndRetrieveResult) {
            case (t, r) =>
                assert(t === ContentTypes.`text/plain(UTF-8)`)
                assert(r === attachment)
        }
    }

    it should "fail if group=true is used together with reduce=false" in {
        intercept[IllegalArgumentException] {
            Await.result(client.executeView("", "")(reduce = false, group = true), 15.seconds)
        }
    }

    it should "check group Parameter on view-execution" in {
        assume(config.dbProvider == "Cloudant" || config.dbProvider == "CouchDB")

        val ids = List("some_doc_1", "some_doc_2", "some_doc_3", "some_doc_4", "some_doc_5")
        val docs = Map(
            ids(0) -> JsObject("key" -> JsString("a"),
                "value" -> JsNumber(1)),
            ids(1) -> JsObject("key" -> JsString("a"),
                "value" -> JsNumber(2)),
            ids(2) -> JsObject("key" -> JsString("b"),
                "value" -> JsNumber(3)),
            ids(3) -> JsObject("key" -> JsString("b"),
                "value" -> JsNumber(4)),
            ids(4) -> JsObject("key" -> JsString("c"),
                "value" -> JsNumber(5)))
        val designDocName = "testDocument"
        val viewName = "sumOfValues"
        val designDoc = JsObject(
            "views" -> JsObject(viewName -> JsObject(
                "reduce" -> JsString("_sum"),
                "map" -> JsString("function (doc) {\n  if(doc.key && doc.value) {\n    emit([doc.key], doc.value);\n  }\n}"))),
            "language" -> JsString("javascript"))

        Await.result(client.putDoc(s"_design/$designDocName", designDoc), 15.seconds)
        docs.map {
            case (id, doc) =>
                Await.result(client.putDoc(id, doc), 15.seconds)
        }

        waitOnView(client, designDocName, viewName, docs.size)

        val resultGroupedTrue = Await.result(client.executeView(designDocName, viewName)(reduce = true, group = true), 15.seconds)
        resultGroupedTrue should be('right)
        val jsObjectTrue = resultGroupedTrue.right.get
        jsObjectTrue.fields("rows").convertTo[List[JsObject]].length should equal(3)

        val resultGroupedFalse = Await.result(client.executeView(designDocName, viewName)(reduce = true, group = false), 15.seconds)
        resultGroupedFalse should be('right)
        val jsObjectFalse = resultGroupedFalse.right.get
        jsObjectFalse.fields("rows").convertTo[List[JsObject]].length should equal(1)

        val resultGroupedWithout = Await.result(client.executeView(designDocName, viewName)(reduce = true), 15.seconds)
        resultGroupedWithout should be('right)
        val jsObjectWithout = resultGroupedWithout.right.get
        jsObjectWithout.fields("rows").convertTo[List[JsObject]].length should equal(1)
    }
}
