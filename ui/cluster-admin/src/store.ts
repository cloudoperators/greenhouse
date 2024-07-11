/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { create } from "zustand"
import { getResourceStatusFromKubernetesConditions } from "./lib/utils/resourceStatus"
import {
  Cluster,
  Plugin,
  ResourceStatus,
  UpdateObjectAction,
  UpdateClusterInput,
} from "../../types/types"

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
  modifyClusters: (clusters: Cluster[]) => void
  deleteClusters: (clusters: Cluster[]) => void

  clusterDetails: {
    cluster: Cluster | null
    clusterStatus: ResourceStatus | null
    plugins?: Plugin[]
  }
  setClusterDetails: (cluster: Cluster | null) => void
  setClusterDetailPlugins: (plugins: Plugin[]) => void
  showClusterDetails: boolean
  setShowClusterDetails: (showClusterDetails: boolean) => void
  showOnBoardCluster: boolean
  setShowOnBoardCluster: (showOnBoardCluster: boolean) => void
  showDownloadKubeConfig: boolean
  setShowDownloadKubeConfig: (showDownloadKubeConfig: boolean) => void

  clusterInEdit?: Cluster
  setClusterInEdit: (cluster?: Cluster) => void
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
  modifyClusters: (clusters: Cluster[]) =>
    set((state) => {
      let newClusters = [...state.clusters]
      clusters.forEach((inputCluster) => {
        const index = newClusters.findIndex((knownCluster) => {
          return knownCluster.metadata!.name === inputCluster.metadata!.name
        })
        if (index >= 0) {
          newClusters[index] = inputCluster
        } else {
          newClusters.push(inputCluster)
        }
      })
      return { ...state, clusters: newClusters }
    }),
  deleteClusters: (clusters: Cluster[]) =>
    set((state) => {
      const newClusters = state.clusters.filter((knownCluster) => {
        return !clusters.some((inputCluster) => {
          return knownCluster.metadata!.name === inputCluster.metadata!.name
        })
      })
      return { clusters: newClusters }
    }),

  clusterDetails: {
    cluster: null,
    clusterStatus: null,
    plugins: [],
  },
  setClusterDetails: (cluster: Cluster | null) =>
    set((state) => {
      if (!cluster) {
        return {
          ...state,
          clusterDetails: {
            cluster: null,
            clusterStatus: null,
            plugins: [],
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
          plugins: [],
        },
      }
    }),

  setClusterDetailPlugins: (plugins: Plugin[]) =>
    set((state) => {
      return {
        ...state,
        clusterDetails: {
          ...state.clusterDetails,
          plugins: plugins,
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

  clusterInEdit: undefined,
  setClusterInEdit: (cluster) => set((state) => ({ clusterInEdit: cluster })),
}))

export default useStore
