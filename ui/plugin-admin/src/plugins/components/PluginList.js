/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React from "react"
import {
  DataGrid,
  DataGridHeadCell,
  DataGridRow,
  DataGridCell,
  DataGridToolbar,
  Button,
  ButtonRow,
  Icon,
  Stack,
} from "juno-ui-components"
import { usePluginConfig, useGlobalsActions } from "./StoreProvider"
import Plugin from "./Plugin"

// Renders the list of plugins
const PluginList = () => {
  const pluginConfig = usePluginConfig()
  const { setShowDefinitionPanel } = useGlobalsActions()

  const onShowDefinitionPanel = () => {
    setShowDefinitionPanel(true)
  }

  return (
    <>
      <DataGridToolbar>
        <ButtonRow>
          <Button onClick={onShowDefinitionPanel}>Add Plugin</Button>
        </ButtonRow>
      </DataGridToolbar>
      <DataGrid
        columns={4}
        cellVerticalAlignment="top"
        className="plugins"
        minContentColumns={[0]}
      >
        {pluginConfig && (
          <DataGridRow>
            <DataGridHeadCell>
              <Icon icon="monitorHeart" />
            </DataGridHeadCell>
            <DataGridHeadCell>Name</DataGridHeadCell>
            <DataGridHeadCell>Cluster</DataGridHeadCell>
            <DataGridHeadCell>External Links</DataGridHeadCell>
          </DataGridRow>
        )}
        {pluginConfig?.length > 0 ? (
          pluginConfig?.map((plugin) => {
            return <Plugin key={plugin.id} plugin={plugin} />
          })
        ) : (
          <DataGridRow className="no-hover">
            <DataGridCell colSpan={4}>
              <Stack gap="3">
                <Icon icon="info" color="text-theme-info" />
                <div>
                  It seems that no plugins have been created yet. Do you want to
                  <a onClick={onShowDefinitionPanel}> create</a> a new one?
                </div>
              </Stack>
            </DataGridCell>
          </DataGridRow>
        )}
      </DataGrid>
    </>
  )
}

export default PluginList
