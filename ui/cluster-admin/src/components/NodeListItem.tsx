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

import { DataGridCell, DataGridRow } from "juno-ui-components"
import React from "react"
import {
  getResourceStatusFromKubernetesConditions,
  resourceStatusUnknown,
} from "../lib/utils/resourceStatus"
import ResourceStatusIcon from "./ResourceStatusIcon"
import { KubernetesCondition } from "../types/types"

interface NodeListItemProps {
  nodeName: string
  nodeConditions?: KubernetesCondition[]
}

const NodeListItem: React.FC<NodeListItemProps> = (
  props: NodeListItemProps
) => {
  const nodeStatus = props.nodeConditions
    ? getResourceStatusFromKubernetesConditions(props.nodeConditions)
    : resourceStatusUnknown
  return (
    <DataGridRow>
      <DataGridCell>
        <ResourceStatusIcon status={nodeStatus} />
      </DataGridCell>
      <DataGridCell>{props.nodeName}</DataGridCell>
      <DataGridCell>{nodeStatus.state}</DataGridCell>
      <DataGridCell>{nodeStatus.message}</DataGridCell>
    </DataGridRow>
  )
}
export default NodeListItem
