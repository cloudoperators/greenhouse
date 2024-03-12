import {
  DataGrid,
  DataGridHeadCell,
  DataGridRow,
  Icon,
} from "juno-ui-components"
import React from "react"
import { Cluster } from "../types/types"
import ClusterListItem from "./ClusterListItem"

interface ClusterListProps {
  clusters: Cluster[]
}

const ClusterList: React.FC<ClusterListProps> = (props: ClusterListProps) => {
  return (
    <>
      <DataGrid columns={5}>
        <DataGridRow>
          <DataGridHeadCell>
            <Icon icon="monitorHeart" />
          </DataGridHeadCell>
          <DataGridHeadCell>Name</DataGridHeadCell>
          <DataGridHeadCell>State</DataGridHeadCell>
          <DataGridHeadCell>Version</DataGridHeadCell>
          <DataGridHeadCell>Message</DataGridHeadCell>
        </DataGridRow>

        {props.clusters.map((cluster) => (
          <ClusterListItem key={cluster.metadata!.name!} cluster={cluster} />
        ))}
      </DataGrid>
    </>
  )
}

export default ClusterList
