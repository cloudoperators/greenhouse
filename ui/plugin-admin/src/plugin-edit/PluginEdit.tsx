/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import {
  Button,
  Form,
  FormRow,
  FormSection,
  Container,
  PanelFooter,
  Modal,
  TextInput,
} from "juno-ui-components"
import React, { useState } from "react"
import { PluginDefinition } from "../../../types/types"
import usePluginApi from "../plugindefinitions/hooks/usePluginApi"
import useStore from "../plugindefinitions/store"
import { useGlobalsActions } from "../plugins/components/StoreProvider"

import ClusterSelect from "./ClusterSelect"
import { OptionInput } from "./OptionInput"
import handleFormChange from "./lib/utils/handleFormChange"
import initPlugin from "./lib/utils/initPlugin"
import SubmitResultMessage, { SubmitMessage } from "./SubmitResultMessage"

interface PluginEditProps {
  pluginDefinition: PluginDefinition
}

// TODO: Hickup observed with resource states: Try getting plugin from server after failed post/put
// TODO: Validate JSON on list/map inputs
const PluginEdit: React.FC<PluginEditProps> = (props: PluginEditProps) => {
  const setShowPluginEdit = useStore((state) => state.setShowPluginEdit)
  const { setPanel } = useGlobalsActions()
  const pluginToEdit = useStore((state) => state.pluginToEdit)
  const setPluginToEdit = useStore((state) => state.setPluginToEdit)

  const isEditMode = useStore((state) => state.isPluginEditMode)
  const setIsEditMode = useStore((state) => state.setIsPluginEditMode)

  const { createPlugin, updatePlugin, deletePlugin } = usePluginApi()

  const [showConfirmationDialog, setConfirmationDialog] = useState(false)

  React.useEffect(() => {
    if (!pluginToEdit) {
      setPluginToEdit(initPlugin(props.pluginDefinition))
    }
  }, [props.pluginDefinition])

  const [submitMessage, setSubmitResultMessage] = React.useState<SubmitMessage>(
    { message: "", ok: false }
  )
  const onSubmit = async () => {
    let pluginCreatePromise = isEditMode
      ? updatePlugin(pluginToEdit!)
      : createPlugin(pluginToEdit!)

    await pluginCreatePromise.then(async (res) => {
      setSubmitResultMessage({ message: res.message, ok: res.ok })
    })
  }

  const clickDelete = () => {
    setConfirmationDialog(true)
  }

  const onDelete = async () => {
    setConfirmationDialog(false)
    let res = await deletePlugin(pluginToEdit!)
    setSubmitResultMessage({ message: res.message, ok: res.ok })
  }

  const onMessageDismiss = (ok: boolean) => {
    if (ok) {
      setShowPluginEdit(false)
      setPluginToEdit(undefined)
      setIsEditMode(false)
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
    <Container px={false} py>
      {submitMessage.message != "" && (
        <FormRow>
          <SubmitResultMessage
            submitMessage={submitMessage}
            onMessageDismiss={() => onMessageDismiss(submitMessage.ok)}
          />
        </FormRow>
      )}
      {pluginToEdit && !submitMessage.ok && (
        <>
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
                        onChange={handleFormElementChange}
                      />
                    </FormRow>
                  )
                })}
              </FormSection>
            )}
          </Form>

          <PanelFooter>
            {isEditMode ? (
              <>
                <Button onClick={clickDelete} variant="primary-danger">
                  Delete Plugin
                </Button>
                {showConfirmationDialog && (
                  <Modal
                    cancelButtonLabel="Cancel"
                    confirmButtonLabel="Proceed irreversible deletion"
                    onCancel={() => setConfirmationDialog(false)}
                    onConfirm={onDelete}
                    open={true}
                    title="Confirmation needed"
                  >
                    <p>
                      Proceeding will result in the permanent loss of the
                      plugin.
                    </p>
                  </Modal>
                )}
              </>
            ) : (
              <></>
            )}
            <Button onClick={onSubmit} variant="primary">
              {isEditMode ? "Update Plugin" : "Create Plugin"}
            </Button>
          </PanelFooter>
        </>
      )}
      {submitMessage.ok && (
        <PanelFooter>
          <Button onClick={() => setPanel(null)}>Close</Button>
        </PanelFooter>
      )}
    </Container>
  )
}

export default PluginEdit
