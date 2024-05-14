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

import useStore from "../store"

import {
  useShowDefinitionPanel,
  useGlobalsActions,
} from "../../plugins/components/StoreProvider"

// Renders the plugin details panel
const PluginDefinitionPanel = () => {
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

  const showDefinitionPanel = useShowDefinitionPanel()
  const { setShowDefinitionPanel } = useGlobalsActions()

  const setShowPluginDefinitionDetails = useStore(
    (state) => state.setShowPluginDefinitionDetails
  )

  const onShowDefinitionPanel = () => {
    setShowDefinitionPanel(false)
    setShowPluginDefinitionDetails(false)
  }

  return (
    <Panel
      opened={showDefinitionPanel || showPluginDefinitionDetails}
      onClose={onShowDefinitionPanel}
      size="large"
      heading="Add Plugin"
    >
      <PanelBody>
        {!showPluginDefinitionDetails && pluginDefinitions?.length > 0 && (
          <PluginDefinitionGrid pluginDefinitions={pluginDefinitions} />
        )}
        {showPluginDefinitionDetails && pluginDefinitionDetail && (
          <PluginDefinitionDetail pluginDefinition={pluginDefinitionDetail} />
        )}
        {showPluginEdit && pluginDefinitionDetail && (
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
