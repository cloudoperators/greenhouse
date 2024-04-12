/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useEffect } from "react"
import { Container, MainTabs, TabList, Tab, TabPanel } from "juno-ui-components"
import useAPI from "./plugins/hooks/useAPI"
import PluginList from "./plugins/components/PluginList"
import PluginDetail from "./plugins/components/PluginDetail"
import usePluginDefinitionsStore from "./plugindefinitions/store"
import PluginDefinitionGrid from "./plugindefinitions/components/PluginDefinitionGrid"
import PluginDefinitionDetail from "./plugindefinitions/components/PluginDefinitionDetail"
import WelcomeView from "./plugindefinitions/components/WelcomeView"
import PluginEdit from "./plugin-edit/PluginEdit"

const AppContent = () => {
  const { getPlugins } = useAPI()

  const pluginDefinitions = usePluginDefinitionsStore(
    (state) => state.pluginDefinitions
  )
  const showPluginDefinitionDetails = usePluginDefinitionsStore(
    (state) => state.showPluginDefinitionDetails
  )
  const pluginDefinitionDetail = usePluginDefinitionsStore(
    (state) => state.pluginDefinitionDetail
  )
  const showPluginEdit = usePluginDefinitionsStore(
    (state) => state.showPluginDefinitionEdit
  )
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
      <MainTabs>
        <TabList>
          <Tab>Available Plugins</Tab>
          <Tab>Enabled Plugins</Tab>
        </TabList>
        <TabPanel>
          {loggedIn && !authError ? (
            <>
              {pluginDefinitions?.length > 0 && (
                <PluginDefinitionGrid pluginDefinitions={pluginDefinitions} />
              )}
              {showPluginDefinitionDetails && pluginDefinitionDetail && (
                <PluginDefinitionDetail
                  pluginDefinition={pluginDefinitionDetail}
                />
              )}
              {showPluginEdit && pluginDefinitionDetail && (
                <PluginEdit pluginDefinition={pluginDefinitionDetail} />
              )}
            </>
          ) : (
            <WelcomeView />
          )}
        </TabPanel>
        <TabPanel>
          <PluginDetail />
          <PluginList />
        </TabPanel>
      </MainTabs>
    </Container>
  )
}

export default AppContent
