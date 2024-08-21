/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { DataGridCell, DataGridRow } from "@cloudoperators/juno-ui-components"
import React from "react"
import useGetPlugins from "../hooks/useGetPlugin"
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
  const clusterDetails = useStore((state) => state.clusterDetails)

  const setClusterDetailPlugins = useStore(
    (state) => state.setClusterDetailPlugins
  )
  const setShowOnBoardCluster = useStore((state) => state.setShowOnBoardCluster)
  const { getPluginsforCluster: getPluginsforCluster } = useGetPlugins()

  const setShowClusterDetails = useStore((state) => state.setShowClusterDetails)
  const showClusterDetails = useStore((state) => state.showClusterDetails)

  let clusterStatus = getResourceStatusFromKubernetesConditions(
    props.cluster.status?.statusConditions?.conditions ?? []
  )
  let message = clusterStatus.message?.substring(0, 66) ?? ""
  message += message.length > 50 ? "..." : ""

  const openDetails = () => {
    setClusterDetails(props.cluster)

    // set showClusterDetails to false if the same cluster is clicked again.
    clusterDetails?.cluster?.metadata?.name ===
      props?.cluster?.metadata?.name && showClusterDetails
      ? setShowClusterDetails(false)
      : setShowClusterDetails(true)

    setShowOnBoardCluster(false)

    // only get plugin configs on click
    const plugins = getPluginsforCluster(props.cluster)
    plugins.then((plugins) => {
      setClusterDetailPlugins(plugins)
    })
  }

  return (
    <DataGridRow
      className={`cursor-pointer ${
        clusterDetails?.cluster?.metadata?.name ===
          props?.cluster?.metadata?.name && showClusterDetails
          ? "active"
          : ""
      }`}
      onClick={() => openDetails()}
    >
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
