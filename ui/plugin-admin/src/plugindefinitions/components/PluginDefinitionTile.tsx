/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { Icon, Stack } from "juno-ui-components"
import React from "react"
import { PluginDefinition } from "../types/types"
import useStore from "../store"

interface PluginDefinitionTileProps {
  pluginDefinition: PluginDefinition
}
const allowedIconFileEndings = [".png"]
const PluginDefinitionTile: React.FC<PluginDefinitionTileProps> = (
  props: PluginDefinitionTileProps
) => {
  const setShowPluginDefinitionDetails = useStore(
    (state) => state.setShowPluginDefinitionDetails
  )
  const setPluginDefinitionDetail = useStore(
    (state) => state.setPluginDefinitionDetail
  )

  let iconUrl: string | undefined
  if (
    allowedIconFileEndings.some((ending) =>
      props.pluginDefinition.spec?.icon?.endsWith(ending)
    )
  ) {
    iconUrl = props.pluginDefinition.spec?.icon
  } else {
    iconUrl = undefined
  }

  const openPluginDefinitionDetails = () => {
    setShowPluginDefinitionDetails(true)
    setPluginDefinitionDetail(props.pluginDefinition)
  }
  return (
    <Stack
      direction="vertical"
      alignment="center"
      distribution="between"
      className="org-info-item bg-theme-background-lvl-1 p-4"
      style={{ cursor: "pointer" }}
      onClick={openPluginDefinitionDetails}
    >
      <h2 className="text-lg font-bold">
        {props.pluginDefinition.spec?.displayName ??
          props.pluginDefinition.metadata?.name}
      </h2>

      {!iconUrl && (
        <Icon
          icon={props.pluginDefinition.spec?.icon ?? "autoAwesomeMosaic"}
          size="100"
        />
      )}
      {iconUrl && (
        <img
          className="filtered"
          src={iconUrl}
          alt="icon"
          width="100"
          height="100"
        />
      )}
      <p>{props.pluginDefinition.spec?.description}</p>

      <div className="bg-theme-background-lvl-4 py-2 px-3 inline-flex">
        {props.pluginDefinition.spec?.version}
      </div>
    </Stack>
  )
}

export default PluginDefinitionTile
