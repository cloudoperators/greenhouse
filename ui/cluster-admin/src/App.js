/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React from "react"

import useStore from "./store"
import { AppShell, AppShellProvider } from "@cloudoperators/juno-ui-components"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import AppContent from "./AppContent"
import styles from "./styles.scss"
import useCommunication from "./hooks/useCommunication"
import AsyncWorker from "./components/AsyncWorker"

/* IMPORTANT: Replace this with your app's name */
const URL_STATE_KEY = "greenhouse-cluster-admin"
/* --------------------------- */

const App = (props = {}) => {
  const setEndpoint = useStore((state) => state.setEndpoint)
  const setUrlStateKey = useStore((state) => state.setUrlStateKey)
  // Create query client which it can be used from overall in the app
  const queryClient = new QueryClient()

  useCommunication()

  // on app initial load save Endpoint and URL_STATE_KEY so it can be
  // used from overall in the application
  React.useEffect(() => {
    // set to empty string to fetch local test data in dev mode
    setEndpoint(props.endpoint || props.currentHost || "")
    setUrlStateKey(URL_STATE_KEY)
  }, [])

  return (
    <QueryClientProvider client={queryClient}>
      <AsyncWorker consumerId={URL_STATE_KEY} />
      <AppShell
        pageHeader="Converged Cloud | Clusters"
        embedded={props.embedded === "true" || props.embedded === true}
      >
        <AppContent props={props} />
      </AppShell>
    </QueryClientProvider>
  )
}

const StyledApp = (props) => {
  return (
    <AppShellProvider theme={`${props.theme ? props.theme : "theme-dark"}`}>
      {/* load styles inside the shadow dom */}
      <style>{styles.toString()}</style>
      <App {...props} />
    </AppShellProvider>
  )
}

export default StyledApp
