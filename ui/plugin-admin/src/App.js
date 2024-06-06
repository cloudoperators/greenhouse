/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useEffect } from "react"

import { AppShell, AppShellProvider } from "juno-ui-components"
import StoreProvider, {
  useGlobalsActions,
} from "./plugins/components/StoreProvider"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import AppContent from "./AppContent"
import styles from "./styles.scss"
import AsyncWorker from "./plugins/components/AsyncWorker"
import Auth from "./plugins/components/Auth"
import useCommunication from "./plugindefinitions/hooks/useCommunication"
import usePluginDefinitionsStore from "./plugindefinitions/store"
import markdown from "github-markdown-css/github-markdown.css"
import markdownDark from "github-markdown-css/github-markdown-dark.css"
import markdownLight from "github-markdown-css/github-markdown-light.css"

const URL_STATE_KEY = "plugin-admin"

const App = (props = {}) => {
  const { setUrlStateKey, setEndpoint } = useGlobalsActions()
  const setPluginDefinitionEndpoint = usePluginDefinitionsStore(
    (state) => state.setEndpoint
  )

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
  useCommunication()

  // on app initial load save Endpoint and URL_STATE_KEY so it can be
  // used from overall in the application
  useEffect(() => {
    // set to empty string to fetch local test data in dev mode
    setEndpoint(props.endpoint || "")
    setPluginDefinitionEndpoint(props.endpoint || "")
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

// the list styles are being reseted bei juno
// add them back so it works within a markdown container
const fixMarkdownLists = `
  ol {
      list-style: decimal;
  }
  ul {
    list-style: disc;
}
`

const StyledApp = (props) => {
  const theme = props.theme ? props.theme : "theme-dark"
  return (
    <AppShellProvider theme={`${props.theme ? props.theme : "theme-dark"}`}>
      {/* load styles inside the shadow dom */}
      <style>{styles.toString()}</style>
      <style>{markdown.toString()}</style>
      <style>
        {theme === "theme-dark"
          ? markdownDark.toString()
          : markdownLight.toString()}
      </style>
      <style>{fixMarkdownLists}</style>
      <StoreProvider>
        <App {...props} />
      </StoreProvider>
    </AppShellProvider>
  )
}

export default StyledApp
