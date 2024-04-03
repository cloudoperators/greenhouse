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
  Icon,
  Stack,
} from "juno-ui-components"
import { usePluginConfig } from "./StoreProvider"
import Plugin from "./Plugin"

// Renders the list of plugins
const PluginList = () => {
  const pluginConfig = usePluginConfig()

  return (
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
              <div>No plugins found.</div>
            </Stack>
          </DataGridCell>
        </DataGridRow>
      )}
    </DataGrid>
  )
}

export default PluginList
