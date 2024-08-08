/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { DataGridCell, DataGridRow } from "@cloudoperators/juno-ui-components"
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
