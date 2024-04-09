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
    plugins?: Plugin[]
  }
  setClusterDetails: (cluster: Cluster | null) => void
  setClusterDetailPlugins: (plugins: Plugin[]) => void
  showClusterDetails: string | undefined
  setShowClusterDetails: (showClusterDetails: string | undefined) => void
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

  showClusterDetails: undefined,
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
