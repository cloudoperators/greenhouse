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
} from "juno-ui-components"
import React from "react"
import { Plugin, PluginDefinition } from "../../../types/types"
import useClient from "../plugindefinitions/hooks/useClient"
import useNamespace from "../plugindefinitions/hooks/useNamespace"
import useStore from "../plugindefinitions/store"
import initPlugin from "./lib/utils/initPlugin"
import postPlugin from "./lib/utils/postPlugin"
import ClusterSelect from "./ClusterSelect"

interface PluginEditProps {
  pluginDefinition: PluginDefinition
  plugin?: Plugin
}

const PluginEdit: React.FC<PluginEditProps> = (props: PluginEditProps) => {
  // we are editing an existing plugin, if prop.plugin is defined
  const isEditMode = !!props.plugin
  let editedPlugin = props.plugin
  if (!isEditMode) {
    editedPlugin = initPlugin(props.pluginDefinition)
  }

  const { client: client } = useClient()
  const { namespace } = useNamespace()
  const setShowPluginEdit = useStore((state) => state.setShowPluginEdit)
  const onPanelClose = () => {
    setShowPluginEdit(false)
  }

  const [plugin, setPlugin] = React.useState<Plugin>(editedPlugin!)

  const [submitMessage, setSubmitResultMessage] = React.useState<string>("")

  const onSubmit = async () => {
    let message = await postPlugin(plugin, namespace, client)
    setSubmitResultMessage(message)
  }

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    let value: string | boolean | number
    if (e.target?.type != undefined) {
      console.error("Unexpected form change event: " + e)
    }
    switch (e.target.type) {
      case "checkbox":
        value = e.target.checked ? true : false
        break
      case "number":
        value = parseInt(e.target.value)
        break
      case "textarea":
        value = JSON.parse(e.target.value)
        break
      default:
        value = e.target.value
        break
    }

    if (e.target.id.startsWith("metadata.")) {
      setPlugin({
        ...plugin,
        metadata: {
          ...plugin.metadata!,
          [e.target.id.split(".")[1]]: value,
        },
      })
    } else if (e.target.id.startsWith("spec.")) {
      setPlugin({
        ...plugin,
        spec: {
          ...plugin.spec!,
          [e.target.id.split(".")[1]]: value,
        },
      })
    } else if (e.target.id.startsWith("optionValues.")) {
      // delete from pluginConfig.spec.optionValues by matching name property if value is empty
      // does not work yet!!
      if (value == "") {
        setPlugin({
          ...plugin,
          spec: {
            ...plugin.spec!,
            optionValues: plugin.spec!.optionValues!.filter(
              (option) => option.name != e.target.id.split(".")[1]
            ),
          },
        })
        console.log(plugin.spec!.optionValues!)
      }
      //   replace in pluginConfig.spec.optionValues by matching name property or push if not found
      let wasFound = false

      setPlugin({
        ...plugin,
        spec: {
          ...plugin.spec!,
          optionValues: plugin.spec!.optionValues!.map((option) => {
            if (option.name == e.target.id.split(".")[1]) {
              wasFound = true
              return { name: option.name, value: value }
            } else {
              return option
            }
          }),
        },
      })
      if (!wasFound) {
        setPlugin({
          ...plugin,
          spec: {
            ...plugin.spec!,
            optionValues: [
              ...plugin.spec!.optionValues!,
              { name: e.target.id.split(".")[1], value: value },
            ],
          },
        })
      }
    }
  }
  return (
    <Panel
      heading={
        <Stack gap="2">
          <span>Configure Plugin</span>
        </Stack>
      }
      opened={!!props.pluginDefinition}
      onClose={onPanelClose}
      size="large"
    >
      <PanelBody>
        <Form
          title={
            props.pluginDefinition.spec?.displayName ??
            props.pluginDefinition.metadata?.name
          }
        >
          <FormSection title="General">
            <FormRow>
              <TextInput
                id="spec.displayName"
                label="Display Name"
                placeholder="The Display Name for this Plugin Instance"
                value={plugin.spec!.displayName}
                onBlur={handleChange}
              />
            </FormRow>
            <FormRow>
              <TextInput
                id="metadata.name"
                label="Name"
                placeholder="Name of this Plugin Instance"
                value={plugin.metadata!.name}
                onBlur={handleChange}
              />
            </FormRow>
            <FormRow>
              <ClusterSelect
                id="spec.clusterName"
                label="Cluster"
                onChange={handleChange}
              />
            </FormRow>
          </FormSection>

          {props.pluginDefinition.spec?.options?.length && (
            <FormSection title="Options">
              {props.pluginDefinition.spec?.options?.map((option, index) => {
                let value = plugin.spec?.optionValues?.find(
                  (o) => o.name == option.name
                )?.value
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
                )
              })}
            </FormSection>
          )}

          <ButtonRow>
            {submitMessage != "" && <p>{submitMessage}</p>}
            <Button onClick={onSubmit} variant="primary">
              Submit
            </Button>
          </ButtonRow>
        </Form>
      </PanelBody>
    </Panel>
  )
}

export default PluginEdit
