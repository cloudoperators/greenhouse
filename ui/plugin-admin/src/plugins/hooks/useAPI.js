/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { useCallback, useMemo } from "react"
import useClient from "./useClient"
import { useAuthData } from "../components/StoreProvider"
import { usePluginActions } from "../components/StoreProvider"

import { getResourceStatusFromKubernetesConditions } from "../../../../utils/resourceStatus"

// Extracts the external services from the object and creates links which are used in the plugin list / detail

// Creates a flat object from the plugin config data
export const createPluginConfig = (items) => {
  let allPlugins = []

  items.forEach((item) => {
    // unknown is used as a last fallback, should not happen
    const id = item?.metadata?.name ? item.metadata?.name : "Unknown"
    const name = item?.spec?.displayName ? item.spec.displayName : id
    const disabled = item?.spec?.disabled
    const version = item?.status?.version
    const clusterName = item?.spec?.clusterName
    // build urls and name in a array of objects
    const externalServicesUrls = buildExternalServicesUrls(
      item?.status?.exposedServices
    )
    const statusConditions = item?.status?.statusConditions?.conditions
    // get a status object with icon and text for the plugin from imported function
    const readyStatus = statusConditions
      ? getResourceStatusFromKubernetesConditions(statusConditions)
      : null
    const optionValues = item?.spec?.optionValues
    const raw = item

    allPlugins.push({
      id,
      name,
      version,
      clusterName,
      externalServicesUrls,
      statusConditions,
      readyStatus,
      optionValues,
      raw,
      disabled,
    })
  })
  return allPlugins
}

export const useAPI = () => {
  const { client } = useClient()
  const authData = useAuthData()
  const {
    addPluginConfigItems,
    modifyPluginConfigItems,
    deletePluginConfigItems,
  } = usePluginActions()

  const namespace = useMemo(() => {
    if (!authData?.raw?.groups) return null
    const orgString = authData?.raw?.groups.find(
      (g) => g.indexOf("organization:") === 0
    )
    if (!orgString) return null
    return orgString.split(":")[1]
  }, [authData?.raw?.groups])

  const getPlugins = useCallback(() => {
    if (!client || !namespace) return

    // const getPromise = client
    //   .get(
    //     `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/plugins`,
    //     {
    //       limit: 500,
    //     }
    //   )
    //   .then((items) => {
    //     setPluginConfig(createPluginConfig(items?.items))
    //   })
    //   .catch((e) => {
    //     console.error("ERROR: Failed to get resource", e)
    //   })

    // return () => {
    //   return getPromise
    // }
    const watch = client
      .watch(`/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/plugins`)
      .on(client.WATCH_ERROR, () => console.log("ERROR"))
      .on(client.WATCH_ADDED, (items) => {
        addPluginConfigItems(items)
      })
      .on(client.WATCH_MODIFIED, (items) => {
        modifyPluginConfigItems(items)
      })
      .on(client.WATCH_DELETED, (items) => {
        deletePluginConfigItems(items)
      })

    watch.start()
    return watch.cancel
  }, [client])

  return { getPlugins }
}

export default useAPI
