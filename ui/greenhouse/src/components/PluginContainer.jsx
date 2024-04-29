/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Juno contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React from "react"
import Plugin from "./Plugin"
import { usePlugin, useGlobalsEnvironment } from "../components/StoreProvider"
import useApi from "../hooks/useApi"
import { useLayoutEffect } from "react"
import HintLoading from "./shared/HintLoading"
import { parseError } from "../lib/helpers"
import { useActions, MessagesProvider } from "messages-provider"

const PluginContainer = () => {
  const { getPluginConfigs } = useApi()
  const environment = useGlobalsEnvironment()
  const config = usePlugin().config()
  const isFetching = usePlugin().isFetching()
  const { addMessage } = useActions()

  // prevent to load a plugin before the config is fetched to avoid rerendering do tue the default plugin greenhouse-mng
  const [displayPlugin, setDisplayPlugin] = React.useState(false)

  const requestConfig = usePlugin().requestConfig
  const receiveConfig = usePlugin().receiveConfig
  const receiveError = usePlugin().receiveError

  const availableAppIds = React.useMemo(() => Object.keys(config), [config])

  useLayoutEffect(() => {
    if (!getPluginConfigs) return
    requestConfig()

    // fetch configs from kubernetes
    getPluginConfigs()
      .then((kubernetesConfig) => {
        receiveConfig(kubernetesConfig)
      })
      .catch((error) => {
        // error fetching configs
        receiveError(error.message)
        addMessage({
          variant: "error",
          text: parseError(error),
        })
      })
      .finally(() => {
        setDisplayPlugin(true)
      })
  }, [getPluginConfigs, environment])

  return (
    <>
      {displayPlugin &&
        availableAppIds.length > 0 &&
        availableAppIds.map((id, i) => (
          <MessagesProvider key={i}>
            <Plugin id={id} />
          </MessagesProvider>
        ))}
      {!isFetching &&
        !displayPlugin &&
        availableAppIds.length <= 0 &&
        "No plugins available."}
    </>
  )
}

export default PluginContainer
