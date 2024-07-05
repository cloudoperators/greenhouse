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

export type AllowedApiObject =
  | Plugin
  | Cluster
  | Secret
  | PluginDefinition
  | PluginPreset

export type AllowedApiObjectKind =
  | "Plugin"
  | "Cluster"
  | "Secret"
  | "PluginDefinition"
  | "PluginPreset"

/**
 * ApiResponse
 * @description ApiResponse object is used to return the result of an k8s API call through our client wrapper methods.
 * We intent to simplify and wrap the fetch api response object: https://developer.mozilla.org/en-US/docs/Web/API/Response
 * and add some convenience properties.
 *
 */
export type ApiResponse = {
  /** @description Overall ok of the response, true indicates no error happened */
  ok: boolean
  /** @description Message of the response */
  message: string
  /** @description The response object */
  response?: AllowedApiObject
  /** @description The status of the response */
  status?: number
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

export type ResultMessage = {
  message?: string
  ok: boolean
}

// some subtypes
export type PluginDefinitionOptions = NonNullable<
  PluginDefinition["spec"]
>["options"]
export type PluginDefinitionOption =
  NonNullable<PluginDefinitionOptions>[number]
export type PluginOptionValues = NonNullable<Plugin["spec"]>["optionValues"]
export type PluginOptionValue = NonNullable<PluginOptionValues>[number]
export type PluginOptionValueFrom = NonNullable<PluginOptionValue>["valueFrom"]

export type PluginPresetSpec = NonNullable<PluginPreset["spec"]>
export type ClusterSelector = NonNullable<PluginPresetSpec["clusterSelector"]>
export type LabelSelector = NonNullable<ClusterSelector["matchLabels"]>

export type SecretDataEntry = NonNullable<Secret["data"]>
