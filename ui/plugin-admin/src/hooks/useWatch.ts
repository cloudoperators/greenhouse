import { useCallback } from "react"
import useStore from "../store"
import { Plugin, UpdateObjectAction } from "../types/types"
import useClient from "./useClient"
import useNamespace from "./useNamespace"

export const useWatch = () => {
  const { namespace } = useNamespace()
  const { client: client } = useClient()
  const updatePlugins = useStore((state) => state.updatePlugins)

  const watchPlugins = useCallback(() => {
    console.log(client)
    if (!client || !namespace) return
    const watch = client
      .watch(`/apis/greenhouse.sap/v1alpha1/plugins`)
      .on(client.WATCH_ERROR, () =>
        console.log("ERROR: Failed to watch resource")
      )
      .on(client.WATCH_ADDED, (items) => {
        console.log("watch added", items)
        updatePlugins({
          plugins: items as Plugin[],
          action: UpdateObjectAction.add,
        })
      })
      .on(client.WATCH_MODIFIED, (items) => {
        updatePlugins({
          plugins: items as Plugin[],
          action: UpdateObjectAction.add,
        })
      })
      .on(client.WATCH_DELETED, (items) => {
        updatePlugins({
          plugins: items as Plugin[],
          action: UpdateObjectAction.delete,
        })
      })
    watch.start()
    return watch.cancel
  }, [client, namespace])

  return {
    watchPlugins: watchPlugins,
  }
}

export default useWatch
