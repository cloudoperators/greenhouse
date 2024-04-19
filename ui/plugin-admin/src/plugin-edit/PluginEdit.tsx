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
  Message,
  Panel,
  PanelBody,
  Stack,
  TextInput,
} from "juno-ui-components"
import React from "react"
import { PluginDefinition } from "../../../types/types"
import usePluginApi from "../plugindefinitions/hooks/usePluginApi"
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

// TODO: Hickup observed with resource states -- check for latest version on plugin update
// TODO: Validate JSON on list/map inputs
const PluginEdit: React.FC<PluginEditProps> = (props: PluginEditProps) => {
  const setShowPluginEdit = useStore((state) => state.setShowPluginEdit)
  const setPluginToEdit = useStore((state) => state.setPluginToEdit)
  const pluginToEdit = useStore((state) => state.pluginToEdit)
  const isEditMode = useStore((state) => state.isEditMode)
  const setIsEditMode = useStore((state) => state.setIsEditMode)

  const { postPlugin, updatePlugin, getPlugin } = usePluginApi()

  //init plugin only if it is not already initialized
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
    setSubmitResultMessage({ message: "", ok: false })
    let res = isEditMode
      ? updatePlugin(pluginToEdit!)
      : postPlugin(pluginToEdit!)

    await res.then(async (res) => {
      console.log("submit result", res)
      if (res.ok) {
        setSubmitResultMessage({ message: res.message, ok: true })
        // get the plugin again to update it's state (e.g. resourceVersion)
        let pluginRequest = getPlugin(pluginToEdit!.metadata!.name!)
        await pluginRequest.then((pluginResponse) => {
          if (pluginResponse.ok) {
            setPluginToEdit(pluginResponse.plugin)
            setIsEditMode(true)
          } else {
            setSubmitResultMessage({
              ok: false,
              message:
                "Failed to get plugin after update: " + pluginResponse.message,
            })
          }
        })
      } else {
        setSubmitResultMessage({ message: res.message, ok: false })
      }
    })
  }

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
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
                  onBlur={handleChange}
                />
              </FormRow>
              <FormRow>
                <TextInput
                  id="metadata.name"
                  label="Name"
                  placeholder="Name of this Plugin Instance"
                  {...(isEditMode && { disabled: true })}
                  value={pluginToEdit!.metadata!.name}
                  onBlur={handleChange}
                />
              </FormRow>
              <FormRow>
                <ClusterSelect
                  id="spec.clusterName"
                  placeholder="The Cluster this Plugin is to be deployed to."
                  label="Cluster"
                  defaultValue={pluginToEdit!.spec!.clusterName}
                  onChange={handleChange}
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
                        onChange={handleChange}
                      />
                    </FormRow>
                  )
                })}
              </FormSection>
            )}

            <ButtonRow>
              {submitMessage.message != "" && (
                <Message
                  autoDismissTimeout={5000}
                  autoDismiss={submitMessage.ok}
                  variant={submitMessage.ok ? "success" : "error"}
                  text={submitMessage.message}
                />
              )}
              <Button onClick={onSubmit} variant="primary">
                {isEditMode ? "Update Plugin" : "Create Plugin"}
              </Button>
            </ButtonRow>
          </Form>
        </PanelBody>
      )}
    </Panel>
  )
}

export default PluginEdit
