/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import {
  Button,
  Container,
  Form,
  FormRow,
  FormSection,
  Modal,
  PanelFooter,
  Switch,
  TextInput,
} from "juno-ui-components"
import React from "react"
import {
  LabelSelector,
  PluginDefinition,
  PluginPreset,
} from "../../../types/types"
import useNamespace from "../plugindefinitions/hooks/useNamespace"
import usePluginApi from "../plugindefinitions/hooks/usePluginApi"
import usePluginPresetApi, {
  PluginPresetApiResponse,
} from "../plugindefinitions/hooks/usePluginPresetApi"
import useStore, { EditFormState } from "../plugindefinitions/store"
import { useGlobalsActions } from "../plugins/components/StoreProvider"
import KeyValueInput from "./KeyValueInput"
import ClusterSelect from "./ClusterSelect"
import { OptionInput } from "./OptionInput"
import SubmitResultMessage, { SubmitMessage } from "./SubmitResultMessage"
import handleFormChange from "./handleFormChange"
import { initPluginFromFormData } from "./initPlugin"
import { initPluginPreset } from "./initPluginPreset"

/**
 * This Form Component is used to edit a Plugin or Plugin Preset.
 * We hold state for the following partial components:
 * - metadata
 * - pluginSpec
 * - labelSelectors
 * and construct a Plugin or Plugin Preset object from these states on submit / delete.
 */

