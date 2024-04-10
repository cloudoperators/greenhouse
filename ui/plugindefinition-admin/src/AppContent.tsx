/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { Container } from "juno-ui-components"
import PluginDefinitionDetail from "./components/PluginDefinitionDetail"
import PluginGrid from "./components/PluginGrid"
import WelcomeView from "./components/WelcomeView"
import useStore from "./store"
import PluginEdit from "./components/plugin-edit/PluginEdit"

const AppContent = () => {
  const pluginDefinitions = useStore((state) => state.pluginDefinitions)
  const showPluginDefinitionDetails = useStore(
    (state) => state.showPluginDefinitionDetails
  )
  const pluginDefinitionDetail = useStore(
    (state) => state.pluginDefinitionDetail
  )
  const showPluginEdit = useStore((state) => state.showPluginDefinitionEdit)
  const auth = useStore((state) => state.auth)
  const authError = auth?.error
  const loggedIn = useStore((state) => state.loggedIn)

  return (
    <Container>
      {loggedIn && !authError ? (
        <>
          {pluginDefinitions.length > 0 && (
            <PluginGrid pluginDefinitions={pluginDefinitions} />
          )}
          {showPluginDefinitionDetails && pluginDefinitionDetail && (
            <PluginDefinitionDetail pluginDefinition={pluginDefinitionDetail} />
          )}
          {showPluginEdit && pluginDefinitionDetail && (
            <PluginEdit pluginDefinition={pluginDefinitionDetail} />
          )}
        </>
      ) : (
        <WelcomeView />
      )}
    </Container>
  )
}

export default AppContent
