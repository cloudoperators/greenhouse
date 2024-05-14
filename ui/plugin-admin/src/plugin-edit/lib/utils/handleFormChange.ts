/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import {
  Plugin,
  PluginOptionValue,
  Secret,
  SecretDataEntry,
} from "../../../../../types/types"

const handleFormChange = (
  e: React.ChangeEvent<HTMLInputElement>,
  plugin: Plugin
): Plugin => {
  let value: string | boolean | number | undefined = undefined
  let secretDataEntry: SecretDataEntry | undefined
  let valueFrom: PluginOptionValue["valueFrom"] | undefined = undefined

  console.log("e.target", e.target)

  if (e.target?.type == undefined) {
    throw new Error("Unexpected form change event: " + JSON.stringify(e))
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
    case "secret-select":
      valueFrom = JSON.parse(e.target.value)
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
        [optionId]: value,
      },
    }
  } else if (e.target.id.startsWith("spec.")) {
    return {
      ...plugin,
      spec: {
        ...plugin.spec!,
        [optionId]: value,
      },
    }
  } else if (e.target.id.startsWith("optionValues.")) {
    // delete from plugin.spec.optionValues by matching name property if value is empty
    if (value == "" && valueFrom == undefined) {
      return {
        ...plugin,
        spec: {
          ...plugin.spec!,
          optionValues: plugin.spec!.optionValues!.filter(
            (option) => option.name != optionId
          ),
        },
      }
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
    console.log("optionValueToSet", optionValueToSet)
    let changedPlugin: Plugin
    changedPlugin = {
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
      changedPlugin = {
        ...plugin,
        spec: {
          ...plugin.spec!,
          optionValues: [...plugin.spec!.optionValues!, optionValueToSet],
        },
      }
    }
    return changedPlugin
  }
  return plugin
}

export default handleFormChange
