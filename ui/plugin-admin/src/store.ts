import { create } from "zustand"
import {
  Plugin,
  UpdateObjectAction,
  UpdatePluginInput
} from "./types/types"

export interface State {
  endpoint: string
  setEndpoint: (newEndpoint: string) => void
  urlStateKey: string
  setUrlStateKey: (newUrlStateKey: string) => void

  auth: any
  setAuth: (auth: any) => void
  loggedIn: boolean
  setLoggedIn: (loggedIn: boolean) => void
  logout: any

  plugins: Plugin[]
  updatePlugins: (input: UpdatePluginInput) => void
  showPluginDetails: boolean
  setShowPluginDetails: (showPluginDetails: boolean) => void
  pluginDetail: Plugin | null
  setPluginDetail: (plugin: Plugin) => void
}

// global zustand store. See how this works here: https://github.com/pmndrs/zustand
const useStore = create<State>((set) => ({
  endpoint: "",
  setEndpoint: (newEndpoint) => set((state) => ({ endpoint: newEndpoint })),
  urlStateKey: "",
  setUrlStateKey: (newUrlStateKey) =>
    set((state) => ({ urlStateKey: newUrlStateKey })),

  auth: null,
  setAuth: (auth) => set((state) => ({ auth: auth })),
  loggedIn: false,
  setLoggedIn: (loggedIn) => set((state) => ({ loggedIn: loggedIn })),
  logout: null,

  plugins: [],
  updatePlugins: (input: UpdatePluginInput) =>
    set((state) => {
      let plugins = [...state.plugins]
      // validate plugins: only accept input.plugins that have metadata.name set
      input.plugins = input.plugins.filter((plugin) => {
        return plugin.metadata?.name ?? undefined !== undefined
      })

      if (input.action === UpdateObjectAction.delete) {
        plugins = plugins.filter((knownPlugin) => {
          return input.plugins.some((inputPlugin) => {
            return knownPlugin.metadata!.name !== inputPlugin.metadata!.name
          })
        })
        return { ...state, plugins: plugins }
      }

      input.plugins.forEach((inputPlugin) => {
        const index = plugins.findIndex((knownPlugin) => {
          return knownPlugin.metadata!.name === inputPlugin.metadata!.name
        })
        if (index >= 0) {
          plugins[index] = inputPlugin
        } else {
          plugins.push(inputPlugin)
        }
      })
      return { ...state, plugins: plugins }
    }),
  showPluginDetails: false,
  setShowPluginDetails: (showPluginDetails) =>
    set((state) => ({ ...state, showPluginDetails: showPluginDetails })),

  pluginDetail: null,
  setPluginDetail: (plugin) => set((state) => ({ pluginDetail: plugin  })),
}))

export default useStore
