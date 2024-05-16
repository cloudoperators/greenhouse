/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

function sortPluginConfigItems(items) {
  return items.sort((a, b) => {
    if (a?.spec?.disabled && !b?.spec?.disabled) {
      return 1
    } else if (!a?.spec?.disabled && b?.spec?.disabled) {
      return -1
    } else {
      return a?.metadata?.uid.localeCompare(b?.metadata?.uid)
    }
  })
}

function uniqPluginConfigItems(items) {
  return items.filter((item, index, array) => array.indexOf(item) === index)
}

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

      addPluginConfigItems: (pluginConfigItems) => {
        const items = (get().plugin.pluginConfig || []).slice()
        let newItems = uniqPluginConfigItems([...items, ...pluginConfigItems])
        newItems = sortPluginConfigItems(newItems)

        set((state) => ({
          plugin: {
            ...state.plugin,
            pluginConfig: newItems,
          },
        }))
      },
      modifyPluginConfigItems: (modifiedItems) => {
        const items = (get().plugin.pluginConfig || []).slice()

        const updatedItems = items.map((item) => {
          const matchingModifiedItem = modifiedItems.find(
            (modifiedItem) => modifiedItem.metadata.uid === item.metadata.uid
          )
          return matchingModifiedItem || item
        })

        let newItems = uniqPluginConfigItems(updatedItems)
        newItems = sortPluginConfigItems(updatedItems)

        set((state) => ({
          plugin: {
            ...state.plugin,
            pluginConfig: newItems,
          },
        }))
      },
      deletePluginConfigItems: (pluginConfigItems) => {
        const items = (get().plugin.pluginConfig || []).slice() // Get items

        let updatedItems = items.filter((item) => {
          return !pluginConfigItems.find(
            (pci) => pci.metadata.uid === item.metadata.uid
          )
        })

        let newItems = uniqPluginConfigItems(updatedItems)
        newItems = sortPluginConfigItems(updatedItems)

        set((state) => ({
          plugin: {
            ...state.plugin,
            pluginConfig: newItems,
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
