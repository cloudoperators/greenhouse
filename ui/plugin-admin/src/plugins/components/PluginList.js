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
  Message,
} from "@cloudoperators/juno-ui-components"
import { usePluginConfig, useGlobalsActions } from "./StoreProvider"
import Plugin from "./Plugin"

// Renders the list of plugins
const PluginList = () => {
  const pluginConfig = usePluginConfig()
  const { setPanel } = useGlobalsActions()

  const onShowPluginDefinition = () => {
    setPanel("showPluginDefinition")
  }

  return (
    <>
      <DataGridToolbar>
        <ButtonRow>
          <Button onClick={onShowPluginDefinition}>Add Plugin</Button>
        </ButtonRow>
      </DataGridToolbar>
      <DataGrid
        columns={5}
        cellVerticalAlignment="top"
        className="plugins"
        minContentColumns={[0, 4]}
      >
        {pluginConfig && (
          <DataGridRow>
            <DataGridHeadCell>
              <Icon icon="monitorHeart" />
            </DataGridHeadCell>
            <DataGridHeadCell>Name</DataGridHeadCell>
            <DataGridHeadCell>Cluster</DataGridHeadCell>
            <DataGridHeadCell>External Links</DataGridHeadCell>
            <DataGridHeadCell>Edit</DataGridHeadCell>
          </DataGridRow>
        )}
        {pluginConfig?.length > 0 ? (
          pluginConfig?.map((plugin) => {
            return <Plugin key={plugin?.metadata?.uid} plugin={plugin} />
          })
        ) : (
          <DataGridRow className="no-hover">
            <DataGridCell colSpan={4}>
              <Stack gap="3">
                <Icon icon="info" color="text-theme-info" />
                <div>
                  It seems that no plugins have been created yet. Do you want to
                  <a onClick={onShowPluginDefinition}> create</a> a new one?
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
