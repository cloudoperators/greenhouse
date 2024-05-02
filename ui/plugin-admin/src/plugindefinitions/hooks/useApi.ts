import { useCallback } from "react"
import { Secret, Plugin, Cluster, PluginDefinition } from "../../../../types/types"
import useClient from "./useClient"
import useNamespace from "./useNamespace"

export type AllowedApiObject = Plugin | Cluster | Secret | PluginDefinition

export type ApiResponse = {
  ok: boolean,
  message: string
  response?: AllowedApiObject
}

export const useApi = () => {
  const { namespace } = useNamespace()
  const { client: client } = useClient()

const get = useCallback(
  async <T extends AllowedApiObject>(url: string, object: T): Promise<ApiResponse> => { 
  let response: T

  if (!client || !namespace) {
    return { ok: false, message: "Client or namespace not available" }
  }

  return await client
    .get(
      url+"/"+object.metadata!.name!,
    )
    .then((res) => {
      if (res.kind !== object.kind) {
        console.log(
          `ERROR: Failed to get ${object.kind}, did not get ${object.kind}`
        )
        return {ok: false, message: `Failed getting ${object.kind}`}
      }
      return {ok: true, response: res as T, message: `Successfully got ${object.kind}`}
    })
    .catch((error) => {
      console.log(`ERROR: Failed to get ${object.kind}`, error)
      return {ok: false, message: `Failed getting ${object.kind}: ${error}`}
    })
  }, [client, namespace]
  )

  const create = useCallback(
    async <T extends AllowedApiObject>(url: string, object: T): Promise<ApiResponse> => {
      if (!client || !namespace) {
        return { ok: false, message: "Client or namespace not available" }
      }

      return await client
        .post(
          url+"/"+object.metadata!.name!,
          object
        )
        .then((res) => {
          if (res.kind !== object.kind) {
            console.log(
              `ERROR: Failed to create ${object.kind}, did not get ${object.kind}`
            )
            return {ok: false, message: `Failed creating ${object.kind}`}
          }
          return {ok: true, response: res as T, message: `Successfully created ${object.kind}`}
        })
        .catch((error) => {
          console.log(`ERROR: Failed to create ${object.kind}`, error)
          return {ok: false, message: `Failed creating ${object.kind}: ${error}`}
        })
    }, [client, namespace]
  )

  const update = useCallback(
    async <T extends AllowedApiObject>(url: string, object: T): Promise<ApiResponse> => {
      if (!client || !namespace) {
        return { ok: false, message: "Client or namespace not available" }
      }

      return await client
        .put(
          url+"/"+object.metadata!.name!,
          object
        )
        .then((res) => {
          if (res.kind !== object.kind) {
            console.log(
              `ERROR: Failed to update ${object.kind}, did not get ${object.kind}`
            )
            return {ok: false, message: `Failed updating ${object.kind}`}
          }
          return {ok: true, response: res as T, message: `Successfully updated ${object.kind}`}
        })
        .catch((error) => {
          console.log(`ERROR: Failed to update ${object.kind}`, error)
          return {ok: false, message: `Failed updating ${object.kind}: ${error}`}
        })
    }, [client, namespace]
  )

  const deleteObject = useCallback(
    async <T extends AllowedApiObject>(url: string, object: T): Promise<ApiResponse> => {
      if (!client || !namespace) {
        return { ok: false, message: "Client or namespace not available" }
      }

      return await client
        .delete(
          url+"/"+object.metadata!.name!,
        )
        .then((res) => {
          if((res.kind == "Plugin") || (res.kind == "Status" && res.status == "Success")){
            return {ok: true, message: `Successfully deleted ${object.kind}`}
          }
          console.log(
            `ERROR: Failed to delete ${object.kind}, did not get ${object.kind}`
          )
          return {ok: false, message: `Failed deleting ${object.kind}`}
        })
        .catch((error) => {
          console.log(`ERROR: Failed to delete ${object.kind}`, error)
          return {ok: false, message: `Failed deleting ${object.kind}: ${error}`}
        })
    }, [client, namespace]
  )

  return { 
    get: get ,
    create: create,
    update: update,
    deleteObject: deleteObject
  }
}

export default useApi
