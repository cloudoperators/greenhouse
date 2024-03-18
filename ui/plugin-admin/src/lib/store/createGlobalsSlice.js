const createGlobalsSlice = (set, get) => ({
  globals: {
    urlStateKey: "",
    apiEndpoint: null,
    loggedIn: false,
    authData: null,
    pluginConfig: null,

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

      setPluginConfig: (pluginConfig) =>
        set((state) => ({
          globals: { ...state.globals, pluginConfig: pluginConfig },
        })),

      setShowDetailsFor: (showDetailsFor) =>
        set((state) => ({
          globals: { ...state.globals, showDetailsFor: showDetailsFor },
        })),
    },
  },
})

export default createGlobalsSlice
