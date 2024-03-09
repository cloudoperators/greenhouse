import React, { useEffect, useMemo, useLayoutEffect } from "react"
import { AppShell, AppShellProvider } from "juno-ui-components"
import AppContent from "./AppContent"
import styles from "./styles.scss"
import useCommunication from "./hooks/useCommunication"
import AsyncWorker from "./components/AsyncWorker"
import { MessagesProvider } from "messages-provider"
import StoreProvider, { useStoreActions } from "./components/StoreProvider"
import { fetchProxyInitDB } from "utils"
import db from "../db.json"

const App = (props = {}) => {
  const { setEndpoint, setMock } = useStoreActions()
  const isMock = useMemo(
    () => props.isMock === true || props.isMock === "true",
    [props.isMock]
  )

  useCommunication()

  // setup the mock db.json
  useEffect(() => {
    if (isMock) {
      setMock(true)

      fetchProxyInitDB(db, {
        debug: true,
        rewriteRoutes: {
          "/(?:apis/greenhouse\\.sap/v1alpha1/namespaces/[^/]+/teammemberships/(.*))":
            "/$1",
        },
      })
    }
  }, [])

  // on load application save the props to be used in oder components
  useLayoutEffect(() => {
    setEndpoint(isMock ? window.location.origin : props?.endpoint)
  }, [props?.endpoint, isMock])

  return (
    <MessagesProvider>
      <AsyncWorker/>
      <AppShell
        pageHeader="Converged Cloud | Team Members"
        embedded={props.embedded === "true" || props.embedded === true}
      >
        <AppContent props={props} />
      </AppShell>
    </MessagesProvider>
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
