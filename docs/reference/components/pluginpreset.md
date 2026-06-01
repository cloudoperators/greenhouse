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
    - name: perses.ingress.host
      expression: |
        "perses.${global.greenhouse.clusterName}.example.com"
    pluginDefinitionRef:
      kind: ClusterPluginDefinition
      name: perses
    releaseName: perses
    releaseNamespace: kube-monitoring
```

## Writing a PluginPreset Spec

`.spec.plugin` is the template for the Plugins that will be created for each matching Cluster. This field has the same structure as the PluginSpec. Only `.spec.clusterName` is not allowed in the PluginPreset's Plugin template, as the Cluster name is determined by the matching Clusters.

> :information_source: A non-existing PluginDefinition can be referenced in the PluginPreset. The PluginPreset will be reconciled once the PluginDefinition is created. This allows rolling out new PluginDefinitions via a Catalog together with the PluginPresets that reference them.

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

## CEL Expressions in OptionValues

PluginPresets support CEL (Common Expression Language) expressions in `optionValues`. Expressions are evaluated during PluginPreset reconciliation, and the resulting Plugin contains only the resolved values with no expression fields remaining.

Expressions use the `${...}` syntax to reference dynamic values:

```yaml
spec:
  plugin:
    optionValues:
    - name: app.hostname
      expression: |
        "myapp.${global.greenhouse.clusterName}.example.com"
```

When this PluginPreset creates a Plugin for a cluster named `cluster-a`, the Plugin will contain:

```yaml
spec:
  optionValues:
  - name: app.hostname
    value: "myapp.cluster-a.example.com"
```

### Available Variables

| Variable                             |         Description        | Example Value                |
|--------------------------------------|----------------------------|------------------------------|
| `global.greenhouse.clusterName`      | Name of the target cluster | `cluster-a`                  |
| `global.greenhouse.organizationName` | Organization namespace     | `my-org`                     |
| `global.greenhouse.clusterNames`     | List of all cluster names  | `["cluster-a", "cluster-b"]` |
| `global.greenhouse.teamNames`        | List of all team names     | `["team-1", "team-2"]`       |
| `global.greenhouse.baseDomain`       | Base DNS domain            | `greenhouse.example.com`     |
| `global.greenhouse.metadata.*`       | Cluster metadata labels    | `eu-de-1`                    |

> :information_source: `global.greenhouse.metadata.*` values are derived from cluster labels prefixed with `metadata.greenhouse.sap/`. For example, the label `metadata.greenhouse.sap/region: eu-de-1` becomes available as `global.greenhouse.metadata.region`.

### Examples

**Hostname per cluster:**

```yaml
- name: ingress.host
  expression: |
    "service.${global.greenhouse.clusterName}.example.com"
# Result for cluster "cluster-a": "service.cluster-a.example.com"
```

**Using cluster metadata:**

```yaml
- name: ingress.host
  expression: |
    "service.${global.greenhouse.metadata.region}.example.com"
# Result: "service.eu-de-1.example.com"
# Requires label metadata.greenhouse.sap/region on the cluster
```

**Combining variables:**

```yaml
- name: app.fqdn
  expression: |
    "${global.greenhouse.clusterName}-${global.greenhouse.organizationName}"
# Result for cluster "cluster-a" in org "my-org": "cluster-a-my-org"
```

### Expressions in ClusterOptionOverrides

Expressions can also be used in `clusterOptionOverrides`. Overrides are merged before expression evaluation, so override expressions are also resolved:

```yaml
spec:
  plugin:
    optionValues:
    - name: app.mode
      value: "standard"
  clusterOptionOverrides:
    - clusterName: special-cluster
      overrides:
      - name: app.hostname
        expression: |
          "special.${global.greenhouse.metadata.region}.example.com"
```

> :information_source: Expressions are only evaluated in PluginPresets.

> :warning: CEL expressions on standalone Plugins are deprecated and will be removed in a future release. Use PluginPresets for expression evaluation.


## Feature Flag

CEL expression evaluation in PluginPresets is enabled only when the feature flag `pluginPreset.expressionEvaluationEnabled` is set to `true` in the Greenhouse feature flags ConfigMap.

 By default, this flag is `false` if it is unset or invalid, and expressions are passed through as literal values without evaluation.

```yaml
# greenhouse-feature-flags ConfigMap
apiVersion: v1
kind: ConfigMap
metadata:
  name: greenhouse-feature-flags
  namespace: greenhouse
data:
  pluginPreset: |
    expressionEvaluationEnabled: true
```

## ValueFrom References Between PluginPresets

PluginPresets can reference option values from other PluginPresets using `valueFrom.ref`.
This enables one PluginPreset to use resolved values from another, including values generated by CEL expressions.

### Reference by Name

```yaml
# Source PluginPreset - generates a hostname per cluster
apiVersion: greenhouse.sap/v1alpha1
kind: PluginPreset
metadata:
  name: backend-preset
