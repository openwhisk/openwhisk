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

package org.apache.openwhisk.core.containerpool.kubernetes

import java.io.ByteArrayInputStream
import java.nio.charset.StandardCharsets

import io.fabric8.kubernetes.api.builder.Predicate
import io.fabric8.kubernetes.api.model.{AffinityBuilder, ContainerBuilder, EnvVarBuilder, Pod, PodBuilder, Quantity}
import io.fabric8.kubernetes.client.NamespacedKubernetesClient
import org.apache.openwhisk.common.TransactionId
import org.apache.openwhisk.core.entity.ByteSize

import scala.collection.JavaConverters._

class WhiskPodBuilder(client: NamespacedKubernetesClient,
                      userPodNodeAffinity: KubernetesInvokerNodeAffinity,
                      podTemplate: Option[String] = None) {
  private val actionContainerName = "user-action"
  private val actionContainerPredicate: Predicate[ContainerBuilder] = (cb) => cb.getName == actionContainerName

  def buildPodSpec(name: String,
                   image: String,
                   memory: ByteSize,
                   environment: Map[String, String],
                   labels: Map[String, String])(implicit transid: TransactionId): Pod = {
    val envVars = environment.map {
      case (key, value) => new EnvVarBuilder().withName(key).withValue(value).build()
    }.toSeq

    val baseBuilder = podTemplate match {
      case Some(template) =>
        new PodBuilder(loadPodSpec(template))
      case None => new PodBuilder()
    }

    val pb1 = baseBuilder
      .editOrNewMetadata()
      .withName(name)
      .addToLabels("name", name)
      .addToLabels(labels.asJava)
      .endMetadata()

    val specBuilder = pb1.editOrNewSpec().withRestartPolicy("Always")

    if (userPodNodeAffinity.enabled) {
      //TODO enable updating affinity
      val invokerNodeAffinity = new AffinityBuilder()
        .withNewNodeAffinity()
        .withNewRequiredDuringSchedulingIgnoredDuringExecution()
        .addNewNodeSelectorTerm()
        .addNewMatchExpression()
        .withKey(userPodNodeAffinity.key)
        .withOperator("In")
        .withValues(userPodNodeAffinity.value)
        .endMatchExpression()
        .endNodeSelectorTerm()
        .endRequiredDuringSchedulingIgnoredDuringExecution()
        .endNodeAffinity()
        .build()
      specBuilder.withAffinity(invokerNodeAffinity)
    }

    val containerBuilder = if (specBuilder.hasMatchingContainer(actionContainerPredicate)) {
      specBuilder.editMatchingContainer(actionContainerPredicate)
    } else specBuilder.addNewContainer()

    //In container its assumed that env, port, resource limits are set explicitly
    //Here if any value exist in template then that would be overridden
    containerBuilder
      .withNewResources()
      .withLimits(Map("memory" -> new Quantity(memory.toMB + "Mi")).asJava)
      .endResources()
      .withName("user-action")
      .withImage(image)
      .withEnv(envVars.asJava)
      .addNewPort()
      .withContainerPort(8080)
      .withName("action")
      .endPort()

    //If any existing context entry is present then "update" it else add new
    containerBuilder
      .editOrNewSecurityContext()
      .editOrNewCapabilities()
      .addToDrop("NET_RAW", "NET_ADMIN")
      .endCapabilities()
      .endSecurityContext()

    val pod = containerBuilder
      .endContainer()
      .endSpec()
      .build()
    pod
  }

  private def loadPodSpec(spec: String): Pod = {
    val resources = client.load(new ByteArrayInputStream(spec.getBytes(StandardCharsets.UTF_8)))
    resources.get().get(0).asInstanceOf[Pod]
  }
}
