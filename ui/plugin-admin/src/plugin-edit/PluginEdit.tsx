/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import {
  Button,
  Form,
  FormRow,
  FormSection,
  Message,
  Panel,
  PanelBody,
  Stack,
  TextInput,
} from "juno-ui-components"
import React from "react"
import { Plugin, PluginDefinition, SecretDataEntry } from "../../../types/types"
import usePluginApi from "../plugindefinitions/hooks/usePluginApi"
import useSecretApi from "../plugindefinitions/hooks/useSecretApi"
import useStore from "../plugindefinitions/store"
import ClusterSelect from "./ClusterSelect"
import { OptionInput } from "./OptionInput"
import handleFormChange from "./lib/utils/handleFormChange"
import initPlugin from "./lib/utils/initPlugin"

interface PluginEditProps {
  pluginDefinition: PluginDefinition
}

type SubmitMessage = {
  message: string
  ok: boolean
}

// TODO: Hickup observed with resource states: Try getting plugin from server after failed post/put
// TODO: Validate JSON on list/map inputs
const PluginEdit: React.FC<PluginEditProps> = (props: PluginEditProps) => {
  const setShowPluginEdit = useStore((state) => state.setShowPluginEdit)

  const pluginToEdit = useStore((state) => state.pluginToEdit)
  const setPluginToEdit = useStore((state) => state.setPluginToEdit)

  // TODO: Need to get the secret from the server if it exists
  // to be able to fill the value in the UI
  const secretToEdit = useStore((state) => state.secretToEdit)
  const setSecretToEdit = useStore((state) => state.setSecretToEdit)

  const isEditMode = useStore((state) => state.isEditMode)
  const setIsEditMode = useStore((state) => state.setIsEditMode)

  const isSecretEditMode = useStore((state) => state.isSecretEditMode)
  const setIsSecretEditMode = useStore((state) => state.setIsSecretEditMode)

  const { createPlugin, updatePlugin, deletePlugin } = usePluginApi()
  const { createSecret, updateSecret } = useSecretApi()

  //init plugin only if it is not already initialized
  React.useEffect(() => {
    if (!pluginToEdit) {
      setPluginToEdit(initPlugin(props.pluginDefinition))
    }
  }, [props.pluginDefinition])

  const onPanelClose = () => {
    setPluginToEdit(undefined)
    setSecretToEdit(undefined)
    setShowPluginEdit(false)
    setIsEditMode(false)
    setIsSecretEditMode(false)
  }

  const [submitMessage, setSubmitResultMessage] = React.useState<SubmitMessage>(
    { message: "", ok: false }
  )
  const onSubmit = async () => {
    // if we have secret values, then first create/update the secrets, then create/update the plugin
    if (secretToEdit) {
      let secretCreatePromise = isSecretEditMode
        ? createSecret(secretToEdit!)
        : updateSecret(secretToEdit!)

      await secretCreatePromise.then(async (res) => {
        if (!res.ok) {
          setSubmitResultMessage({
            message: "Failed to create/update plugin + " + res.message,
            ok: res.ok,
          })
          return
        }
        setIsSecretEditMode(true)
        let pluginCreatePromise = isEditMode
          ? updatePlugin(pluginToEdit!)
          : createPlugin(pluginToEdit!)

        await pluginCreatePromise.then(async (res) => {
          setSubmitResultMessage({ message: res.message, ok: res.ok })
          return
        })
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
    let res = await deletePlugin(pluginToEdit!)
    setSubmitResultMessage({ message: res.message, ok: res.ok })
  }

  const onMessageDismiss = (ok: boolean) => {
    if (ok) {
      setShowPluginEdit(false)
      setPluginToEdit(undefined)
      setIsEditMode(false)
      setIsSecretEditMode(false)
      // TODO: Implement a way to open the details for the plugin
      console.log("I want to open the details for my plugin now :)")
    }
  }

  const handleFormElementChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    try {
      let changedPlugin: Plugin
      let changedSecretEntry: SecretDataEntry | undefined
      ;[changedPlugin, changedSecretEntry] = handleFormChange(e, pluginToEdit!)
      setPluginToEdit(changedPlugin)

      if (changedSecretEntry != undefined) {
        // if secretToEdit is not set, then create a new secret object
        if (!secretToEdit) {
          setSecretToEdit({
            apiVersion: "v1",
            kind: "Secret",
            metadata: {
              name: pluginToEdit!.metadata!.name,
            },
            stringData: changedSecretEntry,
          })
        } else {
          // update secretToEdit.data with changedSecretEntry
          setSecretToEdit({
            ...secretToEdit,
            stringData: {
              ...secretToEdit.stringData,
              ...changedSecretEntry,
            },
          })
        }
      }
    } catch (e) {
      console.error(e)
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
                  value={pluginToEdit!.metadata!.name}
                  onBlur={handleFormElementChange}
                />
              </FormRow>
              <FormRow>
                <ClusterSelect
                  id="spec.clusterName"
                  placeholder="The Cluster this Plugin is to be deployed to."
                  label="Cluster"
                  defaultValue={pluginToEdit!.spec!.clusterName}
                  onChange={handleFormElementChange}
                />
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
                      <p>{option.description}</p>
                      <OptionInput
                        pluginDefinitionOption={option}
                        pluginOptionValue={optionValue}
                        isEditMode={isEditMode}
                        onChange={handleFormElementChange}
                      />
                    </FormRow>
                  )
                })}
              </FormSection>
            )}

            <Stack distribution="between">
              <Button onClick={onDelete} variant="primary-danger">
                Delete Plugin
              </Button>
              {submitMessage.message != "" && (
                <Message
                  autoDismissTimeout={3000}
                  autoDismiss={submitMessage.ok}
                  onDismiss={() => onMessageDismiss(submitMessage.ok)}
                  variant={submitMessage.ok ? "success" : "error"}
                  text={submitMessage.message}
                />
              )}
              <Button onClick={onSubmit} variant="primary">
                {isEditMode ? "Update Plugin" : "Create Plugin"}
              </Button>
            </Stack>
          </Form>
        </PanelBody>
      )}
    </Panel>
  )
}

export default PluginEdit
