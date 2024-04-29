/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Juno contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState } from "react"
import { Container } from "juno-ui-components"
import { usePluginConfig } from "./StoreProvider"
import Plugin from "./Plugin"
import { MessagesProvider } from "messages-provider"

const PluginContainer = () => {
  const pluginConfig = usePluginConfig()

  return (
    <>
      {Object.keys(pluginConfig).map((key, index) => (
        <MessagesProvider key={index}>
          <Plugin config={pluginConfig[key]} />
        </MessagesProvider>
      ))}
    </>
  )
}

export default PluginContainer
