import { Plugin, PluginOptionValue, Secret, SecretDataEntry } from "../../../../../types/types"

const handleFormChange = (
  e: React.ChangeEvent<HTMLInputElement>,
  plugin: Plugin
  ): [Plugin, SecretDataEntry?] => {
  let value: string | boolean | number | undefined = undefined
  let secretDataEntry: SecretDataEntry | undefined
  let valueFrom: PluginOptionValue["valueFrom"] | undefined = undefined

  if (e.target?.type == undefined) {
    throw new Error("Unexpected form change event: "+JSON.stringify(e) )
  }
  const optionId = e.target.id.split(".")[1]
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
    case "password":
      valueFrom = {
        secret: {
          key: optionId,
          name: plugin.metadata!.name!,
        },
      }
      secretDataEntry = {
        [optionId]:  e.target.value,
      }
      break
    default:
      value = e.target.value
      break
  }

  // the incoming id consists of the path to the property in the plugin object separated by dots
  if (e.target.id.startsWith("metadata.")) {
    return [{
      ...plugin,
      metadata: {
        ...plugin.metadata!,
        [optionId]: value ,
      },
    }, secretDataEntry]
  } else if (e.target.id.startsWith("spec.")) {
    return [{
      ...plugin,
      spec: {
        ...plugin.spec!,
        [optionId]: value,
      },
    }, secretDataEntry]
  } else if (e.target.id.startsWith("optionValues.")) {
    
    // delete from plugin.spec.optionValues by matching name property if value is empty
    if (value == "" && valueFrom == undefined) {
      return [{
        ...plugin,
        spec: {
          ...plugin.spec!,
          optionValues: plugin.spec!.optionValues!.filter(
            (option) => option.name != optionId
          ),
        },
      }, secretDataEntry]
    }
    //   replace in plugin.spec.optionValues by matching name property or push if not found
    let wasFound = false
    let optionValueToSet: PluginOptionValue = { name: optionId }
    if (value != undefined) {
      optionValueToSet.value = value
    }
    if (valueFrom != undefined) {
      optionValueToSet.valueFrom = valueFrom
    }
    let changedPlugin: Plugin
    changedPlugin ={
      ...plugin,
      spec: {
        ...plugin.spec!,
        optionValues: plugin.spec!.optionValues!.map((option) => {
          if (option.name == optionValueToSet.name) {
            wasFound = true
            return optionValueToSet
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
            optionValueToSet,
          ],
        },
      }
    }
    return [changedPlugin, secretDataEntry]
  }
  return [plugin, secretDataEntry]
}


export default handleFormChange