/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

const createPluginSlice = (set, get) => ({
  plugin: {
    pluginConfig: null,
    showDetailsFor: null,

    actions: {
      setPluginConfig: (pluginConfig) => {
        // Sort plugins by id alphabetically, but put disabled plugins at the end
        let sortedPlugins = pluginConfig.sort((a, b) => {
          if (a?.disabled && !b?.disabled) {
            return 1
          } else if (!a?.disabled && b?.disabled) {
            return -1
          } else {
            return a.id.localeCompare(b.id)
          }
        })
        set((state) => ({
          plugin: {
            ...state.plugin,
            pluginConfig: sortedPlugins,
          },
        }))
      },

      setShowDetailsFor: (showDetailsFor) =>
        set((state) => ({
          plugin: { ...state.plugin, showDetailsFor: showDetailsFor },
        })),
    },
  },
})

export default createPluginSlice
