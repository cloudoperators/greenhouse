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
  Button,
  Container,
  DataGridToolbar,
  ButtonRow,
} from "juno-ui-components"
import ClusterDetail from "./components/ClusterDetail"
import ClusterList from "./components/ClusterList"
import DownloadKubeConfig from "./components/DownloadKubeConfig"
import OnBoardCluster from "./components/OnBoardCluster"
import WelcomeView from "./components/WelcomeView"
import useNamespace from "./hooks/useNamespace"
import useStore from "./store"

const AppContent = () => {
  const clusters = useStore((state) => state.clusters)
  const clusterDetails = useStore((state) => state.clusterDetails)
  const showClusterDetails = useStore((state) => state.showClusterDetails)
  const showOnBoardCluster = useStore((state) => state.showOnBoardCluster)
  const showDownloadKubeConfig = useStore(
    (state) => state.showDownloadKubeConfig
  )
  const auth = useStore((state) => state.auth)
  const authError = auth?.error
  const expiryTimestamp = auth?.parsed.expiresAt
  const { namespace } = useNamespace()
  const apiEndpoint = useStore((state) => state.endpoint)
  const loggedIn = useStore((state) => state.loggedIn)
  const setShowOnBoardCluster = useStore((state) => state.setShowOnBoardCluster)
  const setShowClusterDetails = useStore((state) => state.setShowClusterDetails)
  const setShowDownloadKubeConfig = useStore(
    (state) => state.setShowDownloadKubeConfig
  )

  const openOnBoardCluster = () => {
    setShowOnBoardCluster(true)
    setShowClusterDetails(false)
    setShowDownloadKubeConfig(false)
  }

  const openShowDownloadKubeConfig = () => {
    setShowOnBoardCluster(false)
    setShowClusterDetails(false)
    setShowDownloadKubeConfig(true)
  }

  return (
    <Container>
      {loggedIn && !authError ? (
        <>
          <DataGridToolbar>
            <ButtonRow>
              <Button
                icon="openInBrowser"
                label="Access greenhouse cluster"
                onClick={() => openShowDownloadKubeConfig()}
              />
              <Button
                icon="addCircle"
                label="Onboard Cluster"
                onClick={() => openOnBoardCluster()}
              />
            </ButtonRow>
          </DataGridToolbar>

          {showOnBoardCluster && <OnBoardCluster />}
          {showDownloadKubeConfig && (
            <DownloadKubeConfig
              namespace={namespace}
              token={auth?.JWT}
              endpoint={apiEndpoint}
              expiry={expiryTimestamp}
            />
          )}
          {clusters.length > 0 && <ClusterList clusters={clusters} />}
          {showClusterDetails && clusterDetails.cluster && <ClusterDetail />}
        </>
      ) : (
        <WelcomeView />
      )}
    </Container>
  )
}

export default AppContent
