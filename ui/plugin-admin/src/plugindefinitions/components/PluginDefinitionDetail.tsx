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
import { useGlobalsActions } from "../../plugins/components/StoreProvider"
import OptionValueTable from "./OptionValueTable"
import PluginList from "./PluginList"
import useStore from "../store"
import { PANEL } from "../../constants"

interface PluginDefinitionDetailProps {
  pluginDefinition: PluginDefinition
}

const PluginDefinitionDetail: React.FC<PluginDefinitionDetailProps> = (
  props: PluginDefinitionDetailProps
) => {
  const { setPanel } = useGlobalsActions()
  const { getPluginsByLabelSelector: getPluginsByLabelSelector } =
    usePluginApi()

  const { fetchMarkDown: fetchMarkDown } = useFetchMarkDown()
  const setIsEditMode = useStore((state) => state.setIsPluginEditMode)

  const openEditPluginDefinition = () => {
    setIsEditMode(false)
    setPanel(PANEL.EDIT_PLUGIN)
  }

  const [deployedPlugins, setDeployedPlugins] = React.useState<Plugin[]>([])
  const greenhousePluginLabelKey = "greenhouse.sap/plugin"

  useEffect(() => {
    const plugins = getPluginsByLabelSelector(
      greenhousePluginLabelKey,
      props.pluginDefinition.metadata!.name!
    )
    plugins.then((plugins) => {
      setDeployedPlugins(plugins)
    })
  }, [setDeployedPlugins])

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
    <>
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
            <DataGridCell>{props.pluginDefinition?.spec?.version}</DataGridCell>
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
                  <p>Url: {props.pluginDefinition.spec?.uiApplication?.url}</p>
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
    </>
  )
}

export default PluginDefinitionDetail
