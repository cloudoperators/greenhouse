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
| `global.greenhouse.baseDomain`       | Base DNS domain            | `greenhouse.example.com`     |
| `global.greenhouse.ownedBy`          | Owning `Team`              | `team-1`                     |
| `global.greenhouse.metadata.*`       | Cluster metadata labels    | `eu-de-1`                    |

> :information_source: `global.greenhouse.metadata.*` values are derived from cluster labels prefixed with `metadata.greenhouse.sap/`. For example, the label `metadata.greenhouse.sap/region: eu-de-1` becomes available as `global.greenhouse.metadata.region`.

> :warning: `global.greenhouse.clusterNames` and `global.greenhouse.teamNames` were removed. An expression still referencing either of them fails to evaluate and blocks the PluginPreset from rolling out its Plugins. See [Injected Greenhouse Values](./../plugin#injected-greenhouse-values).

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

> :information_source: Expressions are only evaluated in PluginPresets. They need the feature flag `pluginPreset.expressionEvaluationEnabled`, see [Feature Flags](#feature-flags).

## ValueFrom References

A PluginPreset option can take its value from other PluginPresets or Plugins in the same namespace with `valueFrom.ref`, including values generated by CEL expressions. To collect a value from every cluster another PluginPreset rolls out to, reference its Plugins, see [Reference a Plugin](#reference-a-plugin). [Connecting Plugins with References](./../../../user-guides/plugin/plugin-integration) shows a full example.

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
              ${spec.plugin.optionValues.filter(v, v.name == "backend.hostname")[0].value}
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
The CEL expression is evaluated against each matching PluginPreset and results are collected into an array, each value once.


```yaml
# Source PluginPreset - backend-b is the same, with backend-b in its name and hostname
apiVersion: greenhouse.sap/v1alpha1
kind: PluginPreset
metadata:
  name: backend-a
  labels:
    example.com/backend: "true"
spec:
  plugin:
    pluginDefinitionRef:
      name: backend-service
    optionValues:
      - name: backend.hostname
        expression: |
          "backend-a.${global.greenhouse.clusterName}.example.com"
  clusterSelector:
    matchLabels:
      env: production
---
# Consumer PluginPreset - collects the hostname of every labelled PluginPreset
apiVersion: greenhouse.sap/v1alpha1
kind: PluginPreset
metadata:
  name: gateway-preset
spec:
  plugin:
    pluginDefinitionRef:
      name: gateway-service
    optionValues:
      - name: gateway.backendUrls
        valueFrom:
          ref:
            kind: PluginPreset
            selector:
              matchLabels:
                example.com/backend: "true"
            expression: |
              ${spec.plugin.optionValues.filter(v, v.name == "backend.hostname")[0].value}
  clusterSelector:
    matchLabels:
      env: production
```

The consumer Plugin receives one entry per matching PluginPreset, ordered by name:

```yaml
# If the matching cluster is named "production-eu":
spec:
  optionValues:
  - name: gateway.backendUrls
    value:
      - "backend-a.production-eu.example.com"
      - "backend-b.production-eu.example.com"
```

A selector that matches nothing fails the resolution.

### Reference a Plugin

Set `ref.kind` to `Plugin` to read an existing Plugin instead of a PluginPreset. Name and label selector work the same way as above.

The two kinds give different results. A reference to a PluginPreset resolves the source for the consumer's own cluster, so it gives one value per PluginPreset. A reference to a Plugin reads the Plugin as it is. The Plugins created by a PluginPreset carry the label `greenhouse.sap/pluginpreset: <preset name>`, so a selector on that label collects one value per cluster the source preset rolls out to.

```yaml
# Consumer PluginPreset - collects one host per cluster of the source preset
apiVersion: greenhouse.sap/v1alpha1
kind: PluginPreset
metadata:
  name: thanos-central
spec:
  plugin:
    pluginDefinitionRef:
      name: thanos
    optionValues:
      - name: thanos.query.stores
        valueFrom:
          ref:
            kind: Plugin
            selector:
              matchLabels:
                greenhouse.sap/pluginpreset: thanos-regional
            expression: |
              ${spec.optionValues.filter(v, v.name == "thanos.query.ingress.grpc.hosts")[0].value[0].host + ":443"}
  clusterSelector:
    matchLabels:
      env: production
```

### What an expression can read

The expression runs against the referenced object as the API server stores it, so every field is addressed by the same path `kubectl get -o yaml` prints. A reference to a Plugin reads its option values under `spec.optionValues`, a reference to a PluginPreset under `spec.plugin.optionValues`, and both can read `metadata`, the rest of `spec` and `status`.

`status` is the only place a value can come from that the source does not know before it runs, the URLs of the services it exposes for example:

```yaml
valueFrom:
  ref:
    kind: Plugin
    selector:
      matchLabels:
        greenhouse.sap/pluginpreset: thanos-regional
    expression: |
      ${status.exposedServices.map(url, url).sort()}
```

`exposedServices` is a map and a map has no order, so the result is sorted. Without that the same expression comes back in a different order on every resolution, which writes the Plugin again every time and never settles. The same holds for any expression reading a map, `metadata.labels` included.

A status change does not bump a Plugin's generation, so the PluginPreset controller watches the status of every Plugin and re-resolves the consumers referencing it. The same does not hold for a reference to a PluginPreset, which re-resolves on a change to the referenced spec, not its status.

An expression reads plain values only. An option that takes its value from a secret or from another reference comes through without its `valueFrom`, so neither its value nor the name of the secret behind it is readable. The `kubectl.kubernetes.io/last-applied-configuration` annotation is dropped for the same reason, it holds a copy of the spec the object was applied with.

### Setting a value next to a reference

An option can set a `value` and a `valueFrom.ref` at the same time. The two are merged into one list, with the value the option sets itself first:

```yaml
      - name: thanos.query.stores
        value:
          - thanos-sidecar.monitoring:10901
        valueFrom:
          ref:
            kind: Plugin
            selector:
              matchLabels:
                greenhouse.sap/pluginpreset: thanos-regional
            expression: |
              ${spec.optionValues.filter(v, v.name == "thanos.query.ingress.grpc.hosts")[0].value[0].host + ":443"}
```

A single value counts as a list with one entry, so the result is always a list. An `expression` on the same option is resolved first and merged the same way, which is how an entry built from the cluster ends up next to the entries read from other Plugins. An entry both the value and the reference hold is kept once, where it first shows up.

### CEL Expression Syntax for References

The expression can be written bare or wrapped in `${...}`, both mean the same:

- `expression: spec.optionValues.filter(v, v.name == "my.value")[0].value`
- `expression: ${spec.optionValues.filter(v, v.name == "my.value")[0].value}`

The wrapper goes around the whole expression, once. An expression holding more than one `${...}` is rejected. Besides `spec`, `metadata` and `status` the whole object is also available as `object`, so `object.spec.optionValues` works too.

> :warning: `ref.kind` accepts `PluginPreset` and `Plugin` only, and defaults to `PluginPreset` when left empty. Any other kind fails the resolution.

### When a reference fails

If a reference can't be resolved for a cluster, the Plugin for that cluster is left as it is. An existing Plugin keeps its last resolved values, a new one is not created, and the other clusters are not affected. The PluginPreset is not Ready until the reference resolves, and its `PluginFailed` condition holds the error for every affected Plugin.

## Feature Flags

Expressions and references in PluginPresets are each behind a feature flag in the `greenhouse-feature-flags` ConfigMap. Both are `false` when unset or invalid.

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
    integrationEnabled: true
```

`expressionEvaluationEnabled` turns on `expression`, `integrationEnabled` turns on `valueFrom.ref`. While a flag is off, a PluginPreset using that field fails to resolve and reports the error in its status.

> :warning: Both are experimental. They cover the scenarios written up here and not much beyond them, and the shape of an expression can still change. Turn them on for a cluster you are willing to debug.

## Next Steps

- [Managing Plugins for multiple clusters](./../../../user-guides/plugin/plugin-management)
- [Connecting Plugins with References](./../../../user-guides/plugin/plugin-integration)
- [Plugin reference](./../plugin)
- [PluginDefinition reference](./../plugindefinition)
