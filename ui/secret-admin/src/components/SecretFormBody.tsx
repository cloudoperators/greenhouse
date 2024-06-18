import { Form, FormRow, FormSection, TextInput } from "juno-ui-components"
import useStore from "../store"
import KeyValueInput from "./KeyValueInput"

const SecretFormBody: React.FC = () => {
  const setSecretDetail = useStore((state) => state.setSecretDetail)
  const secretDetail = useStore((state) => state.secretDetail)
  const isSecretEditMode = useStore((state) => state.isSecretEditMode)

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

  return (
    <Form>
      <FormSection title="General">
        <FormRow>
          <TextInput
            id="name"
            label="Name"
            placeholder="Name of this secret"
            {...(isSecretEditMode && { disabled: true })}
            value={secretDetail?.metadata!.name}
            onBlur={(e) => handleNameChange(e.target.value)}
          />
        </FormRow>
        <FormRow>
          <TextInput
            id="type"
            label="Type"
            placeholder='Type of this secret, leave empty for default "Opaque" type'
            {...(isSecretEditMode && { disabled: true })}
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
    </Form>
  )
}

export default SecretFormBody
