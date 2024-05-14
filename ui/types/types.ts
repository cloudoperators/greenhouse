/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { components } from "./schema"
import { Secret as k8sSecret } from "kubernetes-types/core/v1"

export type Secret = k8sSecret
export type Cluster = components["schemas"]["Cluster"]
export type PluginDefinition = components["schemas"]["PluginDefinition"]
export type Plugin = components["schemas"]["Plugin"]
export type PluginPreset = components["schemas"]["PluginPreset"]
export type UpdateClusterInput = {
  clusters: Cluster[]
  action: UpdateObjectAction
}
export type UpdatePluginDefinitionInput = {
  pluginDefinitions: PluginDefinition[]
  action: UpdateObjectAction
}
export type UpdateSecretInput = {
  secrets: Secret[]
  action: UpdateObjectAction
}
export enum UpdateObjectAction {
  "add",
  "delete",
}

export type ResourceStatus = {
  state: string
  color: string
  icon: string
  message?: string
}

export enum ResourceStatusCondition {
  "ready",
  "unkown",
  "notReady",
}

export type KubernetesCondition = {
  type: string
  status: string
  message?: string
  lastTransitionTime?: string
}

export type PluginDefinitionOptions = NonNullable<
  PluginDefinition["spec"]
>["options"]
export type PluginDefinitionOption =
  NonNullable<PluginDefinitionOptions>[number]
export type PluginOptionValues = NonNullable<Plugin["spec"]>["optionValues"]
export type PluginOptionValue = NonNullable<PluginOptionValues>[number]
export type PluginOptionValueFrom = NonNullable<PluginOptionValue>["valueFrom"]

export type SecretDataEntry = NonNullable<Secret["data"]>
