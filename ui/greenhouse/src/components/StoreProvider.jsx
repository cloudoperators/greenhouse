/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Juno contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { createContext, useContext } from "react"
import { useStore as create } from "zustand"
import createStore from "../lib/store"

const StoreContext = createContext()
const StoreProvider = ({ options, children }) => (
  <StoreContext.Provider value={createStore(options || {})}>
    {children}
  </StoreContext.Provider>
)

// build a hook from the store
const useStore = (selector) => create(useContext(StoreContext).store, selector)

// AUTH
export const useAuthData = () => useStore((s) => s.auth.data)
export const useAuthIsProcessing = () => useStore((s) => s.auth.isProcessing)
export const useAuthLoggedIn = () => useStore((s) => s.auth.loggedIn)
export const useAuthError = () => useStore((s) => s.auth.error)
export const useAuthLastAction = () => useStore((s) => s.auth.lastAction)
export const useAuthAppLoaded = () => useStore((s) => s.auth.appLoaded)
export const useAuthAppIsLoading = () => useStore((s) => s.auth.appIsLoading)
export const useAuthActions = () => useStore((s) => s.auth.actions)

// APPS
export const usePlugin = () => useContext(StoreContext).plugin

// GLOBAL
export const useGlobalsApiEndpoint = () =>
  useStore((s) => s.globals.apiEndpoint)
export const useGlobalsAssetsHost = () => useStore((s) => s.globals.assetsHost)
export const useGlobalsIsUrlStateSetup = () =>
  useStore((state) => state.globals.isUrlStateSetup)
export const useGlobalsActions = () => useStore((s) => s.globals.actions)
export const useGlobalsEnvironment = () =>
  useStore((s) => s.globals.environment)
export const useDemoMode = () => useStore((s) => s.globals.demoMode)
export const useDemoUserToken = () => useStore((s) => s.globals.demoUserToken)

export default StoreProvider
