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

package whisk.core.connector.tests

import akka.actor.ActorSystem
import common.rest.WskRest
import org.scalatest.FlatSpec
import org.scalatest.Matchers
import whisk.connector.kafka.KafkaConsumerConnector
import whisk.core.WhiskConfig

import scala.concurrent.duration._
import common._
import whisk.core.connector.{Activation, EventMessage, Metric}

class UserEventTests extends FlatSpec with Matchers with WskTestHelpers with StreamLogging {

  implicit val wskprops = WskProps()
  implicit val system = ActorSystem("UserEventTestSystem")
  val config = new WhiskConfig(WhiskConfig.kafkaHosts)

  val wsk = new WskRest

  val groupid = "kafkatest"
  val topic = "events"
  val maxPollInterval = 10.seconds

  val consumer = new KafkaConsumerConnector(config.kafkaHosts, groupid, topic)
  val testActionsDir = WhiskProperties.getFileRelativeToWhiskHome("tests/dat/actions")

  behavior of "user events"

  it should "invoke an action and produce user events" in withAssetCleaner(wskprops) { (wp, assetHelper) =>
    val file = Some(TestUtils.getTestActionFilename("hello.js"))
    val name = "testUserEvents"

    assetHelper.withCleaner(wsk.action, name, confirmDelete = true) { (action, _) =>
      action.create(name, file)
    }

    val run = wsk.action.invoke(name, blocking = true)

    withActivation(wsk.activation, run) { result =>
      withClue("successfully invoke an action") {
        result.response.status shouldBe "success"
      }
    }

    val received =
      consumer.peek(maxPollInterval).map { case (_, _, _, msg) => EventMessage.parse(new String(msg, "utf-8")) }
    received.map(event => {
      event match {
        case EventMessage(_, a: Activation, _, _, _, _, _) => Array(a.statusCode) should contain oneOf (0, 1, 2, 3)
        case EventMessage(_, m: Metric, _, _, _, _, _) =>
          Array(m.metricName) should contain oneOf ("concurrent_activations", "ConcurrentRateLimit", "TimedRateLimit")
      }
    })
    // produce at least 2 events - an Activation and a 'concurrent_invocation' Metric
    // >= 2 is due to events that might have potentially occurred in between
    received.size should be >= 2
    consumer.commit()
  }

  it should "produce a metric when user exceeds system defined throttle limit" in withAssetCleaner(wskprops) {
    (wp, assetHelper) =>
      val concurrentLimit = WhiskProperties.getProperty("limits.actions.invokes.perMinute").toInt
      val numberOfControllers = WhiskProperties.getProperty("controller.instances").toInt

      val overhead = if (WhiskProperties.getControllerHosts.split(",").length > 1) 1.2 else 1.0
      val activationsToFire = (concurrentLimit * numberOfControllers * overhead).toInt + 2

      for (i <- 1 to activationsToFire) {
        val file = Some(TestUtils.getTestActionFilename("hello.js"))
        val name = s"testUserEvents$i"

        assetHelper.withCleaner(wsk.action, name, confirmDelete = true) { (action, _) =>
          action.create(name, file)
        }
        wsk.action.invoke(name, expectedExitCode = TestUtils.DONTCARE_EXIT)
      }

      val events =
        consumer.peek(maxPollInterval).map { case (_, _, _, msg) => EventMessage.parse(new String(msg, "utf-8")) }
      val throttled = events.map(event => {
        event match {
          case EventMessage(_, m: Metric, _, _, _, _, _) =>
            Array(m.metricName).filter(_.equalsIgnoreCase("TimedRateLimit"))
          case _ => //
        }
      })
      throttled.size should be > 0
      consumer.commit()
  }

}
