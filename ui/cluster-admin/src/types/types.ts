/*
 * Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
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
