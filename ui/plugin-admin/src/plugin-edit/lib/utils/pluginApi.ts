import { Plugin } from "../../../../../types/types"

export type PluginApiResponse = {
  ok: boolean,
  message: string
  plugin?: Plugin
}

export const postPlugin = async (plugin: Plugin, namespace:String, client:any):Promise<PluginApiResponse> => {

  return client
      .post(
        `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/plugins/${
          plugin.metadata!.name
        }`,
        { ...plugin }
      ).then((res) => {
        if (res.kind == "Plugin") {
          return { ok:true, message: "Success creating plugin", plugin: res }
        }
        return { ok: false, message: "Failed creating plugin: "+JSON.stringify(res) }
      })
      .catch((err) => {
        return { ok: false, message: "Failed creating plugin: "+err.message }
      })
    
}

export const updatePlugin = async (plugin: Plugin, namespace:String, client:any):Promise<PluginApiResponse> => {
  
    return await client
        .put(
          `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/plugins/${
            plugin.metadata!.name
          }`,
          { ...plugin }
        ).then((res) => {
          if (res.kind == "Plugin") {
            return { ok:true, message: "Success updating plugin", plugin: res }
          }
          return { ok: false, message: "Failed updating plugin: "+JSON.stringify(res) }
        })
        .catch((err) => {
          return { ok: false, message: "Failed updating plugin: "+err.message }
        })
  }

  export const getPlugin = async (plugin: Plugin,namespace:String, client:any):Promise<PluginApiResponse> => {
    
      return await client
          .get(
            `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/plugins/${
              plugin.metadata!.name
            }`,
          ).then((res) => {
            if (res.kind == "Plugin") {
              return { ok:true, message: "Success getting plugins", plugin: res }
            }
            return { ok: false, message: "Failed getting plugins: "+JSON.stringify(res) }
          })
          .catch((err) => {
            return { ok: false, message: "Failed getting plugins: "+err.message }
          })
    }