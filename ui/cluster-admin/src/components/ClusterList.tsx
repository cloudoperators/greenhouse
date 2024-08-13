/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import {
  DataGrid,
  DataGridHeadCell,
  DataGridRow,
  Icon,
} from "@cloudoperators/juno-ui-components"
import React from "react"
import { Cluster } from "../types/types"
import ClusterListItem from "./ClusterListItem"

interface ClusterListProps {
  clusters: Cluster[]
}

const ClusterList: React.FC<ClusterListProps> = (props: ClusterListProps) => {
  return (
    <>
      <DataGrid columns={5} minContentColumns={[0]} className="clusters">
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
