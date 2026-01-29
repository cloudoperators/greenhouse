---
title: "Plugins"
weight: 4
---

A Plugin is an instance of a PluginDefinition and is used to deploy infrastructure components such as observability, compliance or system components to a Kubernetes cluster managed with Greenhouse. A Plugin provides the specific configuration for deploying the Helm chart associated with the referenced PluginDefinition to a specific cluster.

## Example Plugin Spec

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: Plugin
metadata:
  name: alerts-plugin
  namespace: example-organization
spec:
  clusterName: example-cluster
  displayName: Example Alerts Plugin
  optionValues:
  - name: image.tag
    value: foobar
  pluginDefinitionRef:
    kind: PluginDefinition
    name: alerts
  releaseName: alerts
  releaseNamespace: kube-monitoring
```

## Writing a Plugin Spec

`.spec.displayName` is an optional human-readable name that is used to display the Plugin in the Greenhouse UI. If not provided, it defaults to the value of `metadata.name`.

`.spec.clusterName` is the name of the Cluster resource where the Helm chart associated with the PluginDefinition will be deployed.

`.spec.pluginDefinitionRef` is the required and immutable reference to a PluginDefinition resource that defines the Helm chart and UI application associated with this Plugin.

```yaml
spec:
  pluginDefinitionRef:
    kind: PluginDefinition
    name: alerts
```

`.spec.releaseName` is the optional and immutable name of the Helm release that will be created when deploying the Plugin to the target cluster. If not provided it defaults to the name of the PluginDefinition's Helm chart.

`.spec.releaseNamespace` is the optional and immutable Kubernetes namespace in the target cluster where the Helm release will be deployed. If not provided, it defaults to the name of the Organization.

`.spec.optionValues` is an optional list of Helm chart values that will be used to customize the deployment of the Helm chart associated with the PluginDefinition. These values are used to set Options required by the PluginDefinition or to override provided default values.

```yaml
  optionValues:
  - name: image.tag
    value: foobar
  - name: secret
    valueFrom:
      secret:
        name: alerts-secret
        key: secret-key
```

| :information_source: A defaulting webhook automatically merges the OptionValues with the defaults set in the PluginDefinition. The defaulting does not update OptionValues when the defaults change and does not remove values when they are removed from the PluginDefinition. |

`.spec.optionValues[].expression` is an optional field that allows you to define dynamic values using [CEL (Common Expression Language)](https://github.com/google/cel-spec) expressions. Expressions use `${...}` placeholders that reference `global.greenhouse.*` variables such as `global.greenhouse.clusterName` or [Cluster Metadata](./../cluster#setting-metadata-labels) via `global.greenhouse.metadata.*`. For available CEL string functions, see the [CEL string extension documentation](https://github.com/google/cel-go/tree/master/ext#strings). See [Using Metadata Labels and Expressions](./../../user-guides/plugin/metadata-expressions) for detailed examples.

```yaml
  optionValues:
  - name: endpoint
    expression: "https://api.${global.greenhouse.metadata.region}.example.com"
```

| :warning: CEL expression evaluation requires the `expressionEvaluationEnabled` feature flag to be enabled. When disabled, expressions are treated as literal strings. |

`.spec.waitFor` is an optional field that specifies PluginPresets or Plugins which have to be successfully deployed before this Plugin can be deployed. This can be used to express dependencies between Plugins. This can be useful if one Plugin depends on Custom Resource Definitions or other resources created by another Plugin.

```yaml
spec:
  waitFor:
  - pluginRef:
      pluginPreset: ingress-nginx
  - pluginRef:
      name: cert-manager-example-cluster
```

| :information_source: The dependency on a PluginPreset ensures that a Plugin created by this PluginPreset has been deployed to the same cluster. The dependency on a Plugin is fulfilled if the referenced Plugin is deployed to the same cluster. |

`.spec.ignoreDifferences` is an optional field that is used to suppress specific differences detected by the drift detection of the deployment tool. This can be useful to ignore differences in fields that are managed by other controllers or tools. [Example configuration](https://fluxcd.io/flux/components/helm/helmreleases/#ignore-rules) when using Flux as the deployment tool.

```yaml
spec:
  ignoreDifferences:
  - group: apps
    version: v1
    kind: Deployment
    paths:
    - /spec/replicas
```

| :warning: The `ignoreDifferences` field is only supported when using Flux as the deployment tool. It is ignored when using the legacy Helm controller. |

## Working with Plugins

### Choosing the deployment tool

The annotation `greenhouse.sap/deployment-tool` can be added to a Plugin resource to choose the deployment tool used to deploy the Helm release. Supported values are `flux` and `legacy`.

### Suspending the Plugin's reconciliation

The annotation `greenhouse.sap/suspend` can be added to a Plugin resource to temporarily suspend the reconciliation of the Plugin. This results in no changes on the Plugin or referenced resources being applied until the annotation is removed. This also includes upgrades of the Helm release on the target cluster. This also blocks the deletion of the Plugin resource until the annotation is removed.

### Triggering reconciliation of the Plugin's managed resources

The annotation `greenhouse.sap/reconcile` can be added to a Plugin resource to trigger a reconciliation of the Plugin and its managed resources. When the Plugin is deployed using FluxCD this annotation is propagated to the Flux HelmRelease resource and triggers a reconciliation of the Helm release on the target cluster. This can be useful to trigger a reconciliation even if no changes were made to the Plugin resource.

## Next Steps

- [Cluster reference](./../cluster)
- [PluginPreset reference](./../pluginpreset)
- [PluginDefinition reference](./../plugindefinition)
- [Using Metadata Labels and Expressions](./../../user-guides/plugin/metadata-expressions)
