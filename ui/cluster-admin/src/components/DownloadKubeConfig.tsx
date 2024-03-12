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
  CodeBlock,
  Container,
  Button,
  Panel,
  PanelBody,
  Stack,
  Code,
} from "juno-ui-components"
import React from "react"
import {
  ENDPOINT_IDENTIFIER,
  KUBECONFIGTEMPLATE,
  NAMESPACE_IDENTIFIER,
  TOKEN_IDENTIFIER,
} from "../assets/kubeconfig-template"
import humanizedTimePeriodToNow from "../lib/utils/humanizedTimePeriodToNow"
import useStore from "../store"

interface DownloadKubeConfigProps {
  namespace: string
  endpoint: string
  token: string
  expiry: string
}

const DownloadKubeConfig: React.FC<DownloadKubeConfigProps> = (
  props: DownloadKubeConfigProps
) => {
  const setShowDownloadKubeConfig = useStore(
    (state) => state.setShowDownloadKubeConfig
  )
  const showDownloadKubeConfig = useStore(
    (state) => state.showDownloadKubeConfig
  )

  const kubeConfigTemplate = KUBECONFIGTEMPLATE
  const kubeConfig = kubeConfigTemplate
    .replace(TOKEN_IDENTIFIER, props.token)
    .replace(NAMESPACE_IDENTIFIER, props.namespace)
    .replace(ENDPOINT_IDENTIFIER, props.endpoint)
  const kubeconfigFile = new Blob([kubeConfig], { type: "text/plain" })
  const kubeconfigFileUrl = URL.createObjectURL(kubeconfigFile)

  const onPanelClose = () => {
    setShowDownloadKubeConfig(false)
  }

  return (
    <Panel
      heading="greenhouse Kubeconfig"
      opened={!!showDownloadKubeConfig}
      onClose={onPanelClose}
    >
      <PanelBody>
        <Container px={false} py>
          <Stack direction="vertical" gap="3" wrap={true}>
            <p>
              This is a kubeconfig generated based on your personal access
              permissions in the greenhouse cluster. These will be limited to
              your organizations namespace. Only organization admins have write
              permissions on resources.
            </p>

            <CodeBlock
              heading="greenhouse kubeconfig"
              size="small"
              content={kubeConfig}
            />
            <a
              href={kubeconfigFileUrl}
              download="greenhouse.kubeconfig"
              target="_blank"
            >
              <Button icon="download" label="Download kubeconfig" />
            </a>
            <p>
              The token used for authentication is valid for another{" "}
              <strong>{humanizedTimePeriodToNow(props.expiry)}</strong>
            </p>
            <p>
              Download the kubeconfig file and use it with kubectl to access the
              cluster:
            </p>
            <Code
              content={`kubectl --kubeconfig /your/path/to/greenhouse.kubeconfig get teams`}
            />
          </Stack>
        </Container>
      </PanelBody>
    </Panel>
  )
}

export default DownloadKubeConfig
