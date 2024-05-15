/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import {
  Button,
  ButtonRow,
  Container,
  DataGrid,
  DataGridCell,
  DataGridHeadCell,
  DataGridRow,
  DataGridToolbar,
  Panel,
  PanelBody,
  Stack,
} from "juno-ui-components"
import React, { useEffect } from "react"
import ReactMarkDown from "react-markdown"
import remarkGfm from "remark-gfm"
import { Plugin, PluginDefinition } from "../../../../types/types"
import useFetchMarkDown from "../hooks/useFetchMarkDown"
import usePluginApi from "../hooks/usePluginApi"
import useStore from "../store"
import OptionValueTable from "./OptionValueTable"
import PluginList from "./PluginList"

interface PluginDefinitionDetailProps {
  pluginDefinition: PluginDefinition
}

const PluginDefinitionDetail: React.FC<PluginDefinitionDetailProps> = (
  props: PluginDefinitionDetailProps
) => {
  const showPluginDefinitionDetails = useStore(
    (state) => state.showPluginDefinitionDetails
  )
  const setShowPluginDefinitionDetails = useStore(
    (state) => state.setShowPluginDefinitionDetails
  )
  const onPanelClose = () => {
    setShowPluginDefinitionDetails(false)
  }

  const setShowPluginDefinitionEdit = useStore(
    (state) => state.setShowPluginEdit
  )
  const { getPluginsByLabelSelector: getPluginsByLabelSelector } =
    usePluginApi()

  const { fetchMarkDown: fetchMarkDown } = useFetchMarkDown()

  const openEditPluginDefinition = () => {
    setShowPluginDefinitionDetails(false)
    setShowPluginDefinitionEdit(true)
  }

  const [deployedPlugins, setDeployedPlugins] = React.useState<Plugin[]>([])
  const greenhousePluginLabelKey = "greenhouse.sap/plugindefinition"

  useEffect(() => {
    const plugins = getPluginsByLabelSelector(
      greenhousePluginLabelKey,
      props.pluginDefinition.metadata!.name!
    )
    plugins.then((plugins) => {
      setDeployedPlugins(plugins)
    })
  }, [props.pluginDefinition.metadata?.name])

  const [markDown, setMarkDown] = React.useState<string>("")
  if (props.pluginDefinition.spec?.docMarkDownUrl) {
    useEffect(() => {
      let getMD = fetchMarkDown(props.pluginDefinition.spec!.docMarkDownUrl!)
      getMD.then((markDown) => {
        setMarkDown(markDown)
      })
    }, [props.pluginDefinition.spec.docMarkDownUrl])
  }

  return (
    <Panel
      opened={!!showPluginDefinitionDetails}
      heading={
        <Stack gap="2">
          <span>
            {props.pluginDefinition.spec?.displayName ??
              (props.pluginDefinition.metadata?.name || "Not found")}
          </span>
        </Stack>
      }
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
                <DataGridHeadCell>Helm Chart</DataGridHeadCell>
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
            <h2 className="text-xl font-bold mb-2 mt-8">Documentation </h2>
            <article className="markdown-body">
              <ReactMarkDown
                rehypePlugins={[remarkGfm]}
                // children={markDown}
              >
                {markDown}
              </ReactMarkDown>
            </article>
          </Container>
        )}
      </PanelBody>
    </Panel>
  )
}

export default PluginDefinitionDetail
