/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { Container, MainTabs, Tab, TabList, TabPanel } from "juno-ui-components"
import { useEffect } from "react"
import PluginEdit from "./plugin-edit/PluginEdit"
import PluginDefinitionDetail from "./plugindefinitions/components/PluginDefinitionDetail"
import PluginDefinitionGrid from "./plugindefinitions/components/PluginDefinitionGrid"
import WelcomeView from "./plugindefinitions/components/WelcomeView"
import usePluginDefinitionsStore from "./plugindefinitions/store"
import PluginDetail from "./plugins/components/PluginDetail"
import PluginList from "./plugins/components/PluginList"
import useAPI from "./plugins/hooks/useAPI"

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
