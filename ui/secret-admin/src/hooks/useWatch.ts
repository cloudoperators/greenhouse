/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { useCallback } from "react"
import useStore from "../store"
import { Secret, UpdateObjectAction } from "../../../types/types"
import useClient from "./useClient"
import useNamespace from "./useNamespace"

export const useWatch = () => {
  const { namespace } = useNamespace()
  const { client: client } = useClient()
  const updateSecrets = useStore((state) => state.updateSecrets)

  // exclude helm secrets from watch
  const fieldSelectorKey = "type"
  const isHelmSecretValue = "helm.sh/release.v1"
  const fieldSelector = `${fieldSelectorKey}!=${isHelmSecretValue}`

  const watchSecrets = useCallback(() => {
    if (!client || !namespace) return
    const watch = client
      .watch(`/api/v1/namespaces/${namespace}/secrets`, {
        params: {
          fieldSelector: fieldSelector,
        },
      })
      .on(client.WATCH_ERROR, (e) => {
        console.log("ERROR: Failed to watch resource")
      })
      .on(client.WATCH_ADDED, (items) => {
        addKind(items, "Secret")
        updateSecrets({
          secrets: items as Secret[],
          action: UpdateObjectAction.add,
        })
      })
      .on(client.WATCH_MODIFIED, (items) => {
        addKind(items, "Secret")
        updateSecrets({
          secrets: items as Secret[],
          action: UpdateObjectAction.add,
        })
      })
      .on(client.WATCH_DELETED, (items) => {
        addKind(items, "Secret")
        updateSecrets({
          secrets: items as Secret[],
          action: UpdateObjectAction.delete,
        })
      })
    watch.start()
    return watch.cancel
  }, [client, namespace])

  const addKind = (items: any, kind: string) => {
    items.forEach((item: any) => {
      item.kind = kind
    })
  }

  return {
    watchSecrets: watchSecrets,
  }
}

export default useWatch
