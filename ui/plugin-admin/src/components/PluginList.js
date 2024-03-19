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

const PluginList = () => {
  const pluginConfig = usePluginConfig()

  return (
    <DataGrid columns={4} cellVerticalAlignment="top" className="plugins">
      {pluginConfig && (
        <DataGridRow>
          <DataGridHeadCell>Name</DataGridHeadCell>
          <DataGridHeadCell>Cluster</DataGridHeadCell>
          <DataGridHeadCell>External Links</DataGridHeadCell>
          <DataGridHeadCell>Ready</DataGridHeadCell>
        </DataGridRow>
      )}
      {pluginConfig?.length > 0 ? (
        pluginConfig?.map((plugin) => {
          return <Plugin key={plugin.id} plugin={plugin} />
        })
      ) : (
        <DataGridRow className="no-hover">
          <DataGridCell colSpan={3}>
            <Stack gap="3">
              <Icon icon="info" color="text-theme-info" />
              <div>We couldn't find anything.</div>
            </Stack>
          </DataGridCell>
        </DataGridRow>
      )}
    </DataGrid>
  )
}

export default PluginList
