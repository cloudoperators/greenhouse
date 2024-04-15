import { Plugin, PluginDefinition } from "../../../../../types/types"

const handleFormChange = (
  e: React.ChangeEvent<HTMLInputElement>,
  plugin: Plugin,
  setPlugin: React.Dispatch<React.SetStateAction<Plugin>>
  ) => {
  let value: string | boolean | number
  if (e.target?.type == undefined) {
    console.error("Unexpected form change event: " )
    console.error(e)
    return
  }
  switch (e.target.type) {
    case "checkbox":
      value = e.target.checked ? true : false
      break
    case "number":
      value = parseInt(e.target.value)
      break
    case "textarea":
      value = JSON.parse(e.target.value)
      break
    default:
      value = e.target.value
      break
  }

  if (e.target.id.startsWith("metadata.")) {
    setPlugin({
      ...plugin,
      metadata: {
        ...plugin.metadata!,
        [e.target.id.split(".")[1]]: value,
      },
    })
  } else if (e.target.id.startsWith("spec.")) {
    setPlugin({
      ...plugin,
      spec: {
        ...plugin.spec!,
        [e.target.id.split(".")[1]]: value,
      },
    })
  } else if (e.target.id.startsWith("optionValues.")) {
    // delete from pluginConfig.spec.optionValues by matching name property if value is empty
    // does not work yet!!
    if (value == "") {
      setPlugin({
        ...plugin,
        spec: {
          ...plugin.spec!,
          optionValues: plugin.spec!.optionValues!.filter(
            (option) => option.name != e.target.id.split(".")[1]
          ),
        },
      })
      console.log(plugin.spec!.optionValues!)
    }
    //   replace in pluginConfig.spec.optionValues by matching name property or push if not found
    let wasFound = false

    setPlugin({
      ...plugin,
      spec: {
        ...plugin.spec!,
        optionValues: plugin.spec!.optionValues!.map((option) => {
          if (option.name == e.target.id.split(".")[1]) {
            wasFound = true
            return { name: option.name, value: value }
          } else {
            return option
          }
        }),
      },
    })
    if (!wasFound) {
      setPlugin({
        ...plugin,
        spec: {
          ...plugin.spec!,
          optionValues: [
            ...plugin.spec!.optionValues!,
            { name: e.target.id.split(".")[1], value: value },
          ],
        },
      })
    }
  }
}

export default handleFormChange