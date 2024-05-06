import { TextInput, Textarea, Checkbox } from "juno-ui-components"
import { PluginDefinitionOption, PluginOptionValue } from "../../../types/types"

interface OptionInputProps {
  pluginDefinitionOption: PluginDefinitionOption
  isEditMode: boolean
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
    props.pluginDefinitionOption.displayName ??
    props.pluginDefinitionOption.name
  const helptext = props.pluginDefinitionOption.type
  const required = props.pluginDefinitionOption.required

  // values have already been defaulted on initPlugin
  let value = props.pluginOptionValue?.value

  let type = "text"

  switch (props.pluginDefinitionOption.type) {
    case "bool":
      return (
        <Checkbox
          id={id}
          label={label}
          required={required}
          helptext={helptext}
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
          helptext={helptext}
          value={JSON.stringify(value)}
          onBlur={handleBlur}
        ></Textarea>
      )
    case "secret":
      type = "password"
      break
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
      helptext={helptext}
      onBlur={handleBlur}
    />
  )
}
