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
import useStore, { EditFormState } from "../store"
import { useGlobalsActions } from "../../plugins/components/StoreProvider"

interface PluginListProps {
  plugins: Plugin[]
}

const PluginList: React.FC<PluginListProps> = (props: PluginListProps) => {
  const setShowPluginEdit = useStore((state) => state.setShowEditForm)
  const setShowPluginDefinitionDetails = useStore(
    (state) => state.setShowPluginDefinitionDetails
  )
  const setEditFormState = useStore((state) => state.setEditFormState)
  const setEditFormData = useStore((state) => state.setEditFormData)

  const { setPanel } = useGlobalsActions()

  const onPluginClick = (plugin: Plugin) => {
    setEditFormData({
      metadata: plugin.metadata!,
      spec: plugin.spec!,
    })
    setShowPluginDefinitionDetails(false)
    setShowPluginEdit(true)
    setEditFormState(EditFormState.PLUGIN_EDIT)
    setPanel("editPlugin")
  }
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
