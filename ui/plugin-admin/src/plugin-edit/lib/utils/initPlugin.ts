/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { Plugin, PluginDefinition } from "../../../../../types/types"

const initPlugin = (pluginDefinition: PluginDefinition) => {
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
        pluginDefinition.spec?.displayName ??
        pluginDefinition.metadata?.name,
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

export default initPlugin