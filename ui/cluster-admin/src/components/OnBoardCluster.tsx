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

import { Markup } from "interweave"
import { Container, Panel, PanelBody, Stack } from "juno-ui-components"
import React from "react"
import useStore from "../store"
import useNamespace from "../hooks/useNamespace"

const ClusterDetail: React.FC<any> = () => {
  const { namespace } = useNamespace()
  const showOnBoardCluster = useStore((state) => state.showOnBoardCluster)
  const setShowOnBoardCluster = useStore((state) => state.setShowOnBoardCluster)

  const onPanelClose = () => {
    setShowOnBoardCluster(false)
  }

  const markup = `
  <p>Download the latest <i>greenhousectl</i> binary from <a target='_blank' href='https://github.com/cloudoperators/greenhouse/releases'>here</a>. </p>
  <p>Place a valid kubeconfig file for the cluster to be onboarded in <code>/path/to/kubeconfig</code> </p>
  <p>Execute the following command to onboard your cluster: <br><br>
  <code>greenhousectl cluster bootstrap --org ${namespace} --bootstrap-kubeconfig /path/to/kubeconfig</code> </p>
  <br>
  <p>For more information, visit <a target='blank' href='https://documentation.greenhouse.global.cloud.sap/docs/user-guides/cluster/onboarding/'>the docs</a> </p>
  `

  return (
    <Panel
      heading="How to onboard a cluster"
      opened={!!showOnBoardCluster}
      onClose={onPanelClose}
    >
      <PanelBody>
        <Container px={false} py>
          <Stack distribution="center" alignment="center" wrap={true}>
            <Markup content={markup} />
          </Stack>
        </Container>
      </PanelBody>
    </Panel>
  )
}

export default ClusterDetail
