---
title: "PluginPresets"
weight: 3
---

A PluginPreset is used to configure Plugins for a set of Clusters. This allows administrators to define standard configurations for Clusters in the same environment or with similar requirements. Greenhouse will create Plugins based on the PluginPreset for each Cluster that matches the specified selector.

## Example PluginPreset Spec

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: PluginPreset
metadata:
  name: perses-preset
  namespace: example-organization
spec:
  clusterOptionOverrides:
    - clusterName: example-cluster
      overrides:
      - name: perses.serviceMonitor.selfMonitor
        value: true
      - name: perses.serviceMonitor.labels
        value:
          plugin: kube-monitoring
  clusterSelector:
    matchExpressions:
    - key: cluster-type
      operator: In
      values:
      - observability
  deletionPolicy: Delete
  plugin:
    optionValues:
    - name: perses.sidecar.enabled
      value: true
    pluginDefinitionRef:
      kind: ClusterPluginDefinition
      name: perses
    releaseName: perses
    releaseNamespace: kube-monitoring
```

## Writing a PluginPreset Spec

`.spec.plugin` is the template for the Plugins that will be created for each matching Cluster. This field has the same structure as the PluginSpec. Only `.spec.clusterName` is not allowed in the PluginPreset's Plugin template, as the Cluster name is determined by the matching Clusters.

```yaml
spec:
  plugin:
    optionValues:
    - name: perses.sidecar.enabled
      value: true
    pluginDefinitionRef:
      kind: ClusterPluginDefinition
      name: perses
    releaseName: perses
    releaseNamespace: kube-monitoring
```

`.spec.clusterSelector` is a required field that specifies the label selector used to list the Clusters for which Plugins will be created based on this PluginPreset.

```yaml
spec:
  clusterSelector:
    matchExpressions:
    - key: cluster-type
      operator: In
      values:
      - observability
```

| :warning: Changing the `clusterSelector` may result in the creation or deletion of Plugins for Clusters that start or stop matching the selector. |

`.spec.clusterOptionOverrides` is an optional field that can be used to provide per-Cluster overrides for the Plugin's OptionValues. This can be used to customize the configuration of the Plugin for specific Clusters.

```yaml
spec:
  clusterOptionOverrides:
    - clusterName: example-cluster
      overrides:
      - name: perses.serviceMonitor.selfMonitor
        value: true
```

`.spec.deletionPolicy` is an optional field that specifies the behaviour when a PluginPreset is deleted. The possible values are `Delete` and `Retain`. If set to `Delete` (the default), all Plugins created by the PluginPreset will also be deleted when the PluginPreset is deleted. If set to `Retain`, the Plugins will remain after the PluginPreset is deleted or if the Cluster stops matching the selector.

## Next Steps

- [Managing Plugins for multiple clusters](./../../../user-guides/plugin/plugin-management)
- [Plugin reference](./../plugin)
- [PluginDefinition reference](./../plugindefinition)
