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
import { Plugin } from "../../../../types/types"
import useStore from "../store"
import { useGlobalsActions } from "../../plugins/components/StoreProvider"

interface PluginListProps {
  plugins: Plugin[]
}

const PluginList: React.FC<PluginListProps> = (props: PluginListProps) => {
  const setPluginToEdit = useStore((state) => state.setPluginToEdit)

  const { setPanel } = useGlobalsActions()

  const onPluginClick = React.useCallback(
    (plugin: Plugin) => {
      setPluginToEdit(plugin)
      setPanel("editPlugin")
    },
    [setPluginToEdit, setPanel]
  )

  return (
    props.plugins.length > 0 && (
      <DataGridRow>
        <DataGridHeadCell>Enabled Plugins</DataGridHeadCell>
        <DataGridCell>
          <Stack gap="2" alignment="start" wrap={true}>
            {props.plugins.map((plugin: Plugin) => {
              return (
                <Button
                  key={plugin.metadata!.name}
                  size="small"
                  onClick={() => {
                    onPluginClick(plugin)
                  }}
                >
                  {plugin.metadata!.name}
                </Button>
              )
            })}
          </Stack>
        </DataGridCell>
      </DataGridRow>
    )
  )
}

export default PluginList
