/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React from "react"
import {
  Button,
  Stack,
  TextInput,
  FormSection,
  InputGroup,
  FormRow,
} from "juno-ui-components"
/*
 * This Element provides a form section for entering and editing key-value pairs.
 * The key-value data and the setData function are passed as props.
 * A mutateValue function can be passed as a prop to modify the value before it is stored.
 */

export interface KeyValueInputProps {
  data?: { [key: string]: string }
  setData: (data: { [key: string]: string }) => void
  mutateValue?: (value: string) => string
  title?: string
  dataName?: string
  isSecret?: boolean
}
const KeyValueInput: React.FC<KeyValueInputProps> = (
  props: KeyValueInputProps
) => {
  const dataName = props.dataName ? props.dataName : "Data"
  const isSecret = props.isSecret ? props.isSecret : false
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

  const addNewDataEntry = () => {
    props.setData({
      ...props.data,
      "": "",
    })
  }

  return (
    <FormSection title={props.title}>
      {props.data &&
        Object.keys(props.data).length > 0 &&
        Object.keys(props.data).map((dataKey) => (
          <FormRow>
            <InputGroup>
              <TextInput
                id={"dataKey." + dataKey}
                label={`${dataName} Key`}
                value={dataKey}
                onBlur={(e) =>
                  handleDataEntryChange("dataKey." + dataKey, e.target.value)
                }
              />

              <TextInput
                id={"dataValue" + dataKey}
                type={isSecret ? "password" : "text"}
                label={`${dataName} Value`}
                value={props.data![dataKey]}
                onBlur={(e) =>
                  handleDataEntryChange("dataValue." + dataKey, e.target.value)
                }
              />
              <Button
                icon="deleteForever"
                onClick={() => deleteDataEntry(dataKey)}
              />
            </InputGroup>
          </FormRow>
        ))}
      <Button
        icon="addCircle"
        label={`Add ${dataName}`}
        onClick={addNewDataEntry}
      />
    </FormSection>
  )
}

export default KeyValueInput
