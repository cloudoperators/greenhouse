/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

/*
 * SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import {
  DataGrid,
  DataGridHeadCell,
  DataGridRow,
  Icon,
} from "juno-ui-components"
import React from "react"
import { Cluster } from "../types/types"
import NodeListItem from "./NodeListItem"

interface NodeListProps {
  cluster: Cluster
}

const NodeList: React.FC<NodeListProps> = (props: NodeListProps) => {
  const nodeList = props.cluster.status?.nodes!

  return (
    <>
      <DataGrid columns={4}>
        <DataGridRow>
          <DataGridHeadCell>
            <Icon icon="monitorHeart" />
          </DataGridHeadCell>
          <DataGridHeadCell>Name</DataGridHeadCell>
          <DataGridHeadCell>State</DataGridHeadCell>
          <DataGridHeadCell>Message</DataGridHeadCell>
        </DataGridRow>

        {Object.keys(nodeList).map((key) => {
          const node = nodeList[key]
          return (
            <NodeListItem
              key={key}
              nodeName={key}
              nodeConditions={node.statusConditions?.conditions}
            />
          )
        })}
      </DataGrid>
    </>
  )
}

export default NodeList
