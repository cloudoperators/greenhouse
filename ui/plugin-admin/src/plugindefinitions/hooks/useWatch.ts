/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { useCallback } from "react"
import usePluginDefinitionsStore from "../store"
import {
  PluginDefinition,
  Secret,
  UpdateObjectAction,
} from "../../../../types/types"
import useClient from "./useClient"
import useNamespace from "./useNamespace"

export const useWatch = () => {
  const { namespace } = useNamespace()
  const { client: client } = useClient()
  const updatePluginDefinitions = usePluginDefinitionsStore(
    (state) => state.updatePluginDefinitions
  )
  const updateSecrets = usePluginDefinitionsStore(
    (state) => state.updateSecrets
  )

  const watchPluginDefinitions = useCallback(() => {
    if (!client || !namespace) return
    const watch = client
      .watch(`/apis/greenhouse.sap/v1alpha1/plugindefinitions`)
      .on(client.WATCH_ERROR, () =>
        console.log("ERROR: Failed to watch resource")
      )
      .on(client.WATCH_ADDED, (items) => {
        console.log("watch pd added", items)
        updatePluginDefinitions({
          pluginDefinitions: items as PluginDefinition[],
          action: UpdateObjectAction.add,
        })
      })
      .on(client.WATCH_MODIFIED, (items) => {
        updatePluginDefinitions({
          pluginDefinitions: items as PluginDefinition[],
          action: UpdateObjectAction.add,
        })
      })
      .on(client.WATCH_DELETED, (items) => {
        updatePluginDefinitions({
          pluginDefinitions: items as PluginDefinition[],
          action: UpdateObjectAction.delete,
        })
      })
    watch.start()
    return watch.cancel
  }, [client, namespace])

  const watchSecrets = useCallback(() => {
    if (!client || !namespace) return
    const watch = client
      .watch(`/api/v1/namespaces/${namespace}/secrets`)
      .on(client.WATCH_ERROR, () =>
        console.log("ERROR: Failed to watch resource")
      )
      .on(client.WATCH_ADDED, (items) => {
        console.log("watch s added", items)
        updateSecrets({
          secrets: items as Secret[],
          action: UpdateObjectAction.add,
        })
      })
      .on(client.WATCH_MODIFIED, (items) => {
        updateSecrets({
          secrets: items as Secret[],
          action: UpdateObjectAction.add,
        })
      })
      .on(client.WATCH_DELETED, (items) => {
        updateSecrets({
          secrets: items as Secret[],
          action: UpdateObjectAction.delete,
        })
      })
    watch.start()
    return watch.cancel
  }, [client, namespace])

  return {
    watchPluginDefinitions: watchPluginDefinitions,
    watchSecrets: watchSecrets,
  }
}

export default useWatch
