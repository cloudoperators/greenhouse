---
title: "Using Metadata Labels and CEL Expressions"
linkTitle: "Metadata and Expressions"
weight: 5
description: >
  Use cluster metadata labels and CEL expressions to create dynamic Plugin configurations.
---

## Overview

Greenhouse allows you to define metadata labels on Clusters and use them in Plugin configurations through CEL (Common Expression Language) expressions. This enables dynamic configuration of Plugins based on cluster-specific attributes like region, environment, or any custom metadata.

For information on setting metadata labels on Clusters, see [Setting Metadata Labels](./../../../reference/components/cluster#setting-metadata-labels).

## Using CEL Expressions in Plugins

Once metadata labels are set on a Cluster, you can reference them in Plugin optionValues using the `expression` field.

### Basic String Interpolation

Use `${...}` placeholders to insert metadata values into strings:

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: Plugin
metadata:
  name: example-plugin
  namespace: example-organization
spec:
  clusterName: example-cluster
  pluginDefinitionRef:
    kind: PluginDefinition
    name: example-app
  optionValues:
    - name: endpoint
      expression: "https://api.${global.greenhouse.metadata.region}.example.com"
    - name: username
      expression: "service-${global.greenhouse.metadata.environment}-user"
```

With cluster metadata labels `metadata.greenhouse.sap/region: europe` and `metadata.greenhouse.sap/environment: production`, this resolves to:
- `endpoint`: `"https://api.europe.example.com"`
- `username`: `"service-production-user"`

### Complex YAML Values

Expressions can produce complex YAML structures:

```yaml
optionValues:
  - name: config
    expression: |
      cluster: ${global.greenhouse.clusterName}
      region: ${global.greenhouse.metadata.region}
      environment: ${global.greenhouse.metadata.environment}
      endpoints:
        api: https://api.${global.greenhouse.metadata.region}.example.com
        metrics: https://metrics.${global.greenhouse.metadata.region}.example.com
```

### Using CEL Functions

CEL string functions can transform values:

```yaml
optionValues:
  # Convert to uppercase
  - name: clusterLabel
    expression: ${global.greenhouse.clusterName.upperAscii()}

  # Split and rejoin with different delimiter
  - name: normalizedName
    expression: ${global.greenhouse.clusterName.split('-').join('_')}

  # Check if environment contains a substring
  - name: isProduction
    expression: ${global.greenhouse.metadata.environment.contains('prod')}
```

For a complete list of available CEL string functions, see the [CEL string extension documentation](https://github.com/google/cel-go/tree/master/ext#strings).

## Using Expressions with PluginPresets

Expressions are particularly powerful with PluginPresets, allowing you to deploy Plugins to multiple clusters with cluster-specific configurations:

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: PluginPreset
metadata:
  name: monitoring-preset
  namespace: example-organization
spec:
  clusterSelector:
    matchLabels:
      metadata.greenhouse.sap/tier: premium
  plugin:
    pluginDefinitionRef:
      kind: PluginDefinition
      name: kube-monitoring
    releaseNamespace: monitoring
    optionValues:
      - name: prometheus.remoteWrite.url
        expression: "https://thanos.${global.greenhouse.metadata.region}.example.com/api/v1/receive"
      - name: grafana.dashboards.annotations
        expression: |
          cluster: ${global.greenhouse.clusterName}
          region: ${global.greenhouse.metadata.region}
          environment: ${global.greenhouse.metadata.environment}
```

This PluginPreset will:
1. Select all clusters with the `metadata.greenhouse.sap/tier: premium` label
2. Create a Plugin for each matching cluster
3. Resolve expressions using each cluster's specific metadata

## Available Variables

The following `global.greenhouse.*` variables are available in expressions:

| Variable | Description |
|----------|-------------|
| `global.greenhouse.clusterName` | The name of the target cluster |
| `global.greenhouse.organizationName` | The name of the organization |
| `global.greenhouse.clusterNames` | Names of all clusters in the organization |
| `global.greenhouse.teamNames` | Names of all teams in the organization |
| `global.greenhouse.baseDomain` | DNS base domain for Greenhouse |
| `global.greenhouse.metadata.*` | Cluster metadata labels |

## Next Steps

- [Setting Metadata Labels](./../../../reference/components/cluster#setting-metadata-labels)
- [Plugin reference](./../../../reference/components/plugin)
- [PluginPreset reference](./../../../reference/components/pluginpreset)
- [Managing Plugins for multiple clusters](./../plugin-management)
