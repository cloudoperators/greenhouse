/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import {
  Container,
  DataGrid,
  DataGridCell,
  DataGridHeadCell,
  DataGridRow,
  Panel,
  PanelBody,
  Stack,
  DataGridToolbar,
  ButtonRow,
  Button,
  Icon,
} from "juno-ui-components"
import React, { useEffect } from "react"
import Markdown from "react-markdown"
import rehypeRaw from "rehype-raw"
import useClient from "../hooks/useClient"
import useNamespace from "../hooks/useNamespace"
import useStore from "../store"
import { PluginDefinition, Plugin } from "../types/types"
import OptionValueTable from "./OptionValueTable"
import PluginList from "./PluginList"

interface PluginDefinitionDetailProps {
  pluginDefinition: PluginDefinition
}

const PluginDefinitionDetail: React.FC<PluginDefinitionDetailProps> = (
  props: PluginDefinitionDetailProps
) => {
  const setShowPluginDefinitionDetails = useStore(
    (state) => state.setShowPluginDefinitionDetails
  )
  const onPanelClose = () => {
    setShowPluginDefinitionDetails(false)
  }

  const setShowPluginDefinitionEdit = useStore(
    (state) => state.setShowPluginEdit
  )
  const openEditPluginDefinition = () => {
    setShowPluginDefinitionDetails(false)
    setShowPluginDefinitionEdit(true)
  }

  const [deployedPlugins, setDeployedPlugins] = React.useState<Plugin[]>([])
  const greenhousePluginLabelKey = "greenhouse.sap/plugin"
  const labelSelector = `${greenhousePluginLabelKey}=${
    props.pluginDefinition.metadata!.name
  }`
  const { client: client } = useClient()
  const { namespace } = useNamespace()
  useEffect(() => {
    client
      .get(`/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/plugins`, {
        params: {
          labelSelector: labelSelector,
        },
      })
      .then((res) => {
        if (res.kind !== "PluginList") {
          console.log("ERROR: Failed to get Plugins for label " + labelSelector)
        } else {
          setDeployedPlugins(res.items as Plugin[])
        }
      })
  }, [client, namespace])

  const [markDown, setMarkDown] = React.useState<string>("")
  if (props.pluginDefinition.spec?.docMarkDownUrl) {
    useEffect(() => {
      fetch(props.pluginDefinition.spec!.docMarkDownUrl!)
        .then((response) => {
          if (!response.ok) {
            console.log(
              `failed fetching plugin ${
                props.pluginDefinition.metadata!.name
              } readme from ${props.pluginDefinition.spec!.docMarkDownUrl}.`
            )
          }
          response.text().then((text) => {
            setMarkDown(text)
          })
        })
        .catch((error) => {
          console.error(error)
        })
    }, [props.pluginDefinition.spec.docMarkDownUrl])
  }

  return (
    <Panel
      heading={
        <Stack gap="2">
          <span>
            {props.pluginDefinition.spec?.displayName ??
              (props.pluginDefinition.metadata?.name || "Not found")}
          </span>
        </Stack>
      }
      opened={!!props.pluginDefinition}
      onClose={onPanelClose}
      size="large"
    >
      <PanelBody>
        <Container px={false} py>
          <DataGridToolbar>
            <ButtonRow>
              <Button
                icon="addCircle"
                label="Configure Plugin"
                onClick={() => openEditPluginDefinition()}
              />
            </ButtonRow>
          </DataGridToolbar>
          <h2 className="text-xl font-bold mb-2 mt-8">General</h2>
          <DataGrid columns={2}>
            <DataGridRow>
              <DataGridHeadCell>Description</DataGridHeadCell>
              <DataGridCell>
                {props.pluginDefinition?.spec?.description}
              </DataGridCell>
            </DataGridRow>
            <DataGridRow>
              <DataGridHeadCell>Version</DataGridHeadCell>
              <DataGridCell>
                {props.pluginDefinition?.spec?.version}
              </DataGridCell>
            </DataGridRow>
            {props.pluginDefinition.spec?.helmChart && (
              <DataGridRow>
                <DataGridHeadCell>UI Application</DataGridHeadCell>
                <DataGridCell>
                  {props.pluginDefinition.spec?.helmChart?.name && (
                    <p>Name: {props.pluginDefinition.spec?.helmChart?.name}</p>
                  )}
                  {props.pluginDefinition.spec?.helmChart?.repository && (
                    <p>
                      Repository:{" "}
                      {props.pluginDefinition.spec?.helmChart?.repository}
                    </p>
                  )}
                  {props.pluginDefinition.spec?.helmChart?.version && (
                    <p>
                      Version: {props.pluginDefinition.spec?.helmChart?.version}
                    </p>
                  )}
                </DataGridCell>
              </DataGridRow>
            )}
            {props.pluginDefinition.spec?.uiApplication && (
              <DataGridRow>
                <DataGridHeadCell>UI Application</DataGridHeadCell>
                <DataGridCell>
                  {props.pluginDefinition.spec?.uiApplication?.name && (
                    <p>
                      Name: {props.pluginDefinition.spec?.uiApplication?.name}
                    </p>
                  )}
                  {props.pluginDefinition.spec?.uiApplication?.url && (
                    <p>
                      Url: {props.pluginDefinition.spec?.uiApplication?.url}
                    </p>
                  )}
                  {props.pluginDefinition.spec?.uiApplication?.version && (
                    <p>
                      Version:{" "}
                      {props.pluginDefinition.spec?.uiApplication?.version}
                    </p>
                  )}
                </DataGridCell>
              </DataGridRow>
            )}
            {deployedPlugins.length > 0 && (
              <PluginList plugins={deployedPlugins} />
            )}
          </DataGrid>
        </Container>

        {props.pluginDefinition?.spec?.options && (
          <OptionValueTable
            optionValues={props.pluginDefinition.spec.options}
          ></OptionValueTable>
        )}
        {markDown !== "" && (
          <Container px={false} py>
            <Stack
              direction="horizontal"
              alignment="center"
              distribution="center"
            >
              <h2 className="text-xl text-center font-bold mb-2 mt-8">
                Documentation{" "}
              </h2>
              <Icon
                target="_blank"
                href={props.pluginDefinition.spec!.docMarkDownUrl}
                icon="openInNew"
              />
            </Stack>

            <Markdown
              rehypePlugins={[rehypeRaw]}
              children={markDown}
            ></Markdown>
          </Container>
        )}
      </PanelBody>
    </Panel>
  )
}

export default PluginDefinitionDetail
