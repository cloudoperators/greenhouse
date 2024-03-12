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

import { create } from "zustand"
import { getResourceStatusFromKubernetesConditions } from "./lib/utils/resourceStatus"
import {
  Cluster,
  PluginConfig,
  ResourceStatus,
  UpdateClusterAction,
  UpdateClusterInput,
} from "./types/types"

export interface State {
  endpoint: string
  setEndpoint: (newEndpoint: string) => void
  urlStateKey: string
  setUrlStateKey: (newUrlStateKey: string) => void

  auth: any
  setAuth: (auth: any) => void
  loggedIn: boolean
  setLoggedIn: (loggedIn: boolean) => void
  logout: any

  clusters: Cluster[]
  updateClusters: (input: UpdateClusterInput) => void

  clusterDetails: {
    cluster: Cluster | null
    clusterStatus: ResourceStatus | null
    pluginConfigs?: PluginConfig[]
  }
  setClusterDetails: (cluster: Cluster | null) => void
  setClusterDetailPluginConfigs: (pluginConfigs: PluginConfig[]) => void
  showClusterDetails: boolean
  setShowClusterDetails: (showClusterDetails: boolean) => void
  showOnBoardCluster: boolean
  setShowOnBoardCluster: (showOnBoardCluster: boolean) => void
  showDownloadKubeConfig: boolean
  setShowDownloadKubeConfig: (showDownloadKubeConfig: boolean) => void
}

// global zustand store. See how this works here: https://github.com/pmndrs/zustand
const useStore = create<State>((set) => ({
  endpoint: "",
  setEndpoint: (newEndpoint) => set((state) => ({ endpoint: newEndpoint })),
  urlStateKey: "",
  setUrlStateKey: (newUrlStateKey) =>
    set((state) => ({ urlStateKey: newUrlStateKey })),

  auth: null,
  setAuth: (auth) => set((state) => ({ auth: auth })),
  loggedIn: false,
  setLoggedIn: (loggedIn) => set((state) => ({ loggedIn: loggedIn })),
  logout: null,

  clusters: [],
  updateClusters: (input: UpdateClusterInput) =>
    set((state) => {
      let clusters = [...state.clusters]
      // validate clusters: only accept input.clusters that have metadata.name set
      input.clusters = input.clusters.filter((cluster) => {
        return cluster.metadata?.name ?? undefined !== undefined
      })

      if (input.action === UpdateClusterAction.delete) {
        clusters = clusters.filter((knownCluster) => {
          return input.clusters.some((inputCluster) => {
            return knownCluster.metadata!.name !== inputCluster.metadata!.name
          })
        })
        return { ...state, clusters: clusters }
      }

      input.clusters.forEach((inputCluster) => {
        // replace existing cluster with new one or add new cluster
        const index = clusters.findIndex((knownCluster) => {
          return knownCluster.metadata!.name === inputCluster.metadata!.name
        })
        if (index >= 0) {
          clusters[index] = inputCluster
        } else {
          clusters.push(inputCluster)
        }
      })
      return { ...state, clusters: clusters }
    }),

  clusterDetails: {
    cluster: null,
    clusterStatus: null,
    pluginConfigs: [],
  },
  setClusterDetails: (cluster: Cluster | null) =>
    set((state) => {
      if (!cluster) {
        return {
          ...state,
          clusterDetails: {
            cluster: null,
            clusterStatus: null,
            pluginConfigs: [],
          },
        }
      }
      let clusterStatus: ResourceStatus =
        getResourceStatusFromKubernetesConditions(
          cluster.status?.statusConditions?.conditions ?? []
        )

      return {
        ...state,
        clusterDetails: {
          cluster: cluster,
          clusterStatus: clusterStatus,
          pluginConfigs: [],
        },
      }
    }),

  setClusterDetailPluginConfigs: (pluginConfigs: PluginConfig[]) =>
    set((state) => {
      return {
        ...state,
        clusterDetails: {
          ...state.clusterDetails,
          pluginConfigs: pluginConfigs,
        },
      }
    }),

  showClusterDetails: false,
  setShowClusterDetails: (showClusterDetails) =>
    set((state) => ({ ...state, showClusterDetails: showClusterDetails })),
  showOnBoardCluster: false,
  setShowOnBoardCluster: (showOnBoardCluster) =>
    set((state) => ({ ...state, showOnBoardCluster: showOnBoardCluster })),

  showDownloadKubeConfig: false,
  setShowDownloadKubeConfig: (showDownloadKubeConfig) => {
    set((state) => ({
      ...state,
      showDownloadKubeConfig: showDownloadKubeConfig,
    }))
  },
}))

export default useStore
