/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import {
  Button,
  ButtonRow,
  Checkbox,
  Form,
  FormRow,
  FormSection,
  Panel,
  PanelBody,
  Stack,
  TextInput,
  Textarea,
} from "juno-ui-components";
import React from "react";
import { Plugin, PluginConfig } from "../../../../shared/types/types";
import useClient from "../../hooks/useClient";
import useNamespace from "../../hooks/useNamespace";
import useStore from "../../store";

interface PluginEditProps {
  plugin: Plugin;
}

const PluginEdit: React.FC<PluginEditProps> = (props: PluginEditProps) => {
  const { client: client } = useClient();
  const { namespace } = useNamespace();
  const setShowPluginEdit = useStore((state) => state.setShowPluginEdit);
  const onPanelClose = () => {
    setShowPluginEdit(false);
  };
  // instantiate new empty PluginConfig from Plugin
  let initPluginConfig: PluginConfig = {
    metadata: {
      name: props.plugin.metadata!.name!,
      namespace: "",
      labels: {},
    },
    kind: "PluginConfig",
    apiVersion: "greenhouse.sap/v1alpha1",
    spec: {
      plugin: props.plugin.metadata!.name!,
      displayName:
        props.plugin.spec?.displayName ?? props.plugin.metadata?.name,
      clusterName: "",
      disabled: false,
      optionValues: [],
    },
  };
  props.plugin.spec?.options?.forEach((option) => {
    if (
      option.default &&
      !initPluginConfig.spec?.optionValues!.some((o) => o.name == option.name)
    ) {
      initPluginConfig.spec?.optionValues!.push({
        name: option.name,
        value: option.default,
      });
    }
  });

  // TODO: We are overwriting empty fields with defaults!!
  // Anyhow need to add logic for editing PluginConfigs instead of creating
  const [pluginConfig, setPluginConfig] =
    React.useState<PluginConfig>(initPluginConfig);

  const [errorMessage, setErrorMessage] = React.useState<string>("");

  const onSubmit = () => {
    console.log(pluginConfig);
    client
      .post(
        `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/pluginconfigs/${
          pluginConfig.metadata!.name
        }`,
        { ...pluginConfig }
      )
      .then((res) => {
        console.log(res);
        setErrorMessage("Success!");
      })
      .catch((err) => {
        console.log(err);
        setErrorMessage(err.message);
      });
  };
  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    let value: string | boolean | number;
    console.log(e.target.type);
    if (e.target.type == "checkbox") {
      value = e.target.checked ? true : false;
    } else if (e.target.type == "number") {
      value = parseInt(e.target.value);
    } else if (e.target.type == "textarea") {
      value = JSON.parse(e.target.value);
    } else {
      value = e.target.value;
    }
    console.log(e.target.id + " " + value);

    if (e.target.id.startsWith("metadata.")) {
      setPluginConfig({
        ...pluginConfig,
        metadata: {
          ...pluginConfig.metadata!,
          [e.target.id.split(".")[1]]: value,
        },
      });
    } else if (e.target.id.startsWith("spec.")) {
      setPluginConfig({
        ...pluginConfig,
        spec: {
          ...pluginConfig.spec!,
          [e.target.id.split(".")[1]]: value,
        },
      });
    } else if (e.target.id.startsWith("optionValues.")) {
      // delete from pluginConfig.spec.optionValues by matching name property if value is empty
      // does not work yet!!
      if (value == "") {
        setPluginConfig({
          ...pluginConfig,
          spec: {
            ...pluginConfig.spec!,
            optionValues: pluginConfig.spec!.optionValues!.filter(
              (option) => option.name != e.target.id.split(".")[1]
            ),
          },
        });
        console.log(pluginConfig.spec!.optionValues!);
      }
      //   replace in pluginConfig.spec.optionValues by matching name property or push if not found
      let wasFound = false;

      setPluginConfig({
        ...pluginConfig,
        spec: {
          ...pluginConfig.spec!,
          optionValues: pluginConfig.spec!.optionValues!.map((option) => {
            if (option.name == e.target.id.split(".")[1]) {
              wasFound = true;
              return { name: option.name, value: value };
            } else {
              return option;
            }
          }),
        },
      });
      if (!wasFound) {
        setPluginConfig({
          ...pluginConfig,
          spec: {
            ...pluginConfig.spec!,
            optionValues: [
              ...pluginConfig.spec!.optionValues!,
              { name: e.target.id.split(".")[1], value: value },
            ],
          },
        });
      }
    }
  };
  return (
    <Panel
      heading={
        <Stack gap="2">
          <span>Configure Plugin</span>
        </Stack>
      }
      opened={!!props.plugin}
      onClose={onPanelClose}
      size="large"
    >
      <PanelBody>
        <Form
          title={props.plugin.spec?.displayName ?? props.plugin.metadata?.name}
        >
          <FormSection title="General">
            <FormRow>
              <TextInput
                id="spec.displayName"
                label="Display Name"
                placeholder="The Display Name for this Plugin Instance"
                value={pluginConfig.spec!.displayName}
                onBlur={handleChange}
              />
            </FormRow>
            <FormRow>
              <TextInput
                id="metadata.name"
                label="Name"
                placeholder="Name of this Plugin Instance"
                value={pluginConfig.metadata!.name}
                onBlur={handleChange}
              />
            </FormRow>
            <FormRow>
              {/* TODO: Add Cluster Selector from available Clusters in Ready state */}
              <TextInput
                id="spec.clusterName"
                label="Cluster"
                placeholder="this is going to be a select of ready clusters"
                value={pluginConfig.spec!.clusterName}
                onBlur={handleChange}
              />
            </FormRow>
          </FormSection>

          {props.plugin.spec?.options?.length && (
            <FormSection title="Options">
              {props.plugin.spec?.options?.map((option, index) => {
                let value = pluginConfig.spec?.optionValues?.find(
                  (o) => o.name == option.name
                )?.value;
                return (
                  <FormRow key={index}>
                    <p>{option.description}</p>
                    {option.type == "string" && (
                      <TextInput
                        id={"optionValues." + option.name}
                        label={option.name}
                        required={option.required}
                        helptext={option.type}
                        placeholder={option.description}
                        value={value}
                        onBlur={handleChange}
                      />
                    )}
                    {option.type == "secret" && (
                      <TextInput
                        id={"optionValues." + option.name}
                        label={option.name}
                        required={option.required}
                        helptext={option.type}
                        placeholder={option.description}
                        value={value}
                        type="password"
                        onBlur={handleChange}
                      />
                    )}
                    {option.type == "bool" && (
                      <Checkbox
                        id={"optionValues." + option.name}
                        label={option.name}
                        required={option.required}
                        helptext={option.type}
                        checked={option.default ?? false}
                        onBlur={handleChange}
                      />
                    )}
                    {option.type == "int" && (
                      <TextInput
                        type="number"
                        id={"optionValues." + option.name}
                        label={option.name}
                        required={option.required}
                        helptext={option.type}
                        placeholder={option.description}
                        value={value}
                        onBlur={handleChange}
                      />
                    )}
                    {(option.type == "list" || option.type == "map") && (
                      <Textarea
                        id={"optionValues." + option.name}
                        label={option.name}
                        required={option.required}
                        helptext={option.type}
                        value={JSON.stringify(value)}
                        onBlur={handleChange}
                      ></Textarea>
                    )}
                  </FormRow>
                );
              })}
            </FormSection>
          )}

          <ButtonRow>
            {errorMessage != "" && <p>{errorMessage}</p>}
            <Button onClick={onSubmit} variant="primary">
              Submit
            </Button>
          </ButtonRow>
        </Form>
      </PanelBody>
    </Panel>
  );
};

export default PluginEdit;
