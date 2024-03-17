/*
 * SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { components } from "./schema"

export type Cluster = components["schemas"]["Cluster"]
export type PluginConfig = components["schemas"]["PluginConfig"]
export type UpdateClusterInput = {
  clusters: Cluster[]
  action: UpdateClusterAction
}
export enum UpdateClusterAction {
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
