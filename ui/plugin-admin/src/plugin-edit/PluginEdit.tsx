/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import {
  Button,
  Form,
  FormRow,
  FormSection,
  Panel,
  PanelBody,
  Stack,
  Switch,
  TextInput,
} from "juno-ui-components"
import React from "react"
import {
  LabelSelector,
  PluginDefinition,
  PluginPreset,
} from "../../../types/types"
import usePluginApi from "../plugindefinitions/hooks/usePluginApi"
import usePluginPresetApi from "../plugindefinitions/hooks/usePluginPresetApi"
import useStore from "../plugindefinitions/store"
import KeyValueInput from "../secrets/KeyValueInput"
import ClusterSelect from "./ClusterSelect"
import { OptionInput } from "./OptionInput"
import SubmitResultMessage, { SubmitMessage } from "./SubmitResultMessage"
import handleFormChange from "./handleFormChange"
import initPlugin from "./initPlugin"
import initPluginPreset from "./initPluginPreset"
import useNamespace from "../plugindefinitions/hooks/useNamespace"

interface PluginEditProps {
  pluginDefinition: PluginDefinition
}

// TODO: If editing existing plugin, we currently cant create preset from it

// TODO: Properly distinguish between **editing** a plugin and a plugin preset
// TODO: Validate JSON on list/map inputs
const PluginEdit: React.FC<PluginEditProps> = (props: PluginEditProps) => {
  const { namespace } = useNamespace()
  const setShowPluginEdit = useStore((state) => state.setShowPluginEdit)

  const pluginToEdit = useStore((state) => state.pluginToEdit)
  const setPluginToEdit = useStore((state) => state.setPluginToEdit)

  const isEditMode = useStore((state) => state.isPluginEditMode)
  const setIsEditMode = useStore((state) => state.setIsPluginEditMode)

  const { createPlugin, updatePlugin, deletePlugin } = usePluginApi()
  const { createPluginPreset, updatePluginPreset, deletePluginPreset } =
    usePluginPresetApi()

  const [isPluginPreset, setIsPluginPreset] = React.useState(false)
  const changeIsPluginPreset = () => {
    setIsPluginPreset(!isPluginPreset)
  }

  const [pluginPresetName, setPluginPresetName] = React.useState("")
  // if plugin metadata labels contain a label with key greenhouse.sap/pluginpreset
  // we assume this plugin is a plugin preset
  React.useEffect(() => {
    if (
      pluginToEdit &&
      pluginToEdit.metadata!.labels &&
      pluginToEdit.metadata!.labels["greenhouse.sap/pluginpreset"]
    ) {
      setIsPluginPreset(true)
      setSubmitResultMessage({
        message: "This Plugin is part of a Preset. You are editing the Preset!",
        ok: false,
        variant: "warning",
      })
      setPluginPresetName(
        pluginToEdit.metadata!.labels["greenhouse.sap/pluginpreset"]
      )
    } else {
      setPluginPresetName(pluginToEdit?.metadata?.name ?? "")
    }
  }, [pluginToEdit])

  const kindName = isPluginPreset ? "Plugin Preset" : "Plugin"

  const emptyLabelSelector: LabelSelector = {
    "": "",
  }
  const [labelSelector, setLabelSelector] = React.useState(emptyLabelSelector)

  React.useEffect(() => {
    if (!pluginToEdit) {
      setPluginToEdit(initPlugin(props.pluginDefinition))
    }
  }, [props.pluginDefinition])

  const onPanelClose = () => {
    setPluginToEdit(undefined)
    setShowPluginEdit(false)
    setIsEditMode(false)
  }

  const [submitMessage, setSubmitResultMessage] = React.useState<SubmitMessage>(
    { message: "", ok: false }
  )
  const onSubmit = async () => {
    if (isPluginPreset) {
      let pluginPreset: PluginPreset = initPluginPreset(
        pluginPresetName,
        pluginToEdit!
      )
      pluginPreset.spec!.clusterSelector.matchLabels = labelSelector

      let pluginPresetCreatePromise = isEditMode
        ? updatePluginPreset(pluginPreset)
        : createPluginPreset(pluginPreset)

      await pluginPresetCreatePromise.then(async (res) => {
        setSubmitResultMessage({ message: res.message, ok: res.ok })
      })
    } else {
      let pluginCreatePromise = isEditMode
        ? updatePlugin(pluginToEdit!)
        : createPlugin(pluginToEdit!)

      await pluginCreatePromise.then(async (res) => {
        setSubmitResultMessage({ message: res.message, ok: res.ok })
      })
    }
  }

  // TODO: Implement second confirmation dialog for delete
  const onDelete = async () => {
    if (isPluginPreset) {
      let res = await deletePluginPreset(
        initPluginPreset(pluginPresetName, pluginToEdit!)
      )
      setSubmitResultMessage({ message: res.message, ok: res.ok })
    } else {
      let res = await deletePlugin(pluginToEdit!)
      setSubmitResultMessage({ message: res.message, ok: res.ok })
    }
  }

  const onMessageDismiss = (ok: boolean) => {
    if (ok) {
      setShowPluginEdit(false)
      setPluginToEdit(undefined)
      setIsEditMode(false)
      // TODO: Implement a way to open the details for the plugin --> just show a button!
      console.log("I want to open the details for my plugin now :)")
    }
  }

  const handleFormElementChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    try {
      setPluginToEdit(handleFormChange(e, pluginToEdit!))
    } catch (e) {
      console.error(e)
    }
  }

  return (
    <Panel
      heading={
        <Stack gap="2">
          <span>Configure {kindName}</span>
        </Stack>
      }
      opened={!!props.pluginDefinition}
      onClose={onPanelClose}
      size="large"
    >
      {pluginToEdit && (
        <PanelBody>
          <Form
            title={
              props.pluginDefinition.spec?.displayName ??
              props.pluginDefinition.metadata?.name
            }
          >
            <FormSection title="General">
              <FormRow>
                <Switch
                  id="switch-plugin-preset"
                  label="Make Plugin Preset"
                  on={isPluginPreset}
                  onChange={changeIsPluginPreset}
                  onClick={changeIsPluginPreset}
                />
              </FormRow>
              <FormRow>
                <TextInput
                  id="spec.displayName"
                  label="Display Name"
                  placeholder="The Display Name for this Plugin Instance"
                  value={pluginToEdit!.spec!.displayName}
                  onBlur={handleFormElementChange}
                />
              </FormRow>
              <FormRow>
                <TextInput
                  id="metadata.name"
                  label="Name"
                  placeholder="Name of this Plugin Instance"
                  {...(isEditMode && { disabled: true })}
                  value={
                    isPluginPreset
                      ? pluginPresetName
                      : pluginToEdit!.metadata!.name
                  }
                  onBlur={handleFormElementChange}
                />
              </FormRow>
              <FormRow>
                {isPluginPreset && (
                  <KeyValueInput
                    data={labelSelector}
                    setData={setLabelSelector}
                    title="Cluster Label Selector"
                    dataName="Label"
                  ></KeyValueInput>
                )}
                {!isPluginPreset && (
                  <ClusterSelect
                    id="spec.clusterName"
                    placeholder="The Cluster this Plugin is to be deployed to."
                    label="Cluster"
                    defaultValue={pluginToEdit!.spec!.clusterName}
                    onChange={handleFormElementChange}
                  />
                )}
              </FormRow>
              <FormRow>
                <TextInput
                  id="spec.releaseNamespace"
                  label="Release Namespace"
                  placeholder={`The namespace in the remote cluster to which the backend is deployed to. Defaults to ${namespace}.`}
                  value={pluginToEdit!.spec!.releaseNamespace}
                  onBlur={handleFormElementChange}
                ></TextInput>
              </FormRow>
            </FormSection>

            {props.pluginDefinition.spec?.options?.length && (
              <FormSection title="Options">
                {props.pluginDefinition.spec?.options?.map((option, index) => {
                  let optionValue = pluginToEdit!.spec?.optionValues?.find(
                    (o) => o.name == option.name
                  )
                  return (
                    <FormRow key={index}>
                      <p style={{ color: "text-theme-light" }}>
                        {option.description}
                      </p>
                      <OptionInput
                        pluginDefinitionOption={option}
                        pluginOptionValue={optionValue}
                        onChange={handleFormElementChange}
                      />
                    </FormRow>
                  )
                })}
              </FormSection>
            )}

            <Stack distribution="between">
              <Button onClick={onDelete} variant="primary-danger">
                Delete {kindName}
              </Button>
              {submitMessage.message != "" && (
                <SubmitResultMessage
                  submitMessage={submitMessage}
                  onMessageDismiss={() => onMessageDismiss(submitMessage.ok)}
                />
              )}
              <Button onClick={onSubmit} variant="primary">
                {isEditMode ? `Update ${kindName}` : `Create ${kindName}`}
              </Button>
            </Stack>
          </Form>
        </PanelBody>
      )}
    </Panel>
  )
}

export default PluginEdit
