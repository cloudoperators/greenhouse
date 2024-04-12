/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React from "react"
import { DataGridRow, DataGridCell } from "juno-ui-components"
import { usePluginActions, useShowDetailsFor } from "./StoreProvider"

import { PluginConditionIcon } from "./PluginConditionIcon"

// renders a single plugin row
const Plugin = (props) => {
  const plugin = props.plugin
  const { setShowDetailsFor } = usePluginActions()
  const showDetailsFor = useShowDetailsFor()

  const showDetails = () => {
    showDetailsFor === plugin.id
      ? setShowDetailsFor(null)
      : setShowDetailsFor(plugin.id)
  }

  return (
    <DataGridRow
      key={plugin.id}
      onClick={showDetails}
      className={`cursor-pointer ${
        showDetailsFor === plugin.id ? "active" : ""
      } ${plugin?.disabled ? "text-theme-disabled" : ""} `}
    >
      <DataGridCell>
        <PluginConditionIcon plugin={plugin} />
      </DataGridCell>
      <DataGridCell>{plugin.name}</DataGridCell>
      <DataGridCell>
        {plugin.clusterName ? plugin.clusterName : <>&mdash;</>}
      </DataGridCell>
      <DataGridCell>
        {plugin.externalServicesUrls ? (
          plugin.externalServicesUrls?.map((url) => {
            return (
              <a
                href={url.url}
                target="_blank"
                rel="noreferrer"
                key={url.url}
                className={`mr-3 ${
                  plugin?.disabled ? "text-theme-link text-opacity-50" : ""
                }`}
              >
                {url.name}
              </a>
            )
          })
        ) : (
          <>&mdash;</>
        )}
      </DataGridCell>
    </DataGridRow>
  )
}

export default Plugin
