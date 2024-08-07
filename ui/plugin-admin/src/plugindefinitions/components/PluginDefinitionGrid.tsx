/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React from "react"
import { PluginDefinition } from "../../../../types/types"
import PluginDefinitionTile from "./PluginDefinitionTile"
import { Container } from "@cloudoperators/juno-ui-components"

interface PluginDefinitionGridProps {
  pluginDefinitions: PluginDefinition[]
}

const PluginDefinitionGrid: React.FC<PluginDefinitionGridProps> = (
  props: PluginDefinitionGridProps
) => {
  return (
    <>
      <Container px={false} py>
        <div className="card-container grid gap-4 grid-cols-3">
          {props.pluginDefinitions.map((plugin) => (
            <PluginDefinitionTile
              key={plugin.metadata!.name!}
              pluginDefinition={plugin}
            />
          ))}
        </div>
      </Container>
    </>
  )
}

export default PluginDefinitionGrid
