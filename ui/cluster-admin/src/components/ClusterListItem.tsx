/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

/*
 * SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
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
