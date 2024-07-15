/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { useCallback } from "react"
import {
  AllowedApiObject,
  AllowedApiObjectKind,
  ApiResponse,
} from "../../../../types/types"
import useClient from "./useClient"
import useNamespace from "./useNamespace"

export const useApi = (debug?: boolean) => {
  const { namespace } = useNamespace()
  const { client: client } = useClient()

  // toggle verbosity
  const isDebug = debug ?? false

  const get = useCallback(
    async <T extends AllowedApiObject>(
      url: string,
      object: T,
      params?: any
    ): Promise<ApiResponse> => {
      let response: T

      if (!client || !namespace) {
        return { ok: false, message: "Client or namespace not available" }
      }

      return await client
        .get(url + "/" + object.metadata!.name!, {
          params: params,
        })
        .then((res) => {
          if (res.kind !== object.kind) {
            isDebug &&
              console.log(
                `ERROR: Failed to get ${object.kind}, did not get ${
                  object.kind
                }: ${JSON.stringify(res)}`
              )
            return {
              ok: false,
              message: `Failed getting ${object.kind}`,
              response: res as T,
            }
          }
          return {
            ok: true,
            response: res as T,
            message: `Successfully got ${object.kind}`,
          }
        })
        .catch((error) => {
          isDebug && console.log(`ERROR: Failed to get ${object.kind}`, error)
          return {
            ok: false,
            message: `Failed getting ${object.kind}: ${error}`,
          }
        })
    },
    [client, namespace]
  )

  const create = useCallback(
    async <T extends AllowedApiObject>(
      url: string,
      object: T
    ): Promise<ApiResponse> => {
      if (!client || !namespace) {
        return { ok: false, message: "Client or namespace not available" }
      }

      return await client
        .post(url, object)
        .then((res) => {
          if (res.kind !== object.kind) {
            isDebug &&
              console.log(
                `ERROR: Failed to create ${object.kind}, did not get ${
                  object.kind
                }: ${JSON.stringify(res)}`
              )
            return { ok: false, message: `Failed creating ${object.kind}` }
          }
          return {
            ok: true,
            response: res as T,
            message: `Successfully created ${object.kind}`,
          }
        })
        .catch((error) => {
          isDebug &&
            console.log(`ERROR: Failed to create ${object.kind}`, error)
          return {
            ok: false,
            message: `Failed creating ${object.kind}: ${error}`,
          }
        })
    },
    [client, namespace]
  )

  const update = useCallback(
    async <T extends AllowedApiObject>(
      url: string,
      object: T
    ): Promise<ApiResponse> => {
      if (!client || !namespace) {
        return { ok: false, message: "Client or namespace not available" }
      }

      return await client
        .put(url + "/" + object.metadata!.name!, object)
        .then((res) => {
          if (res.kind !== object.kind) {
            isDebug &&
              console.log(
                `ERROR: Failed to update ${object.kind}, did not get ${
                  object.kind
                }: ${JSON.stringify(res)}`
              )
            return { ok: false, message: `Failed updating ${object.kind}` }
          }
          return {
            ok: true,
            response: res as T,
            message: `Successfully updated ${object.kind}`,
          }
        })
        .catch((error) => {
          isDebug &&
            console.log(`ERROR: Failed to update ${object.kind}`, error)
          return {
            ok: false,
            message: `Failed updating ${object.kind}: ${error}`,
          }
        })
    },
    [client, namespace]
  )

  // "delete" is a reserved keyword in JavaScript, so we use "deleteObject" instead of "delete
  const deleteObject = useCallback(
    async <T extends AllowedApiObject>(
      url: string,
      object: T,
      params?: any
    ): Promise<ApiResponse> => {
      if (!client || !namespace) {
        return { ok: false, message: "Client or namespace not available" }
      }

      return await client
        .delete(url + "/" + object.metadata!.name!, {
          params: params,
        })
        .then((res) => {
          if (
            res.kind == object.kind ||
            (res.kind == "Status" && res.status == "Success")
          ) {
            return { ok: true, message: `Successfully deleted ${object.kind}` }
          }
          isDebug &&
            console.log(
              `ERROR: Failed to delete ${object.kind} did not get ${
                object.kind
              }: ${JSON.stringify(res)}`
            )
          return { ok: false, message: `Failed deleting ${object.kind}` }
        })
        .catch((error) => {
          isDebug &&
            console.log(`ERROR: Failed to delete ${object.kind}`, error)
          return {
            ok: false,
            message: `Failed deleting ${object.kind}: ${error}`,
          }
        })
    },
    [client, namespace]
  )

  const watch = useCallback(
    async <T extends AllowedApiObject>(
      url: string,
      kind: AllowedApiObjectKind,
      onAdded: (items: T[]) => void,
      onModified: (items: T[]) => void,
      onDeleted: (items: T[]) => void,
      params?: any
    ): Promise<ApiResponse> => {
      if (!client || !namespace) {
        console.log(
          "Cannot initialize watch: client or namespace not available, waiting for client or namespace to become available"
        )
        return { ok: false, message: "Client or namespace not available" }
      }

      // check if user is allowed to retrieve the object
      return canList(url, kind).then((res) => {
        if (!res.ok) {
          return {
            ok: false,
            message: `Cannot initialize watch for ${kind}: ${res.message}`,
            status: 403,
          }
        } else {
          const watch = client
            .watch(url, {
              params: params,
            })
            .on(client.WATCH_ERROR, () => {
              console.log("ERROR: Failed to watch resource")
            })
            .on(client.WATCH_ADDED, (items) => {
              console.log(`added ${items}`)
              addKind(items, kind)
              onAdded(items as T[])
            })
            .on(client.WATCH_MODIFIED, (items) => {
              console.log(`modified ${items}`)
              addKind(items, kind)
              onModified(items as T[])
            })
            .on(client.WATCH_DELETED, (items) => {
              console.log(`deleted ${items}`)
              addKind(items, kind)
              onDeleted(items as T[])
            })
          watch.start()
          return { ok: true, message: `Successfully watching ${kind}s` }
        }
      })
    },
    [client, namespace]
  )

  const addKind = (items: any, kind: string) => {
    items.forEach((item: any) => {
      item.kind = kind
    })
  }

  const canList = useCallback(
    async (url: string, kind: AllowedApiObjectKind): Promise<ApiResponse> => {
      let object = { metadata: { name: "" }, kind: kind }
      let result = await get<AllowedApiObject>(url, object)

      // we get back an object list, if we are authorized
      // @ts-ignore
      if (result.response?.kind == `${kind}List`) {
        return { ok: true, message: "" }
      }

      return { ok: result.ok, message: result.message }
    },
    [client, namespace]
  )

  return {
    get: get,
    create: create,
    update: update,
    deleteObject: deleteObject,
    watch: watch,
  }
}

export default useApi
