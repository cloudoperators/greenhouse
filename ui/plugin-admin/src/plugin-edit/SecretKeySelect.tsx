/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import {
  PluginOptionValueFrom,
  Secret,
  SecretDataEntry,
} from "../../../types/types"
import { Stack, Select, SelectOption } from "juno-ui-components"
import usePluginDefinitionsStore from "../plugindefinitions/store"

interface SecretKeySelectProps {
  id: string
  label: string
  required: boolean
  helptext?: string
  value: PluginOptionValueFrom
  onChange: (e: React.ChangeEvent<HTMLInputElement>) => void
}

export const SecretKeySelect: React.FC<SecretKeySelectProps> = (
  props: SecretKeySelectProps
) => {
  const secrets = usePluginDefinitionsStore((state) => state.secrets)
  const selectedSecretName = props.value?.secret?.name ?? ""
  const selectedSecret = secrets.find(
    (secret) => secret.metadata!.name === selectedSecretName
  )
  const selectedSecretKey =
    props.value?.secret?.key != undefined &&
    selectedSecret != undefined &&
    selectedSecret!.data![props.value.secret.key] != undefined
      ? props.value?.secret?.key
      : ""

  let valueFrom = {
    secret: {
      name: selectedSecretName,
      key: selectedSecretKey,
    },
  }

  const handleSecretNameChange = (value: string): void => {
    valueFrom.secret.name = value
    handleFormChange(valueFrom)
  }
  const handleSecretKeyChange = (value: string): void => {
    valueFrom.secret.key = value
    handleFormChange(valueFrom)
  }
  const handleFormChange = (valueFrom: PluginOptionValueFrom): void => {
    let e = {
      target: {
        value: JSON.stringify(valueFrom),
        id: props.id,
        type: "secret-select",
      },
    } as React.ChangeEvent<HTMLInputElement>
    if (props.onChange) {
      props.onChange(e)
    }
  }

  return (
    <Stack>
      <Select
        id={props.id + "-name"}
        label={props.label}
        required={props.required}
        helptext="Select secret"
        value={selectedSecretName}
        onChange={handleSecretNameChange}
      >
        {secrets.map((secret) => {
          return (
            <SelectOption
              key={secret.metadata!.name}
              value={secret.metadata!.name}
            />
          )
        })}
      </Select>

      <Select
        disabled={!selectedSecretName}
        id={props.id + "-key"}
        label={props.label}
        required={props.required}
        helptext="Select secret data key"
        value={selectedSecretKey}
        onChange={handleSecretKeyChange}
      >
        {selectedSecret &&
          Object.keys(selectedSecret.data!).map((key) => {
            return (
              <SelectOption key={key} value={key}>
                {key}
              </SelectOption>
            )
          })}
      </Select>
    </Stack>
  )
}

export default SecretKeySelect
