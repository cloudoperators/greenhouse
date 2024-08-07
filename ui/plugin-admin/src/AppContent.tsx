/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { Container } from "@cloudoperators/juno-ui-components"
import { Messages } from "@cloudoperators/juno-messages-provider"
import { useEffect } from "react"
import PluginDefinitionPanel from "./plugindefinitions/components/PluginDefinitionPanel"
import WelcomeView from "./plugindefinitions/components/WelcomeView"
import usePluginDefinitionsStore from "./plugindefinitions/store"
import PluginDetail from "./plugins/components/PluginDetail"
import PluginList from "./plugins/components/PluginList"
import useAPI from "./plugins/hooks/useAPI"

const AppContent = () => {
  const { getPlugins } = useAPI()

  const auth = usePluginDefinitionsStore((state) => state.auth)
  const authError = auth?.error
  const loggedIn = usePluginDefinitionsStore((state) => state.loggedIn)

  useEffect(() => {
    if (!getPlugins) return
    const plugins = getPlugins()
  }, [getPlugins])

  return (
    <Container py>
      {loggedIn && !authError ? (
        <>
          <PluginDefinitionPanel />
          <PluginDetail />
          <Messages />
          <PluginList />
        </>
      ) : (
        <WelcomeView />
      )}
    </Container>
  )
}

export default AppContent
