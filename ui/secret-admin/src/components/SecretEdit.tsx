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
import useSecretApi from "../hooks/useSecretApi"
import useStore from "../store"
import KeyValueInput from "./KeyValueInput"
import ResultMessageComponent, { ResultMessage } from "./SubmitResultMessage"

const SecretEdit: React.FC<any> = () => {
  const setShowSecretEdit = useStore((state) => state.setShowSecretEdit)
  const setSecretDetail = useStore((state) => state.setSecretDetail)
  const secretDetail = useStore((state) => state.secretDetail)
  const isSecreEditMode = useStore((state) => state.isSecretEditMode)
  const setIsSecretEditMode = useStore((state) => state.setIsSecretEditMode)

  const { createSecret, updateSecret, deleteSecret } = useSecretApi()

  const [submitMessage, setSubmitResultMessage] = React.useState<ResultMessage>(
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

  const handleTypeChange = (value: string) => {
    setSecretDetail({
      ...secretDetail,
      type: value,
    })
  }

  const setSecretData = (data: { [key: string]: string }) => {
    setSecretDetail({
      ...secretDetail,
      data: data,
    })
  }

  const setSecretLabels = (labels: { [key: string]: string }) => {
    setSecretDetail({
      ...secretDetail,
      metadata: {
        ...secretDetail?.metadata,
        labels: labels,
      },
    })
  }

  const base64Endcode = (value: string) => {
    return btoa(value)
  }

  const onPanelClose = () => {
    setShowSecretEdit(false)
    setSecretDetail(undefined)
    setIsSecretEditMode(false)
  }
  const onDelete = async () => {
    let res = await deleteSecret(secretDetail!)
    setSubmitResultMessage({ message: res.message, ok: res.ok })
  }
  const onSubmit = () => {
    let base64EncodedSecret = { ...secretDetail }
    if (base64EncodedSecret.data) {
      let data = {}
      Object.keys(base64EncodedSecret.data).forEach((key) => {
        data[key] = base64Endcode(base64EncodedSecret.data![key])
      })
      base64EncodedSecret.data = data
    }
    let secretCreatePromise = isSecreEditMode
      ? updateSecret(base64EncodedSecret)
      : createSecret(base64EncodedSecret)

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
            <FormRow>
              <TextInput
                id="type"
                label="Type"
                placeholder='Type of this secret, leave empty for default "Opaque" type'
                {...(isSecreEditMode && { disabled: true })}
                value={secretDetail?.type}
                onBlur={(e) => handleTypeChange(e.target.value)}
              ></TextInput>
            </FormRow>
          </FormSection>
          <KeyValueInput
            title="Labels"
            dataName="Label"
            data={secretDetail!.metadata!.labels}
            setData={setSecretLabels}
          ></KeyValueInput>
          <KeyValueInput
            title="Data"
            data={secretDetail!.data}
            setData={setSecretData}
            isSecret={true}
          ></KeyValueInput>
          <Stack distribution="between">
            <Button onClick={onDelete} variant="primary-danger">
              Delete Secret
            </Button>
            {submitMessage.message != "" && (
              <ResultMessageComponent submitMessage={submitMessage} />
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
