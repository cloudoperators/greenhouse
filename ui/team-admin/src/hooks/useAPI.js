import { useCallback } from "react"
import {
  useNamespace,
  useStoreActions,
  useTeamMemberships,
} from "../components/StoreProvider"
import useClient from "./useClient"
import { useActions } from "messages-provider"
import { parseError } from "../lib/helpers"
import updateTeamMemberships from "./helper/updateTeamMemberships"

export const useAPI = () => {
  const namespace = useNamespace()
  const { client } = useClient()
  const { addMessage } = useActions()
  const { setTeamMemberships } = useStoreActions()
  const teamMemberships = useTeamMemberships()

  const watchTeamMembers = useCallback(() => {
    if (!client || !namespace) return

    const watch = client.watch(
      `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/teammemberships`
    )

    watch.on(client.WATCH_ERROR, (e) => {
      console.log("ERROR: Failed to watch resource", e)
      addMessage({
        variant: "error",
        text: parseError(e.message),
      })
    })

    watch.on(client.WATCH_ADDED, (items) => {
      updateTeamMemberships(teamMemberships, setTeamMemberships, {
        added: items,
      })
    })
    watch.on(client.WATCH_MODIFIED, (items) => {
      updateTeamMemberships(teamMemberships, setTeamMemberships, {
        modified: items,
      })
    })
    watch.on(client.WATCH_DELETED, (items) => {
      updateTeamMemberships(teamMemberships, setTeamMemberships, {
        deleted: items,
      })
    })

    watch.start()

    const getPromise = client
      .get(
        `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/teammemberships`
      )
      .then((items) => {
        updateTeamMemberships(teamMemberships, setTeamMemberships, {
          added: items,
        })
      })
      .catch((e) => {
        console.error("ERROR: Failed to get resource", e)
        addMessage({
          variant: "error",
          text: parseError(e.message),
        })
      })

    return () => {
      watch.cancel()
      return getPromise
    }
  }, [
    client,
    namespace,
    teamMemberships,
    setTeamMemberships,
    addMessage,
    parseError,
  ])

  return { watchTeamMembers }
}

export default useAPI
