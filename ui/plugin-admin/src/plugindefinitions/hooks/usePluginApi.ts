/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { useCallback } from "react"
import { Plugin } from "../../../../types/types"
import useClient from "./useClient"
import useNamespace from "./useNamespace"


export type PluginApiResponse = {
  ok: boolean,
  message: string
  plugin?: Plugin
}

// ENUM for Plugin API response message strings
enum PluginApiResponseMessage {
  CLIENT_OR_NAMESPACE_NOT_AVAILABLE = "Client or namespace not available",
  FAILED_GETTING_PLUGIN = "Failed getting Plugin",
  FAILED_CREATING_PLUGIN = "Failed creating Plugin",
  FAILED_UPDATING_PLUGIN = "Failed updating Plugin",
  FAILED_DELETING_PLUGIN = "Failed deleting Plugin",
  SUCCESS_GETTING_PLUGIN = "Successfully got Plugin",
  SUCCESSFULLY_CREATED_PLUGIN = "Successfully created Plugin",
  SUCCESSFULLY_UPDATED_PLUGIN = "Successfully updated Plugin",
  SUCCESSFULLY_DELETED_PLUGIN = "Successfully deleted Plugin",
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
        return { ok: false, message: PluginApiResponseMessage.CLIENT_OR_NAMESPACE_NOT_AVAILABLE }
      }

      return await client
        .get(
          `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/plugins/${pluginName}`
        )
        .then((res) => {
          if (res.kind !== "Plugin") {
            return { ok: false, message: PluginApiResponseMessage.FAILED_GETTING_PLUGIN + ": " + JSON.stringify(res) }
          }
          return { ok: true, message: PluginApiResponseMessage.SUCCESS_GETTING_PLUGIN, plugin: res }
        })
        .catch((err) => {
          return { ok: false, message: PluginApiResponseMessage.FAILED_GETTING_PLUGIN + ": " + err.message }
        })
    },
    [client, namespace]
  )

  // postPlugin creates a new plugin
  const postPlugin = async (plugin: Plugin): Promise<PluginApiResponse> => {
    if (!client || !namespace) {
      return { ok: false, message: PluginApiResponseMessage.CLIENT_OR_NAMESPACE_NOT_AVAILABLE }
    }

    return client
      .post(
        `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/plugins/${plugin.metadata!.name}`,
        { ...plugin }
      )
      .then((res) => {
        if (res.kind !== "Plugin") {
          return { ok: false, message: PluginApiResponseMessage.FAILED_CREATING_PLUGIN+": " + JSON.stringify(res) }
        }
        return { ok: true, message: PluginApiResponseMessage.SUCCESSFULLY_CREATED_PLUGIN, plugin: res }
      })
      .catch((err) => {
        return { ok: false, message: PluginApiResponseMessage.FAILED_CREATING_PLUGIN+": " + err.message }
      })
  }

  // updatePlugin updates an existing plugin
  const updatePlugin = async (plugin: Plugin): Promise<PluginApiResponse> => {
    if (!client || !namespace) {
      return { ok: false, message: PluginApiResponseMessage.CLIENT_OR_NAMESPACE_NOT_AVAILABLE }
    }

    return client
      .put(
        `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/plugins/${plugin.metadata!.name}`,
        { ...plugin }
      )
      .then((res) => {
        if (res.kind !== "Plugin") {
          return { ok: false, message: PluginApiResponseMessage.FAILED_UPDATING_PLUGIN+ ": " + JSON.stringify(res) }
        }
        return { ok: true, message: PluginApiResponseMessage.SUCCESSFULLY_UPDATED_PLUGIN, plugin: res }
      })
      .catch((err) => {
        return { ok: false, message: PluginApiResponseMessage.FAILED_UPDATING_PLUGIN+": " + err.message }
      })
  }

  // deletePlugin deletes an existing plugin
  // Attention: ambiguous response type from api server
  // See: https://github.com/kubernetes/kubernetes/issues/59501
  const deletePlugin = async (pluginName: string): Promise<PluginApiResponse> => {
    if (!client || !namespace) {
      return { ok: false, message: PluginApiResponseMessage.CLIENT_OR_NAMESPACE_NOT_AVAILABLE }
    }

    return client
      .delete(
        `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/plugins/${pluginName}`
      )
      .then((res) => {
        if((res.kind == "Plugin") || (res.kind == "Status" && res.status == "Success")){
          return { ok: true, message: PluginApiResponseMessage.SUCCESSFULLY_DELETED_PLUGIN }
        }
        else{
          return { ok: false, message: PluginApiResponseMessage.FAILED_DELETING_PLUGIN+": " + JSON.stringify(res) }
        }
      })
      .catch((err) => {
        return { ok: false, message: PluginApiResponseMessage.FAILED_DELETING_PLUGIN+": " + err.message }
      })
  }


  return {
    getPluginsByLabelSelector: getPluginsByLabelSelector,
    getPlugin: getPlugin,
    postPlugin: postPlugin,
    updatePlugin: updatePlugin,
    deletePlugin: deletePlugin,
  }
}

export default usePluginApi
