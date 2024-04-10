/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React from "react"
import { PluginDefinition } from "../types/types"
import PluginDefinitionTile from "./PluginDefinitionTile"

interface PluginGridProps {
  pluginDefinitions: PluginDefinition[]
}

const PluginGrid: React.FC<PluginGridProps> = (props: PluginGridProps) => {
  return (
    <>
      <div className="org-info p-8 mb-8 bg-theme-background-lvl-0">
        <div className="grid grid-cols-[repeat(auto-fit,_minmax(20rem,_1fr))] auto-rows-[minmax(8rem,_1fr)] gap-6 pt-8">
          {props.pluginDefinitions.map((plugin) => (
            <PluginDefinitionTile
              key={plugin.metadata!.name!}
              pluginDefinition={plugin}
            />
          ))}
        </div>
      </div>
    </>
  )
}

export default PluginGrid
