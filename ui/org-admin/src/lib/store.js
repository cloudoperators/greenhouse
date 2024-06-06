/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { createStore } from "zustand"
import { devtools } from "zustand/middleware"
import { managementPluginConfig } from "../../package.json"
import { useActions as messageActions } from "messages-provider"

export default (options) => {
  // check the managementPluginConfig is an object and not array or string
  const { addMessage } = messageActions()
  let configs = managementPluginConfig

  // check if the managementPluginConfig is an object with key values
  if (
    typeof configs !== "object" ||
    Array.isArray(configs) ||
    Object.keys(configs).length === 0
  ) {
    configs = {}
    addMessage({
      variant: "error",
      text: "managementPluginConfig is not an object with key values in the package.json",
    })
  }

  // set the endpoint and embedded props to the admin plugin comming package.json
  Object.keys(configs).forEach((key) => {
    // pull latest version in dev and qa
    configs[key].version =
      options.environment == "qa" || options.environment == "development"
        ? "latest"
        : configs[key].version
    configs[key].props = {
      endpoint: options.apiEndpoint,
      embedded: true,
    }
    // remove attribute releaseState from the plugin config if it set and it is different from "alpha" or "beta" or "stable"
    if (
      configs[key].releaseState &&
      !["alpha", "beta", "stable"].includes(configs[key].releaseState)
    ) {
      delete configs[key].releaseState
      addMessage({
        variant: "warning",
        text: `The releaseState attribute in the plugin config ${configs[key].name} is not valid. It should be "alpha", "beta" or "stable"`,
      })
    }
  })

  return createStore(
    devtools((set, get) => ({
      isUrlStateSetup: false,
      assetsUrl: options.assetsUrl,
      apiEndpoint: options.apiEndpoint,
      pluginConfig: configs,
      authData: {
        loggedIn: false,
        error: null,
        data: null,
      },
      authAppLoaded: false,
      pluginActive: "greenhouse-cluster-admin", // name of the active plugin default

      actions: {
        setPluginActive: (pluginId) => {
          // find the pluginConfig which plugin name matches the plugin id
          const plugin = Object.values(get().pluginConfig).find(
            (plugin) => plugin.name === pluginId
          )
          if (!plugin) return

          set(
            (state) => {
              state.pluginActive = plugin.name
            },
            false,
            "setPluginActive"
          )
        },
        setIsUrlStateSetup: (isSetup) =>
          set(
            (state) => {
              state.isUrlStateSetup = isSetup
            },
            false,
            "setIsUrlStateSetup"
          ),
        setAuthData: (data) =>
          set(
            (state) => ({
              authData: {
                ...state.auth,
                loggedIn: data ? data.loggedIn : false,
                error: data ? data.error : null,
                data: data ? data.auth : null,
              },
            }),
            false,
            "setAuthData"
          ),
        setAuthAppLoaded: (loaded) =>
          set(
            (state) => {
              state.authAppLoaded = loaded
            },
            false,
            "setAuthAppLoaded"
          ),
      },
    }))
  )
}
