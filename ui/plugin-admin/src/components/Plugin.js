import React from "react"
import { DataGridRow, DataGridCell } from "juno-ui-components"
import { useGlobalsActions, useShowDetailsFor } from "./StoreProvider"

import { Icon } from "juno-ui-components"

const Plugin = (props) => {
  const plugin = props.plugin
  const { setShowDetailsFor } = useGlobalsActions()
  const showDetailsFor = useShowDetailsFor()

  const showDetails = () => {
    setShowDetailsFor(plugin.id)
  }

  const buildExternalServicesUrls = () => {
    // extract url and the name from the object and create a link
    if (!plugin.externalServicesUrls) return null

    const links = []
    for (const url in plugin.externalServicesUrls) {
      const currentObject = plugin.externalServicesUrls[url]

      links.push(
        <p>
          <a href={url} target="_blank" rel="noreferrer">
            {currentObject.name ? currentObject.name : url}
          </a>
        </p>
      )
    }
    return links
  }

  return (
    <DataGridRow
      key={plugin.id}
      onClick={showDetails}
      className={`cursor-pointer ${
        showDetailsFor === plugin.id ? "active" : ""
      }`}
    >
      <DataGridCell>
        <p>{plugin.name}</p>
      </DataGridCell>
      <DataGridCell>
        <p>{plugin.clusterName}</p>
      </DataGridCell>
      <DataGridCell>
        {plugin.externalServicesUrls?.map((url) => {
          return (
            <a href={url.url} target="_blank" rel="noreferrer" key={url.url}>
              {url.name + " "}
            </a>
          )
        })}
      </DataGridCell>
      <DataGridCell>
        <p>
          <Icon
            icon={plugin.readyStatus?.icon}
            color={plugin.readyStatus?.color}
          />
        </p>
      </DataGridCell>
    </DataGridRow>
  )
}

export default Plugin
