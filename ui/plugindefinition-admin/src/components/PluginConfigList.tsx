/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
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

interface PluginConfigListProps {
  pluginConfigs: PluginConfig[]
}

const PluginConfigList: React.FC<PluginConfigListProps> = (
  props: PluginConfigListProps
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
                  key={pluginConfig.metadata.name}
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

export default PluginConfigList
