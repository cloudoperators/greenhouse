/*
 * Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
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
