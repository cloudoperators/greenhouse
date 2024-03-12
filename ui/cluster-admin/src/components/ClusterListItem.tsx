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
import useGetPluginConfigs from "../hooks/useGetPluginConfig"
import { getResourceStatusFromKubernetesConditions } from "../lib/utils/resourceStatus"
import useStore from "../store"
import { Cluster } from "../types/types"
import ResourceStatusIcon from "./ResourceStatusIcon"

interface ClusterListItemProps {
  cluster: Cluster
}

const ClusterListItem: React.FC<ClusterListItemProps> = (
  props: ClusterListItemProps
) => {
  const setClusterDetails = useStore((state) => state.setClusterDetails)
  const setClusterDetailPluginConfigs = useStore(
    (state) => state.setClusterDetailPluginConfigs
  )
  const setShowOnBoardCluster = useStore((state) => state.setShowOnBoardCluster)
  const { getPluginConfigsforCluster: getPluginConfigsforCluster } =
    useGetPluginConfigs()

  const setShowClusterDetails = useStore((state) => state.setShowClusterDetails)

  let clusterStatus = getResourceStatusFromKubernetesConditions(
    props.cluster.status?.statusConditions?.conditions ?? []
  )
  let message = clusterStatus.message?.substring(0, 66) ?? ""
  message += message.length > 50 ? "..." : ""

  const openDetails = () => {
    setClusterDetails(props.cluster)
    setShowClusterDetails(true)
    setShowOnBoardCluster(false)

    // only get plugin configs on click
    const pluginConfigs = getPluginConfigsforCluster(props.cluster)
    pluginConfigs.then((pluginConfigs) => {
      setClusterDetailPluginConfigs(pluginConfigs)
    })
  }

  return (
    <DataGridRow style={{ cursor: "pointer" }} onClick={() => openDetails()}>
      <DataGridCell>
        <ResourceStatusIcon status={clusterStatus} />
      </DataGridCell>
      <DataGridCell>{props.cluster.metadata!.name}</DataGridCell>
      <DataGridCell>{clusterStatus.state}</DataGridCell>
      <DataGridCell>{props.cluster.status?.kubernetesVersion}</DataGridCell>

      <DataGridCell>{message}</DataGridCell>
    </DataGridRow>
  )
}
export default ClusterListItem
