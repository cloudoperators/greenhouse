/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import useApi from "./useApi"
import { Plugin } from "../../../../types/types"
import useNamespace from "./useNamespace"
import useClient from "./useClient"
import { useCallback } from "react"

export type PluginApiResponse = {
  ok: boolean
  message: string
  response?: Plugin
}

export const usePluginApi = () => {
  const { get, create, update, deleteObject } = useApi()
  const { namespace } = useNamespace()
  const { client } = useClient()

  const getPlugin = (plugin: Plugin): Promise<PluginApiResponse> => {
    return get<Plugin>(
      `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/plugins`,
      plugin
    ) as Promise<PluginApiResponse>
  }

  const createPlugin = (plugin: Plugin): Promise<PluginApiResponse> => {
    return create<Plugin>(
      `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/plugins`,
      plugin
    ) as Promise<PluginApiResponse>
  }

  const updatePlugin = (plugin: Plugin): Promise<PluginApiResponse> => {
    return update<Plugin>(
      `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/plugins`,
      plugin
    ) as Promise<PluginApiResponse>
  }

  const deletePlugin = (plugin: Plugin): Promise<PluginApiResponse> => {
    return deleteObject<Plugin>(
      `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/plugins`,
      plugin
    ) as Promise<PluginApiResponse>
  }

  const getPluginsByLabelSelector = useCallback(
    async (labelSelectorKey, labelSelectorValue: string): Promise<Plugin[]> => {
      let plugins: Plugin[] = []

      if (!client || !namespace) {
        return plugins
      }
      const labelselector = `${labelSelectorKey}=${labelSelectorValue}`

      plugins = await client
        .get(`/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/plugins`, {
          params: {
            labelSelector: labelselector,
          },
        })
        .then((res) => {
          if (res.kind !== "PluginList") {
            console.log("ERROR: Failed to get Plugins, did not get PluginList")
            return [] as Plugin[]
          }
          return res.items as Plugin[]
        })

      return plugins
    },
    [client, namespace]
  )

  return {
    getPlugin,
    createPlugin,
    updatePlugin,
    deletePlugin,
    getPluginsByLabelSelector,
  }
}

export default usePluginApi
