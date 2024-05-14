/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { Icon, Stack } from "juno-ui-components"
import React from "react"
import { PluginDefinition } from "../../../../types/types"
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

  const cardHeaderCss = `
  font-bold
  text-lg
  `

  const cardCss = `
  card
  bg-theme-background-lvl-1
  hover:bg-theme-background-lvl-2
  rounded
  p-8
  h-full
  w-full
  cursor-pointer
  `

  return (
    <div className={cardCss} onClick={openPluginDefinitionDetails}>
      <Stack direction="vertical" alignment="start" className="h-full w-full">
        <div className={cardHeaderCss}>
          {props.pluginDefinition.spec?.displayName ??
            props.pluginDefinition.metadata?.name}
        </div>
        <div className="mt-4">{props.pluginDefinition.spec?.description}</div>

        <div className="mt-auto w-full">
          <Stack alignment="center">
            <div className="w-full">{props.pluginDefinition.spec?.version}</div>

            {!iconUrl && (
              <Icon
                icon={props.pluginDefinition.spec?.icon ?? "autoAwesomeMosaic"}
                className="filtered fill-current text-theme-high"
                size="50"
              />
            )}
            {iconUrl && (
              <img
                className="filtered fill-current "
                src={iconUrl}
                alt="icon"
                width="50"
              />
            )}
          </Stack>
        </div>
      </Stack>
    </div>
  )
}

export default PluginDefinitionTile
