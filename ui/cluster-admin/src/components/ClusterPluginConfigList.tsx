/*
 * Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
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
