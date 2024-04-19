/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { useCallback } from "react"
import { Plugin } from "../../../../types/types"
import useClient from "./useClient"
import useNamespace from "./useNamespace"

export enum ResponseFailReasons {
  RESOURCE_VERSION_MISMATCH = "resourceVersionMismatch",
  UNKOWN = "unkown",
}

export type PluginApiResponse = {
  ok: boolean,
  message: string
  reason?: ResponseFailReasons
  plugin?: Plugin
}

export const usePluginApi = () => {
  const { namespace } = useNamespace()
  const { client: client } = useClient()

  // getPluginsByLabelSelector returns a list of plugins that match the given label selector
  const getPluginsByLabelSelector = useCallback(
    async (labelSelectorKey, labelSelectorValue: string): Promise<Plugin[]> => {
      let plugins: Plugin[] = []

      if (!client || !namespace) {
        return plugins
      }
      const labelselector = `${labelSelectorKey}=${
        labelSelectorValue
      }`

      plugins = await client
        .get(
          `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/plugins`,
          {
            params: {
              labelSelector: labelselector,
            },
          }
        )
        .then((res) => {
          if (res.kind !== "PluginList") {
            console.log(
              "ERROR: Failed to get Plugins, did not get PluginList"
            )
            return [] as Plugin[]
          }
          return res.items as Plugin[]
        })

      return plugins
    },
    [client, namespace]
  )

  // getPlugin returns a single plugin by name
  const getPlugin = useCallback(
    async (pluginName: string): Promise<PluginApiResponse> => {
      if (!client || !namespace) {
        return { ok: false, message: "Client or namespace not available" }
      }

      return await client
        .get(
          `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/plugins/${pluginName}`
        )
        .then((res) => {
          if (res.kind !== "Plugin") {
            return { ok: false, message: "Failed getting plugin" }
          }
          return { ok: true, message: "Success getting plugin", plugin: res }
        })
        .catch((err) => {
          return { ok: false, message: "Failed getting plugin: " + err.message }
        })
    },
    [client, namespace]
  )

  // postPlugin creates a new plugin
  const postPlugin = async (plugin: Plugin): Promise<PluginApiResponse> => {
    if (!client || !namespace) {
      return { ok: false, message: "Client or namespace not available" }
    }

    return client
      .post(
        `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/plugins/${plugin.metadata!.name}`,
        { ...plugin }
      )
      .then((res) => {
        if (res.kind !== "Plugin") {
          return { ok: false, message: "Failed creating plugin: " + JSON.stringify(res) }
        }
        return { ok: true, message: "Success creating plugin", plugin: res }
      })
      .catch((err) => {
        return { ok: false, message: "Failed creating plugin: " + err.message }
      })
  }

  // updatePlugin updates an existing plugin
  const updatePlugin = async (plugin: Plugin): Promise<PluginApiResponse> => {
    if (!client || !namespace) {
      return { ok: false, message: "Client or namespace not available" }
    }

    return client
      .put(
        `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/plugins/${plugin.metadata!.name}`,
        { ...plugin }
      )
      .then((res) => {
        if (res.kind !== "Plugin") {
          return { ok: false, message: "Failed updating plugin: " + JSON.stringify(res) }
        }
        return { ok: true, message: "Success updating plugin", plugin: res }
      })
      .catch((err) => {
        return { ok: false, message: "Failed updating plugin: " + err.message }
      })
  }


  return {
    getPluginsByLabelSelector: getPluginsByLabelSelector,
    getPlugin: getPlugin,
    postPlugin: postPlugin,
    updatePlugin: updatePlugin,
  }
}

export default usePluginApi
