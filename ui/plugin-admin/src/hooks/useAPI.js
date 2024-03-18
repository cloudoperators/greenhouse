import { useCallback, useMemo } from "react"
import useClient from "./useClient"
import { useAuthData, usePluginConfig } from "../components/StoreProvider"
import { useGlobalsActions } from "../components/StoreProvider"

export const useAPI = () => {
  const { client } = useClient()
  const authData = useAuthData()
  const pluginConfig = usePluginConfig()

  const { setPluginConfig } = useGlobalsActions()

  const namespace = useMemo(() => {
    if (!authData?.raw?.groups) return null
    const orgString = authData?.raw?.groups.find(
      (g) => g.indexOf("organization:") === 0
    )
    if (!orgString) return null
    return orgString.split(":")[1]
  }, [authData?.raw?.groups])

  const buildExternalServicesUrls = (exposedServices) => {
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
        let allPlugins = []

        items?.items?.forEach((item) => {
          console.log("jjoooo", JSON.stringify(item.status?.exposedServices))

          const name = item.spec?.displayName
            ? item.spec.displayName
            : item.metadata?.name
          const disabled = item.spec?.disabled
          const version = item.status?.version
          const clusterName = item.spec?.clusterName
          const externalServicesUrls = buildExternalServicesUrls(
            item.status?.exposedServices
          )
          const statusConditions = item.status?.statusConditions?.conditions
          const optionValues = item.spec?.optionValues
          const raw = item

          console.log("jjoooo", JSON.stringify(externalServicesUrls))

          if (!disabled) {
            allPlugins.push({
              name,
              version,
              clusterName,
              externalServicesUrls,
              statusConditions,
              optionValues,
              raw,
            })
          }
        })
        setPluginConfig(allPlugins)
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
