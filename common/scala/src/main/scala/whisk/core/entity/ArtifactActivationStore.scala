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

package whisk.core.entity

import java.time.Instant

import akka.actor.ActorSystem
import akka.stream.ActorMaterializer

import spray.json.JsObject

import whisk.common.{Logging, TransactionId}
import whisk.core.database.{ArtifactStore, CacheChangeNotification, StaleParameter}

import scala.concurrent.Future
import scala.util.{Failure, Success}

class ArtifactActivationStore(actorSystem: ActorSystem, actorMaterializer: ActorMaterializer, logging: Logging)
    extends ActivationStore {

  implicit val executionContext = actorSystem.dispatcher

  private val artifactStore: ArtifactStore[WhiskActivation] =
    WhiskActivationStore.datastore()(actorSystem, logging, actorMaterializer)

  def store(activation: WhiskActivation)(implicit transid: TransactionId,
                                         notifier: Option[CacheChangeNotification]): Future[DocInfo] = {

    logging.debug(this, s"recording activation '${activation.activationId}'")

    val res = WhiskActivation.put(artifactStore, activation)

    res onComplete {
      case Success(id) => logging.debug(this, s"recorded activation")
      case Failure(t) =>
        logging.error(
          this,
          s"failed to record activation ${activation.activationId} with error ${t.getLocalizedMessage}")
    }

    res
  }

  def get(activationId: ActivationId)(implicit transid: TransactionId): Future[WhiskActivation] = {
    WhiskActivation.get(artifactStore, DocId(activationId.asString))
  }

  def delete(activationId: ActivationId)(implicit transid: TransactionId,
                                         notifier: Option[CacheChangeNotification]): Future[Boolean] = {
    WhiskActivation.get(artifactStore, DocId(activationId.asString)) flatMap { doc =>
      WhiskActivation.del(artifactStore, doc.docinfo)
    }
  }

  def countActivationsInNamespace(name: Option[EntityPath] = None,
                                  namespace: EntityPath,
                                  skip: Int,
                                  since: Option[Instant] = None,
                                  upto: Option[Instant] = None)(implicit transid: TransactionId): Future[JsObject] = {
    WhiskActivation.countCollectionInNamespace(
      artifactStore,
      name.map(p => namespace.addPath(p)).getOrElse(namespace),
      skip,
      since,
      upto,
      StaleParameter.UpdateAfter,
      name.map(_ => WhiskActivation.filtersView).getOrElse(WhiskActivation.view))
  }

  def listActivationsMatchingName(namespace: EntityPath,
                                  path: EntityPath,
                                  skip: Int,
                                  limit: Int,
                                  includeDocs: Boolean = false,
                                  since: Option[Instant] = None,
                                  upto: Option[Instant] = None)(
    implicit transid: TransactionId): Future[Either[List[JsObject], List[WhiskActivation]]] = {
    WhiskActivation.listActivationsMatchingName(
      artifactStore,
      namespace,
      path,
      skip,
      limit,
      includeDocs,
      since,
      upto,
      StaleParameter.UpdateAfter)
  }

  def listActivationsInNamespace(path: EntityPath,
                                 skip: Int,
                                 limit: Int,
                                 includeDocs: Boolean = false,
                                 since: Option[Instant] = None,
                                 upto: Option[Instant] = None)(
    implicit transid: TransactionId): Future[Either[List[JsObject], List[WhiskActivation]]] = {
    WhiskActivation.listCollectionInNamespace(
      artifactStore,
      path,
      skip,
      limit,
      includeDocs,
      since,
      upto,
      StaleParameter.UpdateAfter)
  }

}

object ArtifactActivationStoreProvider extends ActivationStoreProvider {
  override def instance(actorSystem: ActorSystem, actorMaterializer: ActorMaterializer, logging: Logging) =
    new ArtifactActivationStore(actorSystem, actorMaterializer, logging)
}
