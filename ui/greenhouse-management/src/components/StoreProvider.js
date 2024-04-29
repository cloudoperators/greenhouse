/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Juno contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { createContext, useContext } from "react"
import { useStore as create } from "zustand"
import createStore from "../lib/store"

const StoreContext = createContext()
const StoreProvider = ({ options, children }) => (
  <StoreContext.Provider value={createStore(options)}>
    {children}
  </StoreContext.Provider>
)

const useAppStore = (selector) => create(useContext(StoreContext), selector)

export const useIsUrlStateSetup = () =>
  useAppStore((state) => state.isUrlStateSetup)
export const useAssetsUrl = () => useAppStore((state) => state.assetsUrl)
export const usePluginConfig = () => useAppStore((state) => state.pluginConfig)
export const usePluginActive = () => useAppStore((state) => state.pluginActive)
export const useApiEndpoint = () => useAppStore((state) => state.apiEndpoint)
export const useAuthData = () => useAppStore((state) => state.authData.data)
export const useAuthAppLoaded = () =>
  useAppStore((state) => state.authAppLoaded)
export const useIsLoggedIn = () =>
  useAppStore((state) => state.authData.loggedIn)

export const useActions = () => useAppStore((state) => state.actions)

export default StoreProvider
