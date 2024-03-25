/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useEffect } from "react"
import { Container } from "juno-ui-components"
import useAPI from "./hooks/useAPI"
import PluginList from "./components/PluginList"
import PluginDetail from "./components/PluginDetail"

const AppContent = () => {
  const { getPlugins } = useAPI()

  useEffect(() => {
    if (!getPlugins) return
    const plugins = getPlugins()
    console.log("getPlugins", plugins)
  }, [getPlugins])

  return (
    <Container>
      <PluginDetail />
      <PluginList />
    </Container>
  )
}

export default AppContent
