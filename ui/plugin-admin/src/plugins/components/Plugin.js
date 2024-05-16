/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React from "react"
import { DataGridRow, DataGridCell, Icon } from "juno-ui-components"
import { usePluginActions, useShowDetailsFor } from "./StoreProvider"
import useStore from "../../plugindefinitions/store"
import { buildExternalServicesUrls } from "./buildExternalServicesUrls"

import { PluginConditionIcon } from "./PluginConditionIcon"

// renders a single plugin row
const Plugin = (props) => {
  const plugin = props.plugin
  const { setShowDetailsFor } = usePluginActions()
  const showDetailsFor = useShowDetailsFor()

  const setPluginToEdit = useStore((state) => state.setPluginToEdit)
  const setShowPluginEdit = useStore((state) => state.setShowPluginEdit)
  const setShowPluginDefinitionDetails = useStore(
    (state) => state.setShowPluginDefinitionDetails
  )
  const setIsEditMode = useStore((state) => state.setIsPluginEditMode)

  const showDetails = () => {
    showDetailsFor === plugin.metadata.uid
      ? setShowDetailsFor(null)
      : setShowDetailsFor(plugin.metadata.uid)
  }

  const onPluginClick = () => {
    setPluginToEdit(plugin)
    setShowPluginDefinitionDetails(true)
    setShowPluginEdit(true)
    setIsEditMode(true)
  }

  return (
    <DataGridRow
      key={plugin?.metadata?.uid}
      onClick={showDetails}
      className={`cursor-pointer ${
        showDetailsFor === plugin?.metadata?.uid ? "active" : ""
      } ${plugin?.spec?.disabled ? "text-theme-disabled" : ""} `}
    >
      <DataGridCell>
        <PluginConditionIcon plugin={plugin} />
      </DataGridCell>
      <DataGridCell>
        {plugin?.spec?.displayName || plugin?.metadata?.name}
      </DataGridCell>
      <DataGridCell>
        {plugin?.spec?.clusterName ? plugin?.spec?.clusterName : <>&mdash;</>}
      </DataGridCell>
      <DataGridCell>
        {plugin?.status?.exposedServices ? (
          buildExternalServicesUrls(plugin.status.exposedServices).map(
            (url) => {
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
            }
          )
        ) : (
          <>&mdash;</>
        )}
      </DataGridCell>
      <DataGridCell>
        <Icon color="jn-global-text" icon="edit" onClick={onPluginClick} />
      </DataGridCell>
    </DataGridRow>
  )
}

export default Plugin
