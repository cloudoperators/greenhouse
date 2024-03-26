/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

const createGlobalsSlice = (set, get) => ({
  globals: {
    urlStateKey: "",
    apiEndpoint: null,
    loggedIn: false,
    authData: null,
    pluginConfig: null,
    showDetailsFor: null,

    actions: {
      setUrlStateKey: (newUrlStateKey) =>
        set((state) => ({
          globals: { ...state.globals, urlStateKey: newUrlStateKey },
        })),

      setLoggedIn: (loggedIn) =>
        set((state) => ({
          globals: { ...state.globals, loggedIn: loggedIn },
        })),

      setAuthData: (authData) =>
        set((state) => ({ globals: { ...state.globals, authData: authData } })),

      setEndpoint: (apiEndpoint) =>
        set((state) => ({
          globals: { ...state.globals, apiEndpoint: apiEndpoint },
        })),

      setPluginConfig: (pluginConfig) => {
        // Sort plugins by id alphabetically, but put disabled plugins at the end
        let sortedPlugins = pluginConfig.sort((a, b) => {
          if (a.disabled && !b.disabled) {
            return 1
          } else if (!a.disabled && b.disabled) {
            return -1
          } else {
            return a.id.localeCompare(b.id)
          }
        })
        set((state) => ({
          globals: {
            ...state.globals,
            pluginConfig: sortedPlugins,
          },
        }))
      },

      setShowDetailsFor: (showDetailsFor) =>
        set((state) => ({
          globals: { ...state.globals, showDetailsFor: showDetailsFor },
        })),
    },
  },
})

export default createGlobalsSlice
