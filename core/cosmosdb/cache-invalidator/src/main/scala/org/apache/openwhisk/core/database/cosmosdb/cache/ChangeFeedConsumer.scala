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

package org.apache.openwhisk.core.database.cosmosdb.cache

import java.util

import akka.Done
import akka.event.slf4j.SLF4JLogging
import com.azure.data.cosmos.internal.changefeed.implementation.ChangeFeedProcessorBuilderImpl
import com.azure.data.cosmos.internal.changefeed.{ChangeFeedObserverCloseReason, ChangeFeedObserverContext}
import com.azure.data.cosmos.{
  ChangeFeedProcessor,
  ChangeFeedProcessorOptions,
  ConnectionPolicy,
  CosmosClient,
  CosmosContainer,
  CosmosItemProperties
}
import reactor.core.publisher.Mono

import scala.collection.JavaConverters._
import scala.collection.immutable.Seq
import scala.compat.java8.FutureConverters._
import scala.concurrent.{ExecutionContext, Future}

trait ChangeFeedObserver {
  def process(context: ChangeFeedObserverContext, docs: Seq[CosmosItemProperties]): Future[Done]
}

class ChangeFeedConsumer(collName: String, config: CacheInvalidatorConfig, observer: ChangeFeedObserver)(
  implicit ec: ExecutionContext)
    extends SLF4JLogging {
  import ChangeFeedConsumer._

  log.info(s"Watching changes in $collName with lease managed in ${config.feedConfig.leaseCollection}")

  private val clients = scala.collection.mutable.Map[ConnectionInfo, CosmosClient]().withDefault(createCosmosClient)
  private val targetContainer = getContainer(collName)
  private val leaseContainer = getContainer(config.feedConfig.leaseCollection, createIfNotExist = true)
  private val (processor, startFuture) = {
    val clusterId = config.invalidatorConfig.clusterId
    val prefix = clusterId.map(id => s"$id-$collName").getOrElse(collName)

    val feedOpts = new ChangeFeedProcessorOptions
    feedOpts.leasePrefix(prefix)

    val builder = ChangeFeedProcessor.Builder
      .hostName(config.feedConfig.hostname)
      .feedContainer(targetContainer)
      .leaseContainer(leaseContainer)
      .options(feedOpts)
      .asInstanceOf[ChangeFeedProcessorBuilderImpl] //observerFactory is not exposed hence need to cast to impl

    builder.observerFactory(() => ObserverBridge)
    val p = builder.build()
    (p, p.start().toFuture.toScala.map(_ => Done))
  }

  def isStarted: Future[Done] = startFuture

  def close(): Future[Done] = {
    val f = processor.stop().toFuture.toScala.map(_ => Done)
    f.andThen {
      case _ =>
        clients.values.foreach(c => c.close())
        Future.successful(Done)
    }
  }

  private def getContainer(name: String, createIfNotExist: Boolean = false): CosmosContainer = {
    val info = config.getCollectionInfo(name)
    val client = clients(info)
    val db = client.getDatabase(info.db)
    val container = db.getContainer(name)

    val resp = if (createIfNotExist) {
      db.createContainerIfNotExists(name, "/id", info.throughput)
    } else container.read()

    resp.block().container()
  }

  private object ObserverBridge extends com.azure.data.cosmos.internal.changefeed.ChangeFeedObserver {
    override def open(context: ChangeFeedObserverContext): Unit = {}
    override def close(context: ChangeFeedObserverContext, reason: ChangeFeedObserverCloseReason): Unit = {}
    override def processChanges(context: ChangeFeedObserverContext,
                                docs: util.List[CosmosItemProperties]): Mono[Void] = {
      val f = observer.process(context, docs.asScala.toList).map(_ => null).toJava.toCompletableFuture
      Mono.fromFuture(f)
    }
  }
}

object ChangeFeedConsumer {
  def createCosmosClient(conInfo: ConnectionInfo): CosmosClient = {
    val policy = ConnectionPolicy.defaultPolicy.connectionMode(conInfo.connectionMode)
    CosmosClient.builder
      .endpoint(conInfo.endpoint)
      .key(conInfo.key)
      .connectionPolicy(policy)
      .consistencyLevel(conInfo.consistencyLevel)
      .build
  }
}
