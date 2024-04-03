/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { useEffect, useState } from "react"
import { registerConsumer } from "url-state-provider"
import useStore from "../store"

const DEFAULT_KEY = "greenhouse-cluster-admin"
const SHOW_CLUSTER_DETAIL = "scd"
const SHOW_ONBOARD_CLUSTER = "soc"
const CLUSTER_DETAIL = "cd"

const useUrlState = (key: string): void => {
  const [isURLRead, setIsURLRead] = useState(false)
  const urlStateManager = registerConsumer(key || DEFAULT_KEY)

  // auth
  const loggedIn = useStore((state) => state.loggedIn)

  // globals
  const showClusterDetails = useStore((state) => state.showClusterDetails)
  const setShowClusterDetails = useStore((state) => state.setShowClusterDetails)
  const showOnboardCluster = useStore((state) => state.showOnBoardCluster)
  const setShowOnboardCluster = useStore((state) => state.setShowOnBoardCluster)
  const clusterDetails = useStore((state) => state.clusterDetails)
  const setClusterDetails = useStore((state) => state.setClusterDetails)
  const setClusterDetailPluginConfigs = useStore(
    (state) => state.setClusterDetailPluginConfigs
  )

  // Set initial state from URL (on login)
  useEffect(() => {
    /* !!!IMPORTANT!!!
      don't read the url if we are already read it or if we are not logged in!!!!!
    */
    if (isURLRead || !loggedIn) return
    console.log(
      `greenhouse-cluster-admin: (${
        key || DEFAULT_KEY
      }) setting up state from url:`,
      urlStateManager.currentState()
    )

    // READ the url state and set the state
    const newShowClusterDetail =
      urlStateManager.currentState()?.[SHOW_CLUSTER_DETAIL]
    const newShowOnboardCluster =
      urlStateManager.currentState()?.[SHOW_ONBOARD_CLUSTER]
    const newClusterDetail = urlStateManager.currentState()?.[CLUSTER_DETAIL]
    // SAVE the state
    if (newShowClusterDetail) setShowClusterDetails(newShowClusterDetail)
    if (newShowOnboardCluster) setShowOnboardCluster(newShowOnboardCluster)
    if (newClusterDetail) {
      setClusterDetails(newClusterDetail.cluster)
      setClusterDetailPluginConfigs(newClusterDetail.pluginConfigs)
    }
    setIsURLRead(true)
  }, [
    loggedIn,
    setShowClusterDetails,
    setShowOnboardCluster,
    setClusterDetails,
  ])

  // SYNC states to URL state
  useEffect(() => {
    // don't sync if we are not logged in OR URL ist not yet read
    if (!isURLRead || !loggedIn) return
    urlStateManager.push({
      [SHOW_CLUSTER_DETAIL]: showClusterDetails,
      [SHOW_ONBOARD_CLUSTER]: showOnboardCluster,
      [CLUSTER_DETAIL]: clusterDetails,
    })
  }, [loggedIn, showClusterDetails, showOnboardCluster, clusterDetails])
}

export default useUrlState
