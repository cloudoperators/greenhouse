/*
 * SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import {
  Button,
  DataGridCell,
  DataGridHeadCell,
  DataGridRow,
  Stack,
} from "juno-ui-components"
import React from "react"
import { PluginConfig } from "../types/types"

interface ClusterPluginConfigListProps {
  // cluster: Cluster;
  pluginConfigs: PluginConfig[]
}

const ClusterPluginConfigList: React.FC<ClusterPluginConfigListProps> = (
  props: ClusterPluginConfigListProps
) => {
  let pluginConfigNames = ""

  props.pluginConfigs.map((pluginConfig: any) => {
    pluginConfigNames += pluginConfig.metadata.name + ", "
  })
  return (
    props.pluginConfigs.length > 0 && (
      <DataGridRow>
        <DataGridHeadCell>Enabled Plugins</DataGridHeadCell>
        <DataGridCell>
          <Stack gap="2" alignment="start" wrap={true}>
            {props.pluginConfigs.map((pluginConfig: any) => {
              return (
                <Button
                  style={{ cursor: "default" }}
                  size="small"
                  onClick={() =>
                    console.log(
                      "go to plugin config page of " +
                        pluginConfig.metadata.name
                    )
                  }
                >
                  {pluginConfig.metadata.name}
                </Button>
              )
            })}
          </Stack>
        </DataGridCell>
      </DataGridRow>
    )
  )
}

export default ClusterPluginConfigList
