/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { PluginOptionValue } from "../../../types/types"
import { EditFormData } from "../plugindefinitions/store"

const handleFormChange = (
  e: React.ChangeEvent<HTMLInputElement>,
  editFormData: EditFormData
): EditFormData => {
  let value: string | boolean | number | undefined = undefined
  let valueFrom: PluginOptionValue["valueFrom"] | undefined = undefined

  if (e.target?.type == undefined) {
    throw new Error("Unexpected form change event: " + JSON.stringify(e))
  }
  // remove everything before the first dot
  const optionId = e.target.id.split(".").slice(1).join(".")
  switch (e.target.type) {
    case "checkbox":
      value = e.target.checked
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
      ...editFormData,
      metadata: {
        ...editFormData.metadata!,
        [optionId]: value,
      },
    }
  } else if (e.target.id.startsWith("spec.")) {
    return {
      ...editFormData,
      spec: {
        ...editFormData.spec!,
        [optionId]: value,
      },
    }
  } else if (e.target.id.startsWith("optionValues.")) {
    //   replace in plugin.spec.optionValues by matching name property or push if not found
    let wasFound = false
    let optionValueToSet: PluginOptionValue = { name: optionId }
    if (value != undefined) {
      optionValueToSet.value = value
    }
    if (valueFrom != undefined) {
      optionValueToSet.valueFrom = valueFrom
    }
    let changedEditFormData: EditFormData
    changedEditFormData = {
      ...editFormData,
      spec: {
        ...editFormData.spec!,
        optionValues: editFormData.spec!.optionValues!.map((option) => {
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
      changedEditFormData = {
        ...editFormData,
        spec: {
          ...editFormData.spec!,
          optionValues: [...editFormData.spec!.optionValues!, optionValueToSet],
        },
      }
    }
    return changedEditFormData
  }
  return editFormData
}

export default handleFormChange
