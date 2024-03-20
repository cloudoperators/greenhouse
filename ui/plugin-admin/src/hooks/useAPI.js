import { useCallback, useMemo } from "react"
import useClient from "./useClient"
import { useAuthData } from "../components/StoreProvider"
import { useGlobalsActions } from "../components/StoreProvider"

import { getResourceStatusFromKubernetesConditions } from "../../../utils/resourceStatus"

export const buildExternalServicesUrls = (exposedServices) => {
  // extract url and the name from the object and create a link
  if (!exposedServices) return null

  const links = []
  for (const url in exposedServices) {
    const currentObject = exposedServices[url]

    links.push({
      url: url,
      name: currentObject.name ? currentObject.name : url,
    })
  }

  return links
}

export const createPluginConfig = (items) => {
  let allPlugins = []
  items.forEach((item) => {
    const id = item?.metadata?.name ? item.metadata?.name : "Unknown"
    const name = item?.spec?.displayName ? item.spec.displayName : id
    const disabled = item?.spec?.disabled
    const version = item?.status?.version
    const clusterName = item?.spec?.clusterName
    const externalServicesUrls = buildExternalServicesUrls(
      item?.status?.exposedServices
    )
    const statusConditions = item?.status?.statusConditions?.conditions
    const readyStatus = statusConditions
      ? getResourceStatusFromKubernetesConditions(statusConditions)
      : null
    const optionValues = item?.spec?.optionValues
    const raw = item

    if (!disabled) {
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
      })
    }
  })
  return allPlugins
}

export const useAPI = () => {
  const { client } = useClient()
  const authData = useAuthData()
  const { setPluginConfig } = useGlobalsActions()

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

    const getPromise = client
      .get(
        `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/pluginconfigs`,
        {
          limit: 500,
        }
      )
      .then((items) => {
        setPluginConfig(createPluginConfig(items?.items))
      })
      .catch((e) => {
        console.error("ERROR: Failed to get resource", e)
      })

    return () => {
      return getPromise
    }
  }, [client])

  return { getPlugins }
}

export default useAPI