interface PluginEditProps {
  pluginDefinition: PluginDefinition
}
const PluginEdit: React.FC<PluginEditProps> = (props: PluginEditProps) => {
  const { namespace } = useNamespace()
  const showEditForm = useStore((state) => state.showEditForm)
  const setShowEditForm = useStore((state) => state.setShowEditForm)

  const editFormState = useStore((state) => state.editFormState)
  const setEditFormState = useStore((state) => state.setEditFormState)

  const isEditMode =
    editFormState == EditFormState.PLUGIN_EDIT ||
    editFormState == EditFormState.PLUGIN_PRESET_EDIT
  const { setPanel } = useGlobalsActions()

  const isPluginPreset =
    editFormState == EditFormState.PLUGIN_PRESET_CREATE ||
    editFormState == EditFormState.PLUGIN_PRESET_EDIT

  const editFormData = useStore((state) => state.editFormData)
  const setEditFormData = useStore((state) => state.setEditFormData)

  const { createPlugin, updatePlugin, deletePlugin } = usePluginApi()
  const {
    getPluginPreset,
    createPluginPreset,
    updatePluginPreset,
    deletePluginPreset,
  } = usePluginPresetApi()

  const changeIsPluginPreset = () => {
    if (isPluginPreset) {
      setEditFormState(
        isEditMode ? EditFormState.PLUGIN_EDIT : EditFormState.PLUGIN_CREATE
      )
    } else {
      if (
        editFormData.metadata!.labels &&
        editFormData.metadata!.labels["greenhouse.sap/pluginpreset"]
      ) {
        setEditFormState(EditFormState.PLUGIN_PRESET_EDIT)
      } else {
        setEditFormState(EditFormState.PLUGIN_PRESET_CREATE)
      }
    }
  }

  const [showConfirmationDialog, setConfirmationDialog] = React.useState(false)

  // initialize labelselector in formData if it is not set
  React.useEffect(() => {
    if (isPluginPreset && !editFormData.labelSelector) {
      setEditFormData({
        ...editFormData,
        labelSelector: {
          "": "",
        },
      })
    }
  }, [isPluginPreset, editFormData.labelSelector])

  // if metadata labels contain a label with key greenhouse.sap/pluginpreset
  // make sure isPluginPreset is set to true
  React.useEffect(() => {
    if (!editFormData.metadata) return
    if (
      editFormData.metadata!.labels &&
      editFormData.metadata!.labels["greenhouse.sap/pluginpreset"]
    ) {
      setSubmitResultMessage({
        message:
          "This Plugin is part of a Preset. You are now editing the Preset!",
        ok: false,
        variant: "warning",
      })

      setEditFormState(EditFormState.PLUGIN_PRESET_EDIT)

      // get the kubernetes resource
      let pluginPresetPromise = getPluginPreset({
        metadata: {
          name: editFormData.metadata!.labels["greenhouse.sap/pluginpreset"],
          namespace: namespace,
        },
        kind: "PluginPreset",
      })
      pluginPresetPromise
        .then((res) => {
          if (res.ok) {
            setEditFormData({
              metadata: res.response!.metadata,
              spec: res.response!.spec!.plugin,
              labelSelector: res.response!.spec!.clusterSelector.matchLabels,
            })
          } else {
            setEditFormState(EditFormState.PLUGIN_PRESET_CREATE)
            setSubmitResultMessage({
              message:
                "This Plugin seems to be part of a Preset, but the Preset could not be found. You are now creating a new Preset!",
              ok: false,
              variant: "warning",
            })
          }
          return
        })
        .catch((e) => {
          setSubmitResultMessage({
            message: e.message,
            ok: false,
            variant: "error",
          })
          return
        })
      // make sure to set metadata.name to the name of the plugin preset
      setEditFormData({
        ...editFormData,
        metadata: {
          ...editFormData.metadata,
          name: editFormData.metadata!.labels["greenhouse.sap/pluginpreset"],
        },
      })
    }
  }, [editFormData.metadata?.labels])

  const kindName = isPluginPreset ? "Plugin Preset" : "Plugin"

  const onPanelClose = () => {
    setShowEditForm(false)
  }

  const [submitMessage, setSubmitResultMessage] = React.useState<SubmitMessage>(
    { message: "", ok: false }
  )
  const onSubmit = async () => {
    if (isPluginPreset) {
      let pluginPreset: PluginPreset = initPluginPreset(editFormData)

      let pluginPresetCreatePromise: Promise<PluginPresetApiResponse>
      if (editFormState == EditFormState.PLUGIN_PRESET_CREATE) {
        pluginPresetCreatePromise = createPluginPreset({
          ...pluginPreset,
          metadata: {
            name: pluginPreset.metadata!.name,
          },
        })
      } else {
        pluginPresetCreatePromise = updatePluginPreset(pluginPreset)
      }

      await pluginPresetCreatePromise.then(async (res) => {
        setSubmitResultMessage({ message: res.message, ok: res.ok })
      })
    } else {
      let plugin = initPluginFromFormData(editFormData)
      let pluginCreatePromise = isEditMode
        ? updatePlugin(plugin)
        : createPlugin({ ...plugin, metadata: { name: plugin.metadata!.name } })

      await pluginCreatePromise.then(async (res) => {
        setSubmitResultMessage({ message: res.message, ok: res.ok })
      })
    }
  }

  const clickDelete = () => {
    setConfirmationDialog(true)
  }

  const onDelete = async () => {
    setConfirmationDialog(false)
    if (isPluginPreset) {
      let res = await deletePluginPreset(initPluginPreset(editFormData))
      setSubmitResultMessage({ message: res.message, ok: res.ok })
    } else {
      let res = await deletePlugin(initPluginFromFormData(editFormData))
      setSubmitResultMessage({ message: res.message, ok: res.ok })
    }
  }

  const onMessageDismiss = (ok: boolean) => {
    if (ok) {
      setShowEditForm(false)
      console.log("I want to open the details for my plugin now :)")
    }
  }

  const handleFormElementChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    try {
      setEditFormData(handleFormChange(e, editFormData))
    } catch (e) {
      console.error(e)
    }
  }

  const setLabelSelector = (labelSelector: LabelSelector) => {
    setEditFormData({
      ...editFormData,
      labelSelector: labelSelector,
    })
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
      {editFormData && !submitMessage.ok && (
        <>
          <Form
            title={
              editFormData.spec?.displayName ?? editFormData.metadata?.name
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
                  value={editFormData?.spec?.displayName}
                  onBlur={handleFormElementChange}
                />
              </FormRow>
              <FormRow>
                <TextInput
                  id="metadata.name"
                  label="Name"
                  placeholder="Name of this Plugin Instance"
                  {...(isEditMode && { disabled: true })}
                  value={editFormData.metadata?.name}
                  onBlur={handleFormElementChange}
                />
              </FormRow>
              <FormRow>
                {isPluginPreset && (
                  <KeyValueInput
                    data={editFormData.labelSelector}
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
                    defaultValue={editFormData.spec?.clusterName}
                    onChange={handleFormElementChange}
                  />
                )}
              </FormRow>
              <FormRow>
                <TextInput
                  id="spec.releaseNamespace"
                  label="Release Namespace"
                  placeholder={`The namespace in the remote cluster to which the backend is deployed to. Defaults to ${namespace}.`}
                  value={editFormData.spec?.releaseNamespace}
                  onBlur={handleFormElementChange}
                ></TextInput>
              </FormRow>
            </FormSection>

            {props.pluginDefinition.spec?.options?.length && (
              <FormSection title="Options">
                {props.pluginDefinition.spec?.options?.map((option, index) => {
                  let optionValue = editFormData.spec?.optionValues?.find(
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
          </Form>

          <PanelFooter>
            {isEditMode ? (
              <>
                <Button onClick={clickDelete} variant="primary-danger">
                  Delete {kindName}
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
              {isEditMode ? `Update ${kindName}` : `Create ${kindName}`}
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
