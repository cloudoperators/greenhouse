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
import useStore from "../store"
import { Cluster, PluginConfig, UpdateClusterAction } from "../types/types"
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
          action: UpdateClusterAction.add,
        })
      })
      .on(client.WATCH_MODIFIED, (items) => {
        updateClusters({
          clusters: items as Cluster[],
          action: UpdateClusterAction.add,
        })
      })
      .on(client.WATCH_DELETED, (items) => {
        updateClusters({
          clusters: items as Cluster[],
          action: UpdateClusterAction.delete,
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
