import { Plugin, PluginDefinition } from "../../../../../types/types"

const initPlugin = (pluginDefinition: PluginDefinition) => {
  // instantiate new empty PluginConfig from Plugin
  let initPlugin: Plugin = {
    metadata: {
      name: pluginDefinition.metadata!.name!,
      namespace: "",
      labels: {},
    },
    kind: "PluginConfig",
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
    if (
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