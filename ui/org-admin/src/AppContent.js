/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useLayoutEffect } from "react"
import PluginContainer from "./components/PluginContainer"
import { useApiEndpoint, useAssetsUrl } from "./components/StoreProvider"
import { useActions as messageActions } from "@cloudoperators/juno-messages-provider"
import { Container } from "@cloudoperators/juno-ui-components"

const AppContent = () => {
  const { addMessage } = messageActions()
  const apiEndpoint = useApiEndpoint()
  const assetsUrl = useAssetsUrl()

  useLayoutEffect(() => {
    if (!apiEndpoint) {
      addMessage({
        variant: "warning",
        text: " required api endpoint not set",
      })
    }

    if (!assetsUrl) {
      addMessage({
        variant: "warning",
        text: "required assets url not set",
      })
    }

    // Make these two props required
    // if a required prop is missing do not set the assetsUrl and no plugin will be loaded
    if (!apiEndpoint || !assetsUrl) return
  }, [])

  return (
    <Container py={true} className="h-full">
      <PluginContainer />
    </Container>
  )
}

export default AppContent
