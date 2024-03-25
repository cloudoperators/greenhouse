/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React from "react"
import { DataGridRow, DataGridCell } from "juno-ui-components"
import { useGlobalsActions, useShowDetailsFor } from "./StoreProvider"

import { Icon } from "juno-ui-components"

// renders a single plugin row
const Plugin = (props) => {
  const plugin = props.plugin
  const { setShowDetailsFor } = useGlobalsActions()
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
        <p>{plugin.name}</p>
      </DataGridCell>
      <DataGridCell>
        {plugin.clusterName ? <p>{plugin.clusterName}</p> : <p>&mdash;</p>}
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
                className={`${
                  plugin?.disabled ? "text-theme-link text-opacity-50" : ""
                } `}
              >
                {url.name + " "}
              </a>
            )
          })
        ) : (
          <p>&mdash;</p>
        )}
      </DataGridCell>
      <DataGridCell>
        <p>
          {plugin.disabled ? (
            <Icon color="jn-global-text" icon="error" title="disabled" />
          ) : (
            <Icon
              icon={plugin.readyStatus?.icon}
              color={plugin.readyStatus?.color}
            />
          )}
        </p>
      </DataGridCell>
    </DataGridRow>
  )
}

export default Plugin
