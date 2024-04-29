/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Juno contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useLayoutEffect } from "react"

import ShellLayout from "./components/layout/ShellLayout"
import Auth from "./components/Auth"
import styles from "./styles.scss"
import { AppShellProvider } from "juno-ui-components"
import PluginContainer from "./components/PluginContainer"
import AsyncWorker from "./components/AsyncWorker"
import StoreProvider, { useGlobalsActions } from "./components/StoreProvider"
import { MessagesProvider } from "messages-provider"

const Shell = (props = {}) => {
  const { setApiEndpoint, setAssetsHost, setDemoUserToken, setEnvironment } =
    useGlobalsActions()

  // INIT
  // on app initial load save Endpoint and URL_STATE_KEY so it can be
  // used from overall in the application
  useLayoutEffect(() => {
    if (!setApiEndpoint || !setAssetsHost || !setDemoUserToken) return
    // set to empty string to fetch local test data in dev mode
    setEnvironment(props.environment)
    setApiEndpoint(props.apiEndpoint)
    setAssetsHost(props.currentHost)
    setDemoUserToken(props.demoUserToken)
  }, [setApiEndpoint, setAssetsHost, setDemoUserToken])

  return (
    <Auth
      clientId={props?.authClientId}
      issuerUrl={props?.authIssuerUrl}
      demoOrg={props?.demoOrg || "demo"}
      mock={props?.mockAuth}
    >
      <ShellLayout>
        <PluginContainer />
      </ShellLayout>
    </Auth>
  )
}

const StyledShell = (props) => {
  return (
    <AppShellProvider>
      {/* load styles inside the shadow dom */}
      <style>{styles.toString()}</style>
      <StoreProvider options={props}>
        <MessagesProvider>
          <AsyncWorker />
          <Shell {...props} />
        </MessagesProvider>
      </StoreProvider>
    </AppShellProvider>
  )
}

export default StyledShell
