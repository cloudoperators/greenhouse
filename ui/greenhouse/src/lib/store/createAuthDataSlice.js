/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Juno contributors
 * SPDX-License-Identifier: Apache-2.0
 */

const ACTIONS = {
  SIGN_ON: "signOn",
  SIGN_OUT: "signOut",
}

const createAuthDataSlice = (set, get) => ({
  auth: {
    data: null,
    isProcessing: false,
    loggedIn: false,
    error: null,
    lastAction: {},
    appLoaded: false,
    appIsLoading: false,

    actions: {
      setAppLoaded: (appLoaded) => {
        set(
          (state) => ({ auth: { ...state.auth, appLoaded } }),
          false,
          "auth/setAppLoaded"
        )
      },
      setData: (data = {}) => {
        set(
          (state) => ({
            auth: {
              ...state.auth,
              isProcessing: data ? data.isProcessing : false,
              loggedIn: data ? data.loggedIn : false,
              error: data ? data.error : null,
              data: data ? data.auth : null,
            },
          }),
          false,
          "auth/setData"
        )
        if (!data) get().auth.actions.setAction(ACTIONS.SIGN_OUT)
      },
      setAction: (name) =>
        set(
          (state) => ({
            auth: {
              ...state.auth,
              lastAction: { name: name, updatedAt: Date.now() },
            },
          }),
          false,
          "auth/setAction"
        ),
      login: () => {
        // logout
        get().auth.actions.setAction(ACTIONS.SIGN_OUT)
        get().auth.actions.setAction(ACTIONS.SIGN_ON)
      },
      logout: () => get().auth.actions.setAction(ACTIONS.SIGN_OUT),
    },
  },
})

export default createAuthDataSlice
