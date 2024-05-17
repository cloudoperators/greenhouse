/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useEffect, useState } from "react"
import {
  CodeBlock,
  Container,
  DataGrid,
  DataGridRow,
  DataGridCell,
  DataGridHeadCell,
  JsonViewer,
  Panel,
  Pill,
  PanelBody,
  Stack,
  Tabs,
  TabList,
  Tab,
  TabPanel,
  Icon,
} from "juno-ui-components"

import PluginEdit from "../../plugin-edit/PluginEdit"
import PluginDefinitionDetail from "./PluginDefinitionDetail"
import PluginDefinitionGrid from "./PluginDefinitionGrid"

import usePluginDefinitionsStore from "../store"

import {
  useGlobalsActions,
  usePanel,
} from "../../plugins/components/StoreProvider"

// Renders the plugin details panel
const PluginDefinitionPanel = () => {
  const pluginDefinitions = usePluginDefinitionsStore(
    (state) => state.pluginDefinitions
  )
  const panel = usePanel()

  const pluginDefinitionDetail = usePluginDefinitionsStore(
    (state) => state.pluginDefinitionDetail
  )

  const { setPanel } = useGlobalsActions()

  const onCloseDefinitionPanel = () => {
    setPanel(null)
  }

  return (
    <Panel
      opened={[
        "showPluginDefinition",
        "showPluginDefinitionDetail",
        "editPlugin",
      ].includes(panel)}
      onClose={onCloseDefinitionPanel}
      size="large"
      heading="Add Plugin"
    >
      <PanelBody>
        {panel === "showPluginDefinition" && pluginDefinitions?.length > 0 && (
          <PluginDefinitionGrid pluginDefinitions={pluginDefinitions} />
        )}
        {panel === "showPluginDefinitionDetail" && pluginDefinitionDetail && (
          <PluginDefinitionDetail pluginDefinition={pluginDefinitionDetail} />
        )}
        {panel === "editPlugin" && pluginDefinitionDetail && (
          <PluginEdit pluginDefinition={pluginDefinitionDetail} />
        )}
      </PanelBody>
    </Panel>
  )
}

export default PluginDefinitionPanel

/*


        <Stack gap="2">
          <span>
            {props.pluginDefinition.spec?.displayName ??
              (props.pluginDefinition.metadata?.name || "Not found")}
          </span>
        </Stack>

        */
