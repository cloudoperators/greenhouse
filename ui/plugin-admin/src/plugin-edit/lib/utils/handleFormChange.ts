import { Plugin, PluginDefinition } from "../../../../../types/types"

const handleFormChange = (
  e: React.ChangeEvent<HTMLInputElement>,
  plugin: Plugin
  ): Plugin => {
  let value: string | boolean | number
  if (e.target?.type == undefined) {
    throw new Error("Unexpected form change event: "+JSON.stringify(e) )
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

  // the incoming id consists of the path to the property in the plugin object separated by dots
  if (e.target.id.startsWith("metadata.")) {
    return {
      ...plugin,
      metadata: {
        ...plugin.metadata!,
        [e.target.id.split(".")[1]]: value ,
      },
    }
  } else if (e.target.id.startsWith("spec.")) {
    return {
      ...plugin,
      spec: {
        ...plugin.spec!,
        [e.target.id.split(".")[1]]: value,
      },
    }
  } else if (e.target.id.startsWith("optionValues.")) {
    
    // delete from plugin.spec.optionValues by matching name property if value is empty
    if (value == "") {
      return {
        ...plugin,
        spec: {
          ...plugin.spec!,
          optionValues: plugin.spec!.optionValues!.filter(
            (option) => option.name != e.target.id.split(".")[1]
          ),
        },
      }
    }
    //   replace in plugin.spec.optionValues by matching name property or push if not found
    let wasFound = false
    let changedPlugin: Plugin
    changedPlugin ={
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
    }
    if (!wasFound) {
      changedPlugin ={
        ...plugin,
        spec: {
          ...plugin.spec!,
          optionValues: [
            ...plugin.spec!.optionValues!,
            { name: e.target.id.split(".")[1], value: value },
          ],
        },
      }
    }
    return changedPlugin
  }
  return plugin
}

export default handleFormChange