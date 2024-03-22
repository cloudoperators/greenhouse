import React, { useEffect } from "react"

import { AppShell, AppShellProvider } from "juno-ui-components"
import StoreProvider, { useGlobalsActions } from "./components/StoreProvider"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import AppContent from "./AppContent"
import styles from "./styles.scss"
import AsyncWorker from "./components/AsyncWorker"
import Auth from "./components/Auth"

const URL_STATE_KEY = "plugin-admin"

const App = (props = {}) => {
  const { setUrlStateKey, setEndpoint } = useGlobalsActions()

  // Create query client which it can be used from overall in the app
  // set default endpoint to fetch data
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        meta: {
          endpoint: props.endpoint || "",
        },
      },
    },
  })

  // on app initial load save Endpoint and URL_STATE_KEY so it can be
  // used from overall in the application
  useEffect(() => {
    // set to empty string to fetch local test data in dev mode
    setEndpoint(props.endpoint || "")
    setUrlStateKey(URL_STATE_KEY)
  }, [])

  return (
    <QueryClientProvider client={queryClient}>
      <AppShell
        pageHeader="Converged Cloud | Plugins"
        embedded={props.embedded === "true" || props.embedded === true}
      >
        <AsyncWorker />
        <Auth>
          <AppContent props={props} />
        </Auth>
      </AppShell>
    </QueryClientProvider>
  )
}

const StyledApp = (props) => {
  return (
    <AppShellProvider theme={`${props.theme ? props.theme : "theme-dark"}`}>
      {/* load styles inside the shadow dom */}
      <style>{styles.toString()}</style>
      <StoreProvider>
        <App {...props} />
      </StoreProvider>
    </AppShellProvider>
  )
}

export default StyledApp
