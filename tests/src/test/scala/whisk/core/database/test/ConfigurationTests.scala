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

package whisk.core.database.test

import akka.http.scaladsl.Http
import akka.http.scaladsl.model._
import akka.http.scaladsl.model.headers.{Authorization, BasicHttpCredentials}
import akka.http.scaladsl.unmarshalling.Unmarshal
import akka.stream.ActorMaterializer
import common.{StreamLogging, WskActorSystem}
import org.junit.runner.RunWith
import org.scalatest.FlatSpec
import org.scalatest.junit.JUnitRunner

import scala.concurrent.Await
import scala.concurrent.duration.DurationInt

@RunWith(classOf[JUnitRunner])
class ConfigurationTests extends FlatSpec with DatabaseScriptTestUtils with StreamLogging with WskActorSystem {

  implicit val materializer = ActorMaterializer()
  val authHeader = Authorization(BasicHttpCredentials(dbUsername, dbPassword))


  behavior of "CouchDB Configuration"

  it should "include reduce_limit as false" in {

    val request = Http()
      .singleRequest(HttpRequest(
        method = HttpMethods.GET,
        uri = Uri(s"${dbUrl}/_node/couchdb@${dbHost}/_config/query_server_config/reduce_limit"),
        headers = List(authHeader)))
      .flatMap { response =>
        Unmarshal(response).to[String].map { resBody =>
          withClue(s"Error in Body: $resBody")(response.status shouldBe StatusCodes.OK)
          resBody.trim.replace("\"","") shouldBe "false"
          resBody
        }
      }

    Await.result(request, 15.seconds)
  }
}
