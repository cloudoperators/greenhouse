/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Juno contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React from "react"

import {
  AppBody,
  AppShellProvider,
  MainContainer,
  MainContainerInner,
  ContentContainer,
} from "juno-ui-components"
import StoreProvider from "./components/StoreProvider"
import UrlState from "./components/UrlState"

import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import AppContent from "./AppContent"
import styles from "./styles.scss"
import OrgInfo from "./components/OrgInfo"
import SideNav from "./components/SideNav"
import AsyncWorker from "./components/AsyncWorker"
import { MessagesProvider, Messages } from "messages-provider"
import Auth from "./components/Auth"

const App = (props = {}) => {
  // to be deleted
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        meta: {
          endpoint: props.endpoint || props.currentHost || "",
        },
      },
    },
  })

  // support only embeded mode for now. This will probably never be started standalone
  // page layout is copied from juno-ui-components/src/components/AppShell/AppShell.component.js
  return (
    <QueryClientProvider client={queryClient}>
      <AsyncWorker />
      <AppBody data-testid="greenhouse-management">
        <Messages className="mb-4" />
        <Auth>
          <UrlState>
            <OrgInfo />
            <MainContainer>
              <MainContainerInner fullWidth={true}>
                <SideNav />
                <ContentContainer>
                  <AppContent {...props} />
                </ContentContainer>
              </MainContainerInner>
            </MainContainer>
          </UrlState>
        </Auth>
      </AppBody>
    </QueryClientProvider>
  )
}

const StyledApp = (props) => {
  return (
    <AppShellProvider theme={`${props.theme ? props.theme : "theme-dark"}`}>
      <style>{styles.toString()}</style>
      <MessagesProvider>
        <StoreProvider options={props}>
          <App {...props} />
        </StoreProvider>
      </MessagesProvider>
    </AppShellProvider>
  )
}

export default StyledApp
