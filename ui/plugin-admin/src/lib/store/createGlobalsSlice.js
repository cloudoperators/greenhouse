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
    },
  },
})

export default createGlobalsSlice
