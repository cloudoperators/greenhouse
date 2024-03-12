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
