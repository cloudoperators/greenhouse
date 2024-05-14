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

const SecretEdit: React.FC<any> = () => {
  const setShowSecretEdit = useStore((state) => state.setShowSecretEdit)
  const setSecretDetail = useStore((state) => state.setSecretDetail)
  const secretDetail = useStore((state) => state.secretDetail)
  const isSecreEditMode = useStore((state) => state.isSecretEditMode)
  const setIsSecretEditMode = useStore((state) => state.setIsSecretEditMode)

  const { handleSecretFormChange, deleteDataEntry } = useSecretEditForm()

  const { createSecret, updateSecret, deleteSecret } = useSecretApi()

  const [submitMessage, setSubmitResultMessage] = React.useState<SubmitMessage>(
    { message: "", ok: false }
  )

  const handleFormElementChange = (key, value: string) => {
    handleSecretFormChange(key, value)
  }

  const deleteData = (key, value: string) => {
    deleteDataEntry(key)
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

  const addData = () => {
    setSecretDetail({
      ...secretDetail,
      data: {
        ...secretDetail?.data,
        "": "",
      },
    })
  }
  console.log("secretDetail", secretDetail)
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
                onBlur={(e) => handleFormElementChange("name", e.target.value)}
              />
            </FormRow>
          </FormSection>

          <FormSection title="Data">
            {secretDetail!.data &&
              Object.keys(secretDetail!.data!).length > 0 &&
              Object.keys(secretDetail!.data!).map((dataKey) => (
                <Stack key={dataKey} distribution="evenly">
                  <TextInput
                    id={"dataKey." + dataKey}
                    label="Data Key"
                    placeholder="Key of this secret data entry"
                    value={dataKey}
                    onBlur={(e) =>
                      handleFormElementChange(
                        "dataKey." + dataKey,
                        e.target.value
                      )
                    }
                  />

                  <TextInput
                    id={"dataValue" + dataKey}
                    type="password"
                    label="Data Value"
                    placeholder="Value of this secret data entry"
                    value={secretDetail!.data![dataKey]}
                    onBlur={(e) =>
                      handleFormElementChange(
                        "dataValue." + dataKey,
                        e.target.value
                      )
                    }
                  />
                  <Button
                    icon="deleteForever"
                    label="Remove entry"
                    onClick={() =>
                      deleteData(dataKey, secretDetail!.data![dataKey])
                    }
                  />
                </Stack>
              ))}
            <Button icon="addCircle" label="Add Data" onClick={addData} />
          </FormSection>

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
