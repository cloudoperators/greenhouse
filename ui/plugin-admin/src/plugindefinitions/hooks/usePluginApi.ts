
import useApi, { ApiResponse } from "./useApi"
import { Plugin } from "../../../../types/types"
import useNamespace from "./useNamespace"
import useClient from "./useClient"
import { useCallback } from "react"

export const usePluginApi = () => {
  const {get, create,  update, deleteObject } = useApi()
  const { namespace } = useNamespace()
  const {client} = useClient()

  const getPlugin= (plugin: Plugin): Promise<ApiResponse> => {
    return get<Plugin>(`/api/v1/namespaces/${namespace}/plugins`, plugin)
  }

  const createPlugin = (plugin: Plugin): Promise<ApiResponse> => {
    return create<Plugin>(`/api/v1/namespaces/${namespace}/plugins`, plugin)
  }

  const updatePlugin = (plugin: Plugin): Promise<ApiResponse> => {
    return update<Plugin>(`/api/v1/namespaces/${namespace}/plugins`, plugin)
  }

  const deletePlugin = (plugin: Plugin): Promise<ApiResponse> => {
    return deleteObject <Plugin>(`/api/v1/namespaces/${namespace}/plugins`, plugin)
  }

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

  return { getPlugin, createPlugin, updatePlugin, deletePlugin, getPluginsByLabelSelector }
}

export default usePluginApi