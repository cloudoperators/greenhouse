import React from "react"
import { DataGridRow, DataGridCell } from "juno-ui-components"
import { useGlobalsActions } from "./StoreProvider"

const Plugin = (props) => {
  const plugin = props.plugin
  const { setShowDetailsFor } = useGlobalsActions()

  const showDetails = () => {
    setShowDetailsFor(plugin)
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
    <DataGridRow key={plugin.name} onClick={showDetails}>
      <DataGridCell>
        <p>{plugin.name}</p>
      </DataGridCell>
      <DataGridCell>
        <p>{plugin.clusterName}</p>
      </DataGridCell>
      <DataGridCell>
        {plugin.externalServicesUrls?.map((url) => {
          return (
            <p>
              <a href={url.url} target="_blank" rel="noreferrer">
                {url.name}
              </a>
            </p>
          )
        })}
      </DataGridCell>
      <DataGridCell>
        <p> TRUE</p>
      </DataGridCell>
    </DataGridRow>
  )
}

export default Plugin
