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
  TextInput,
} from "juno-ui-components"
import React from "react"
import SubmitResultMessage, {
  SubmitMessage,
} from "../plugin-edit/SubmitResultMessage"
import useSecretApi from "../plugindefinitions/hooks/useSecretApi"
import useStore from "../plugindefinitions/store"
import useSecretEditForm from "./handleSecretFormChange"
import KeyValueInput from "./KeyValueInput"

const SecretEdit: React.FC<any> = () => {
  const setShowSecretEdit = useStore((state) => state.setShowSecretEdit)
  const setSecretDetail = useStore((state) => state.setSecretDetail)
  const secretDetail = useStore((state) => state.secretDetail)
  const isSecreEditMode = useStore((state) => state.isSecretEditMode)
  const setIsSecretEditMode = useStore((state) => state.setIsSecretEditMode)

  const { createSecret, updateSecret, deleteSecret } = useSecretApi()

  const [submitMessage, setSubmitResultMessage] = React.useState<SubmitMessage>(
    { message: "", ok: false }
  )

  const handleNameChange = (value: string) => {
    setSecretDetail({
      ...secretDetail,
      metadata: {
        ...secretDetail?.metadata,
        name: value,
      },
    })
  }

  const setSecretData = (data: { [key: string]: string }) => {
    setSecretDetail({
      ...secretDetail,
      data: data,
    })
  }

  const base64Endcode = (value: string) => {
    return btoa(value)
  }

  const onPanelClose = () => {
    setShowSecretEdit(false)
    setSecretDetail(undefined)
  }
  const onDelete = async () => {
    let res = await deleteSecret(secretDetail!)
    setSubmitResultMessage({ message: res.message, ok: res.ok })
  }
  const onSubmit = () => {
    let secretCreatePromise = isSecreEditMode
      ? updateSecret(secretDetail!)
      : createSecret(secretDetail!)

    secretCreatePromise.then(async (res) => {
      setSubmitResultMessage({ message: res.message, ok: res.ok })
      if (res.ok) {
        setIsSecretEditMode(false)
      }
    })
  }

  return (
    <Panel
      heading={
        <Stack gap="2">
          <span>Edit Secret</span>
        </Stack>
      }
      opened={!!secretDetail}
      onClose={onPanelClose}
      size="large"
    >
      <PanelBody>
        <Form title={secretDetail?.metadata?.name}>
          <FormSection title="General">
            <FormRow>
              <TextInput
                id="name"
                label="Name"
                placeholder="Name of this secret"
                {...(isSecreEditMode && { disabled: true })}
                value={secretDetail?.metadata!.name}
                onBlur={(e) => handleNameChange(e.target.value)}
              />
            </FormRow>
          </FormSection>
          <KeyValueInput
            data={secretDetail!.data}
            setData={setSecretData}
            mutateValue={base64Endcode}
          ></KeyValueInput>
          <Stack distribution="between">
            <Button onClick={onDelete} variant="primary-danger">
              Delete Secret
            </Button>
            {submitMessage.message != "" && (
              <SubmitResultMessage submitMessage={submitMessage} />
            )}
            <Button onClick={onSubmit} variant="primary">
              {isSecreEditMode ? "Update Secret" : "Create Secret"}
            </Button>
          </Stack>
        </Form>
      </PanelBody>
    </Panel>
  )
}

export default SecretEdit
