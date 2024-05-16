/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { Checkbox, TextInput, Textarea } from "juno-ui-components"
import { PluginDefinitionOption, PluginOptionValue } from "../../../types/types"
import SecretKeySelect from "./SecretKeySelect"

interface OptionInputProps {
  pluginDefinitionOption: PluginDefinitionOption
  pluginOptionValue?: PluginOptionValue
  onChange?: (e: React.ChangeEvent<HTMLInputElement>) => void
}

export const OptionInput: React.FC<OptionInputProps> = (
  props: OptionInputProps
) => {
  const handleBlur = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (props.onChange) {
      props.onChange(e)
    }
  }
  const id = "optionValues." + props.pluginDefinitionOption.name
  const label =
    (props.pluginDefinitionOption.displayName ??
      props.pluginDefinitionOption.name) +
    ` (${props.pluginDefinitionOption.type})`
  const required = props.pluginDefinitionOption.required

  // values have already been defaulted on initPlugin
  let value = props.pluginOptionValue?.value
  let valueFrom = props.pluginOptionValue?.valueFrom

  let type = "text"

  switch (props.pluginDefinitionOption.type) {
    case "bool":
      return (
        <Checkbox
          id={id}
          label={label}
          required={required}
          checked={value}
          onBlur={handleBlur}
        />
      )
    case "list":
    case "map":
      return (
        <Textarea
          id={id}
          label={label}
          required={required}
          value={JSON.stringify(value)}
          onBlur={handleBlur}
        ></Textarea>
      )
    case "secret":
      return (
        <SecretKeySelect
          id={id}
          label={label}
          required={required}
          value={valueFrom}
          onChange={handleBlur}
        />
      )
    case "int":
      type = "number"
      break
  }
  return (
    <TextInput
      id={id}
      type={type}
      label={label}
      value={value}
      required={required}
      onBlur={handleBlur}
    />
  )
}
