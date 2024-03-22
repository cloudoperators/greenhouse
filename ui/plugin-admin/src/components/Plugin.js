import React from "react"
import { DataGridRow, DataGridCell } from "juno-ui-components"
import { useGlobalsActions, useShowDetailsFor } from "./StoreProvider"

import { Icon } from "juno-ui-components"

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
