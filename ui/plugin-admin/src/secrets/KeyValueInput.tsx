import React from "react"
import { Button, Stack, TextInput, FormSection } from "juno-ui-components"

export interface KeyValueInputProps {
  data?: { [key: string]: string }
  setData: (data: { [key: string]: string }) => void
  mutateValue?: (value: string) => string
}
const KeyValueInput: React.FC<KeyValueInputProps> = (
  props: KeyValueInputProps
) => {
  const handleDataEntryChange = (key, value: string) => {
    // key is in format dataKey.key or dataValue.key
    let keyInfo = key.split(".")
    let keyIdentifier = keyInfo[0]
    let keyData = keyInfo[1]

    switch (keyIdentifier) {
      case "dataKey":
        // remove entry with old key and add new entry with new key
        let data = { ...props.data }
        let dataValue = data[keyData]
        delete data[keyData]
        data[value] = dataValue
        props.setData(data)
        break
      case "dataValue":
        if (props.mutateValue) {
          value = props.mutateValue(value)
        }
        props.setData({
          ...props.data,
          [keyData]: value,
        })
        break
      default:
        console.log("keyIdentifier not found")
        break
    }
  }

  const deleteDataEntry = (key: string) => {
    let data = { ...props.data }
    delete data[key]
    props.setData(data)
  }

  const addData = () => {
    props.setData({
      ...props.data,
      "": "",
    })
  }

  return (
    <FormSection title="Data">
      {props.data &&
        Object.keys(props.data).length > 0 &&
        Object.keys(props.data).map((dataKey) => (
          <Stack key={dataKey} distribution="evenly">
            <TextInput
              id={"dataKey." + dataKey}
              label="Data Key"
              placeholder="Key of this secret data entry"
              value={dataKey}
              onBlur={(e) =>
                handleDataEntryChange("dataKey." + dataKey, e.target.value)
              }
            />

            <TextInput
              id={"dataValue" + dataKey}
              type="password"
              label="Data Value"
              placeholder="Value of this secret data entry"
              value={props.data![dataKey]}
              onBlur={(e) =>
                handleDataEntryChange("dataValue." + dataKey, e.target.value)
              }
            />
            <Button
              icon="deleteForever"
              label="Remove entry"
              onClick={() => deleteDataEntry(dataKey)}
            />
          </Stack>
        ))}
      <Button icon="addCircle" label="Add Data" onClick={addData} />
    </FormSection>
  )
}

export default KeyValueInput
