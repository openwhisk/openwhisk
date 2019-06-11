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

package org.apache.openwhisk.core.grpc.test

import akka.actor.{Actor, ActorSystem, Props}
import akka.stream.ActorMaterializer
import akka.testkit.{ImplicitSender, TestKit}
import common.StreamLogging
import org.apache.openwhisk.common.TransactionId
import org.apache.openwhisk.core.connector.ActivationMessage
import org.apache.openwhisk.core.entity._
import org.apache.openwhisk.core.grpc.ActivationServiceImpl
import org.apache.openwhisk.core.grpc.NoMemoryQueue
import org.apache.openwhisk.grpc.{FetchRequest, FetchResponse}
import org.junit.runner.RunWith
import org.scalatest.junit.JUnitRunner
import org.scalatest.{BeforeAndAfterAll, FlatSpecLike, Matchers}
import org.apache.openwhisk.core.grpc.{ActivationRequest, ActivationResponse}
import org.scalatest.concurrent.ScalaFutures

import scala.concurrent.duration._

@RunWith(classOf[JUnitRunner])
class ActivationServiceImplTests
    extends TestKit(ActorSystem("ActivationService"))
    with ImplicitSender
    with FlatSpecLike
    with Matchers
    with BeforeAndAfterAll
    with ScalaFutures
    with StreamLogging {

  override def afterAll = TestKit.shutdownActorSystem(system)

  behavior of "ActivationService"

  implicit val mat = ActorMaterializer()
  implicit val ec = system.dispatcher

  val messageTransId = TransactionId(TransactionId.testing.meta.id)
  val uuid = UUID()
  val testNamespace = "test-namespace"
  val testFQN = FullyQualifiedEntityName(EntityPath(testNamespace), EntityName("test-action"))
  val testREV = DocRevision("1-fake")
  val testDOC = testFQN.toDocId.asDocInfo(testREV)
  val message = ActivationMessage(
    messageTransId,
    FullyQualifiedEntityName(EntityPath(testNamespace), EntityName("test-action")),
    DocRevision.empty,
    Identity(
      Subject(),
      Namespace(EntityName(testNamespace), uuid),
      BasicAuthenticationAuthKey(uuid, Secret()),
      Set.empty),
    ActivationId.generate(),
    ControllerInstanceId("0"),
    blocking = false,
    content = None)

  it should "delegate the FetchRequest to the QueueManager" in {

    val mock = system.actorOf(Props(new Actor() {
      override def receive: Receive = {
        case ActivationRequest(fqn, docInfo) =>
          testActor ! ActivationRequest(fqn, docInfo)
          sender() ! ActivationResponse(Right(message))
      }
    }))

    val activationServiceImpl = ActivationServiceImpl(mock)

    activationServiceImpl
      .fetchActivation(FetchRequest(testFQN.serialize, testREV.serialize))
      .futureValue shouldBe FetchResponse(ActivationResponse(Right(message)).serialize)

    expectMsg(ActivationRequest(testFQN, testDOC))
  }

  it should "return without any retry if there is no such queue" in {
    val mock = system.actorOf(Props(new Actor() {

      override def receive: Receive = {
        case ActivationRequest(fqn, doc) =>
          testActor ! ActivationRequest(fqn, doc)
          sender() ! ActivationResponse(Left(NoMemoryQueue()))
      }
    }))

    val activationServiceImpl = ActivationServiceImpl(mock)

    activationServiceImpl
      .fetchActivation(FetchRequest(testFQN.serialize, testREV.serialize))
      .futureValue shouldBe FetchResponse(ActivationResponse(Left(NoMemoryQueue())).serialize)

    expectMsg(ActivationRequest(testFQN, testDOC))
    expectNoMessage(200.millis)
  }
}
