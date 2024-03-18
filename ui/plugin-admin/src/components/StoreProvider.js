import React, { createContext, useContext } from "react"
import { useStore as create } from "zustand"
import createStore from "../lib/store"

const StoreContext = createContext()
const StoreProvider = ({ children }) => (
  <StoreContext.Provider value={createStore()}>
    {children}
  </StoreContext.Provider>
)

const useAppStore = (selector) => create(useContext(StoreContext), selector)

export const useGlobalsUrlStateKey = () =>
  useAppStore((state) => state.globals.urlStateKey)
export const useApiEndpoint = () =>
  useAppStore((state) => state.globals.apiEndpoint)
export const useLoggedIn = () => useAppStore((state) => state.globals.loggedIn)
export const useAuthData = () => useAppStore((state) => state.globals.authData)
export const usePluginConfig = () =>
  useAppStore((state) => state.globals.pluginConfig)
export const useShowDetailsFor = () =>
  useAppStore((state) => state.globals.showDetailsFor)

export const useGlobalsActions = () =>
  useAppStore((state) => state.globals.actions)

export default StoreProvider
