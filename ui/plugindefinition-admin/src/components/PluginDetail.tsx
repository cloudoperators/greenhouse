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
} from "juno-ui-components";
import React, { useEffect } from "react";
import Markdown from "react-markdown";
import rehypeRaw from "rehype-raw";
import useClient from "../hooks/useClient";
import useNamespace from "../hooks/useNamespace";
import useStore from "../store";
import { Plugin, PluginConfig } from "../types/types";
import OptionValueTable from "./OptionValueTable";
import PluginConfigList from "./PluginConfigList";

interface PluginDetailProps {
  plugin: Plugin;
}

const PluginDetail: React.FC<PluginDetailProps> = (
  props: PluginDetailProps
) => {
  const setShowPluginDetails = useStore((state) => state.setShowPluginDetails);
  const onPanelClose = () => {
    setShowPluginDetails(false);
  };

  const setShowPluginEdit = useStore((state) => state.setShowPluginEdit);
  const openEditPlugin = () => {
    setShowPluginDetails(false);
    setShowPluginEdit(true);
  };

  const [deployedPlugins, setDeployedPlugins] = React.useState<PluginConfig[]>(
    []
  );
  const greenhousePluginLabelKey = "greenhouse.sap/plugin";
  const labelSelector = `${greenhousePluginLabelKey}=${
    props.plugin.metadata!.name
  }`;
  const { client: client } = useClient();
  const { namespace } = useNamespace();
  useEffect(() => {
    client
      .get(
        `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/pluginconfigs`,
        {
          params: {
            labelSelector: labelSelector,
          },
        }
      )
      .then((res) => {
        if (res.kind !== "PluginConfigList") {
          console.log(
            "ERROR: Failed to get Plugins for label " + labelSelector
          );
        } else {
          setDeployedPlugins(res.items as PluginConfig[]);
        }
      });
  }, [client, namespace]);

  const [markDown, setMarkDown] = React.useState<string>("");
  if (props.plugin.spec?.docMarkDownUrl) {
    useEffect(() => {
      fetch(props.plugin.spec!.docMarkDownUrl!)
        .then((response) => {
          if (!response.ok) {
            console.log(
              `failed fetching plugin ${
                props.plugin.metadata!.name
              } readme from ${props.plugin.spec!.docMarkDownUrl}.`
            );
          }
          response.text().then((text) => {
            setMarkDown(text);
          });
        })
        .catch((error) => {
          console.error(error);
        });
    }, [props.plugin.spec.docMarkDownUrl]);
  }

  return (
    <Panel
      heading={
        <Stack gap="2">
          <span>
            {props.plugin.spec?.displayName ??
              (props.plugin.metadata?.name || "Not found")}
          </span>
        </Stack>
      }
      opened={!!props.plugin}
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
                onClick={() => openEditPlugin()}
              />
            </ButtonRow>
          </DataGridToolbar>
          <h2 className="text-xl font-bold mb-2 mt-8">General</h2>
          <DataGrid columns={2}>
            <DataGridRow>
              <DataGridHeadCell>Description</DataGridHeadCell>
              <DataGridCell>{props.plugin?.spec?.description}</DataGridCell>
            </DataGridRow>
            <DataGridRow>
              <DataGridHeadCell>Version</DataGridHeadCell>
              <DataGridCell>{props.plugin?.spec?.version}</DataGridCell>
            </DataGridRow>
            {props.plugin.spec?.helmChart && (
              <DataGridRow>
                <DataGridHeadCell>UI Application</DataGridHeadCell>
                <DataGridCell>
                  {props.plugin.spec?.helmChart?.name && (
                    <p>Name: {props.plugin.spec?.helmChart?.name}</p>
                  )}
                  {props.plugin.spec?.helmChart?.repository && (
                    <p>
                      Repository: {props.plugin.spec?.helmChart?.repository}
                    </p>
                  )}
                  {props.plugin.spec?.helmChart?.version && (
                    <p>Version: {props.plugin.spec?.helmChart?.version}</p>
                  )}
                </DataGridCell>
              </DataGridRow>
            )}
            {props.plugin.spec?.uiApplication && (
              <DataGridRow>
                <DataGridHeadCell>UI Application</DataGridHeadCell>
                <DataGridCell>
                  {props.plugin.spec?.uiApplication?.name && (
                    <p>Name: {props.plugin.spec?.uiApplication?.name}</p>
                  )}
                  {props.plugin.spec?.uiApplication?.url && (
                    <p>Url: {props.plugin.spec?.uiApplication?.url}</p>
                  )}
                  {props.plugin.spec?.uiApplication?.version && (
                    <p>Version: {props.plugin.spec?.uiApplication?.version}</p>
                  )}
                </DataGridCell>
              </DataGridRow>
            )}
            {deployedPlugins.length > 0 && (
              <PluginConfigList pluginConfigs={deployedPlugins} />
            )}
          </DataGrid>
        </Container>

        {props.plugin?.spec?.options && (
          <OptionValueTable
            optionValues={props.plugin.spec.options}
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
                href={props.plugin.spec!.docMarkDownUrl}
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
  );
};

export default PluginDetail;
