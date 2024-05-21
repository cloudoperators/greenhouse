/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import {
  Plugin,
  PluginDefinition,
  PluginOptionValue,
} from "../../../types/types"
import { EditFormData } from "../plugindefinitions/store"

export const initPluginFromFormData = (formData: EditFormData) => {
  let plugin: Plugin = {
    metadata: formData.metadata!,
    kind: "Plugin",
    apiVersion: "greenhouse.sap/v1alpha1",
    spec: formData.spec!,
  }
  return plugin
}

export const initPluginFromPluginDef = (pluginDefinition: PluginDefinition) => {
  // instantiate new empty PluginConfig from Plugin
  let initPlugin: Plugin = {
    metadata: {
      name: pluginDefinition.metadata!.name!,
      namespace: "",
      labels: {},
    },
    kind: "Plugin",
    apiVersion: "greenhouse.sap/v1alpha1",
    spec: {
      pluginDefinition: pluginDefinition.metadata!.name!,
      displayName:
        pluginDefinition.spec?.displayName ?? pluginDefinition.metadata?.name,
      clusterName: "",
      disabled: false,
      optionValues: [],
    },
  }
  pluginDefinition.spec?.options?.forEach((option) => {
    // if we have a default value, add it to the plugin
    // we do not default secrets
    if (
      option.type != "secret" &&
      option.default &&
      !initPlugin.spec?.optionValues!.some((o) => o.name == option.name)
    ) {
      initPlugin.spec?.optionValues!.push({
        name: option.name,
        value: option.default,
      })
    }
  })
  return initPlugin
}

export const initMetadata = (
  pluginDefinition: PluginDefinition
): Plugin["metadata"] => {
  return {
    name: pluginDefinition.metadata!.name!,
    namespace: "",
    labels: {},
  }
}

export const initPluginSpec = (
  pluginDefinition: PluginDefinition
): Plugin["spec"] => {
  let optionValues: PluginOptionValue[] = []
  let spec = {
    pluginDefinition: pluginDefinition.metadata!.name!,
    displayName:
      pluginDefinition.spec?.displayName ?? pluginDefinition.metadata?.name,
    clusterName: "",
    disabled: false,
    optionValues: optionValues,
  }

  pluginDefinition.spec?.options?.forEach((option) => {
    // if we have a default value, add it to the pluginSpec
    // we do not default secrets
    if (option.type != "secret" && option.default) {
      spec.optionValues.push({
        name: option.name,
        value: option.default,
      })
    }
  })

  return spec
}
