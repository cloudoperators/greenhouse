/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import {
  Container,
  MainTabs,
  Message,
  Stack,
  Tab,
  TabList,
  TabPanel,
} from "juno-ui-components"
import { useEffect } from "react"
import WelcomeView from "./plugindefinitions/components/WelcomeView"
import usePluginDefinitionsStore from "./plugindefinitions/store"
import PluginDetail from "./plugins/components/PluginDetail"
import PluginList from "./plugins/components/PluginList"
import PluginDefinitionPanel from "./plugindefinitions/components/PluginDefinitionPanel"
import useAPI from "./plugins/hooks/useAPI"
import SecretEdit from "./secrets/SecretEdit"
import SecretList from "./secrets/SecretList"

const AppContent = () => {
  const { getPlugins } = useAPI()

  const auth = usePluginDefinitionsStore((state) => state.auth)
  const authError = auth?.error
  const loggedIn = usePluginDefinitionsStore((state) => state.loggedIn)

  useEffect(() => {
    if (!getPlugins) return
    const plugins = getPlugins()
    console.log("getPlugins", plugins)
  }, [getPlugins])

  return (
    <Container>
      {loggedIn && !authError ? (
        <>
          <PluginDefinitionPanel />
          <PluginDetail />
          <PluginList />
        </>
      ) : (
        <WelcomeView />
      )}
    </Container>
  )
}

export default AppContent
