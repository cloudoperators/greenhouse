/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import {
  Button,
  ButtonRow,
  Form,
  FormRow,
  FormSection,
  Panel,
  PanelBody,
  Stack,
  TextInput,
} from "juno-ui-components"
import React from "react"
import { Plugin, PluginDefinition } from "../../../types/types"
import useClient from "../plugindefinitions/hooks/useClient"
import useNamespace from "../plugindefinitions/hooks/useNamespace"
import useStore from "../plugindefinitions/store"
import ClusterSelect from "./ClusterSelect"
import { OptionInput } from "./OptionInput"
import handleFormChange from "./lib/utils/handleFormChange"
import initPlugin from "./lib/utils/initPlugin"
import postPlugin from "./lib/utils/postPlugin"

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

  const setShowPluginEdit = useStore((state) => state.setShowPluginEdit)
  const onPanelClose = () => {
    setShowPluginEdit(false)
  }

  const { client: client } = useClient()
  const { namespace } = useNamespace()
  const [submitMessage, setSubmitResultMessage] = React.useState<string>("")
  const [plugin, setPlugin] = React.useState<Plugin>(editedPlugin!)

  const onSubmit = async () => {
    let message = await postPlugin(plugin, namespace, client)
    setSubmitResultMessage(message)
  }

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    handleFormChange(e, plugin, setPlugin)
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
                placeholder="The Cluster this Plugin is to be deployed to."
                label="Cluster"
                onChange={handleChange}
              />
            </FormRow>
          </FormSection>

          {props.pluginDefinition.spec?.options?.length && (
            <FormSection title="Options">
              {props.pluginDefinition.spec?.options?.map((option, index) => {
                let optionValue = plugin.spec?.optionValues?.find(
                  (o) => o.name == option.name
                )
                return (
                  <FormRow key={index}>
                    <p>{option.description}</p>
                    <OptionInput
                      pluginDefinitionOption={option}
                      pluginOptionValue={optionValue}
                      isEditMode={isEditMode}
                      onChange={handleChange}
                    />
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
