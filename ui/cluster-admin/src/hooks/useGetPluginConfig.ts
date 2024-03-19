/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { useCallback } from "react"
import { Cluster, PluginConfig } from "../types/types"
import useClient from "./useClient"
import useNamespace from "./useNamespace"

export const useGetPluginConfigs = () => {
  const { namespace } = useNamespace()
  const { client: client } = useClient()
  const getPluginConfigsforCluster = useCallback(
    async (cluster: Cluster): Promise<PluginConfig[]> => {
      let pluginConfigs: PluginConfig[] = []
      const greenhouseClusterLabelKey = "greenhouse.sap/cluster"

      if (!client || !namespace || !cluster?.metadata?.namespace) {
        return pluginConfigs
      }
      const labelselector = `${greenhouseClusterLabelKey}=${
        cluster.metadata!.name
      }`

      pluginConfigs = await client
        .get(
          `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/pluginconfigs`,
          {
            params: {
              labelSelector: labelselector,
            },
          }
        )
        .then((res) => {
          if (res.kind !== "PluginConfigList") {
            console.log(
              "ERROR: Failed to get PluginConfigs for cluster, did not get PluginConfigList"
            )
            return [] as PluginConfig[]
          }
          return res.items as PluginConfig[]
        })

      return pluginConfigs
    },
    [client, namespace]
  )

  return {
    getPluginConfigsforCluster: getPluginConfigsforCluster,
  }
}

export default useGetPluginConfigs