spec:
  plugin:
    pluginDefinitionRef:
      name: backend-service
    optionValues:
      - name: backend.hostname
        expression: |
          "backend.${global.greenhouse.clusterName}.example.com"
  clusterSelector:
    matchLabels:
      env: production
---
# Consumer PluginPreset - references the backend hostname
apiVersion: greenhouse.sap/v1alpha1
kind: PluginPreset
metadata:
  name: frontend-preset
spec:
  plugin:
    pluginDefinitionRef:
      name: frontend-service
    optionValues:
      - name: frontend.backendUrl
        valueFrom:
          ref:
            kind: PluginPreset
            name: backend-preset
            expression: |
              ${spec.optionValues.filter(v, v.name == "backend.hostname")[0].value}
  clusterSelector:
    matchLabels:
      env: production
```

The resulting Plugin will contain the resolved value. For example:

```yaml
# If the matching cluster is named "production-eu":
spec:
  optionValues:
  - name: frontend.backendUrl
    value: "backend.production-eu.example.com"
```

### Reference by Label Selector
When multiple PluginPresets need to be referenced, use a label selector.
The CEL expression is evaluated against each matching PluginPreset and results are collected into an array.


```yaml
# Multiple source PluginPresets with shared label
apiVersion: greenhouse.sap/v1alpha1
kind: PluginPreset
metadata:
  name: selector-source-a
  namespace: demo
  labels:
    e2e.greenhouse.sap/selector-test: "true"
spec:
  plugin:
    pluginDefinitionRef:
      name: perses
    releaseName: perses-sel-source-a
    releaseNamespace: kube-monitoring
    optionValues:
      - name: source.endpoint
        expression: |
          "endpoint-a.${global.greenhouse.clusterName}.example.com"
  clusterSelector:
    matchLabels:
      greenhouse.sap/cluster: kind-greenhouse-remote
---
apiVersion: greenhouse.sap/v1alpha1
kind: PluginPreset
metadata:
  name: selector-source-b
  namespace: demo
  labels:
    e2e.greenhouse.sap/selector-test: "true"
spec:
  plugin:
    pluginDefinitionRef:
      name: perses
    releaseName: perses-sel-source-b
    releaseNamespace: kube-monitoring
    optionValues:
      - name: source.endpoint
        expression: |
          "endpoint-b.${global.greenhouse.clusterName}.example.com"
  clusterSelector:
    matchLabels:
      greenhouse.sap/cluster: kind-greenhouse-remote
---
apiVersion: greenhouse.sap/v1alpha1
kind: PluginPreset
metadata:
  name: selector-consumer
  namespace: demo
spec:
  plugin:
    pluginDefinitionRef:
      name: perses
    releaseName: perses-sel-consumer
    releaseNamespace: kube-monitoring
    optionValues:
      - name: consumer.endpoints
        valueFrom:
          ref:
            kind: PluginPreset
            selector:
              matchLabels:
                e2e.greenhouse.sap/selector-test: "true"
            expression: |
              ${spec.optionValues.filter(v, v.name == "source.endpoint")[0].value}
  clusterSelector:
    matchLabels:
      greenhouse.sap/cluster: kind-greenhouse-remote
```

The consumer Plugin will receive an array of all resolved values:

```yaml
spec:
  optionValues:
  - name: consumer.endpoints
    value: [ "endpoint-a.kind-greenhouse-remote.example.com",
      "endpoint-b.kind-greenhouse-remote.example.com"]
```

### CEL Expression Syntax for References
The expression field in valueFrom.ref supports multiple syntax styles:

#### New simplified syntax
```expression: spec.optionValues.filter(v, v.name == "my.value")[0].value```

#### With ${...} wrapper
```expression: ${spec.optionValues.filter(v, v.name == "my.value")[0].value}```

#### Legacy syntax (backward compatible)
```expression: object.spec.optionValues.filter(v, v.name == "my.value")[0].value```


> :warning: ValueFrom references in PluginPresets only support referencing other PluginPresets (kind: PluginPreset). Referencing standalone Plugins is not supported.

### Feature Flags
Expression evaluation and ValueFrom.Ref resolution in PluginPresets are controlled by feature flags:

```yaml
# greenhouse-feature-flags ConfigMap
pluginPreset: |
  expressionEvaluationEnabled: true
  integrationEnabled: true
```

## Next Steps

- [Managing Plugins for multiple clusters](./../../../user-guides/plugin/plugin-management)
- [Plugin reference](./../plugin)
- [PluginDefinition reference](./../plugindefinition)