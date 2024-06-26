/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { useCallback } from "react"
import useStore from "../store"
import { Cluster, UpdateObjectAction } from "../../../types/types"
import useClient from "./useClient"
import useNamespace from "./useNamespace"

export const useWatch = () => {
  const { namespace } = useNamespace()
  const { client: client } = useClient()
  const updateClusters = useStore((state) => state.updateClusters)

  const watchClusters = useCallback(() => {
    if (!client || !namespace) return
    const watch = client
      .watch(`/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/clusters`)
      .on(client.WATCH_ERROR, () =>
        console.log("ERROR: Failed to watch resource")
      )
      .on(client.WATCH_ADDED, (items) => {
        updateClusters({
          clusters: items as Cluster[],
          action: UpdateObjectAction.add,
        })
      })
      .on(client.WATCH_MODIFIED, (items) => {
        updateClusters({
          clusters: items as Cluster[],
          action: UpdateObjectAction.add,
        })
      })
      .on(client.WATCH_DELETED, (items) => {
        updateClusters({
          clusters: items as Cluster[],
          action: UpdateObjectAction.delete,
        })
      })
    watch.start()
    return watch.cancel
  }, [client, namespace])

  return {
    watchClusters: watchClusters,
  }
}

export default useWatch
