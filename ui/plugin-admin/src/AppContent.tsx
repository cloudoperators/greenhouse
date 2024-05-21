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
import PluginEdit from "./plugin-edit/PluginEdit"
import PluginDefinitionDetail from "./plugindefinitions/components/PluginDefinitionDetail"
import PluginDefinitionGrid from "./plugindefinitions/components/PluginDefinitionGrid"
import WelcomeView from "./plugindefinitions/components/WelcomeView"
import usePluginDefinitionsStore from "./plugindefinitions/store"
import PluginDetail from "./plugins/components/PluginDetail"
import PluginList from "./plugins/components/PluginList"
import useAPI from "./plugins/hooks/useAPI"
import SecretEdit from "./secrets/SecretEdit"
import SecretList from "./secrets/SecretList"

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
  const showEditForm = usePluginDefinitionsStore((state) => state.showEditForm)
  const secrets = usePluginDefinitionsStore((state) => state.secrets)
  const showSecretEdit = usePluginDefinitionsStore(
    (state) => state.showSecretEdit
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
        <Stack distribution="between">
          <TabList>
            <Tab>Available Plugins</Tab>
            <Tab>Enabled Plugins</Tab>
            <Tab>Secrets</Tab>
          </TabList>
          <Message variant={"warning"} text="feature in beta" />
        </Stack>

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
              {showEditForm && pluginDefinitionDetail && (
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
        <TabPanel>
          {loggedIn && !authError ? (
            <>
              {secrets.length > 0 && <SecretList secrets={secrets} />}
              {showSecretEdit && <SecretEdit />}
            </>
          ) : (
            <WelcomeView />
          )}
        </TabPanel>
      </MainTabs>
    </Container>
  )
}

export default AppContent
