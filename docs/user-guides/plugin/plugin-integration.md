---
title: "Connecting Plugins with References"
linkTitle: "Plugin Integration"
weight: 6
description: >
  Use values of other Plugins in a PluginPreset.
---

## Overview

This guide describes how to use values of other Plugins in a PluginPreset.

An option of a PluginPreset can take its value from another Plugin or PluginPreset in the same namespace with `valueFrom.ref`. A CEL expression selects the value. Greenhouse resolves the reference for each cluster and writes the result into the generated Plugin as a plain value. When the spec or labels of the referenced Plugin or PluginPreset change, or the status of a referenced Plugin, the value is resolved again.

The guide covers the following cases:

| Case | Description | Example |
|------|-------------|---------|
| [n:n](#nn-integration) | Each Plugin reads a value from the other Plugin on the same cluster. | Perses on every cluster gets the ServiceMonitor labels of the Prometheus on the same cluster. |
| [1:n](#1n-integration) | One Plugin collects a value from multiple Plugins. | A global Thanos Query discovers the Thanos endpoints of all clusters. |
| [n:1](#n1-integration) | Multiple Plugins read a value from the same Plugin. | The Prometheus on every cluster sends its alerts to one central Alertmanager. |
| [m:n](#mn-integration) | Multiple Plugins each collect a value from a different group of Plugins. | A Thanos Query per domain, such as compute or storage, discovers the Thanos endpoints of the clusters in its domain. |

The examples use the `kube-monitoring`, `perses`, `thanos` and `alerts` Plugins. They are only examples, references work the same way between any other Plugins.

See [ValueFrom References](./../../../reference/components/pluginpreset#valuefrom-references) for more details on all fields.

## Prerequisites

The feature flags `pluginPreset.expressionEvaluationEnabled` and `pluginPreset.integrationEnabled` must be enabled in your Greenhouse installation. Both are off by default, see [Feature Flags](./../../../reference/components/pluginpreset#feature-flags).

## n:n integration

Each Plugin reads a value from the other Plugin on the same cluster. Both are created from PluginPresets deployed to the same clusters, and the reference points to the other PluginPreset.

For example, Perses and Prometheus are deployed to the same clusters. The Prometheus only scrapes ServiceMonitors with certain labels, so the ServiceMonitor of Perses needs these labels.

The `kube-monitoring` PluginPreset sets the labels the Prometheus selects ServiceMonitors by:

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: PluginPreset
metadata:
  name: kube-monitoring
  namespace: example-organization
spec:
  clusterSelector:
    matchLabels:
      cluster-type: observability
  plugin:
    pluginDefinitionRef:
      kind: ClusterPluginDefinition
      name: kube-monitoring
    releaseName: kube-monitoring
    releaseNamespace: kube-monitoring
    optionValues:
      - name: kubeMonitoring.prometheus.prometheusSpec.serviceMonitorSelector
        value:
          matchLabels:
            plugin: kube-monitoring
```

The `perses` PluginPreset reads these labels from the `kube-monitoring` PluginPreset:

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: PluginPreset
metadata:
  name: perses
  namespace: example-organization
spec:
  clusterSelector:
    matchLabels:
      cluster-type: observability
  plugin:
    pluginDefinitionRef:
      kind: ClusterPluginDefinition
      name: perses
    releaseName: perses
    releaseNamespace: kube-monitoring
    optionValues:
      - name: perses.serviceMonitor.selfMonitor
        value: true
      - name: perses.serviceMonitor.labels
        valueFrom:
          ref:
            kind: PluginPreset
            name: kube-monitoring
            expression: |
              ${spec.plugin.optionValues.filter(v, v.name == "kubeMonitoring.prometheus.prometheusSpec.serviceMonitorSelector")[0].value.matchLabels}
```

The Plugin created for each cluster contains the resolved value:

```yaml
spec:
  optionValues:
    - name: perses.serviceMonitor.selfMonitor
      value: true
    - name: perses.serviceMonitor.labels
      value:
        plugin: kube-monitoring
```

A reference to a PluginPreset is resolved for each cluster separately. If `kube-monitoring` sets different labels for a cluster in its `clusterOptionOverrides`, the Perses on that cluster gets these labels.

### Referencing a Plugin instead of a PluginPreset

A reference to a PluginPreset can only read what is set in the PluginPreset. To read from the Plugin on a cluster, for example its `status`, use `kind: Plugin`. The name of a Plugin is `<PluginPreset name>-<cluster name>`, so the reference has to be set for each cluster in `clusterOptionOverrides`:

```yaml
spec:
  clusterOptionOverrides:
    - clusterName: obs-eu-de-1
      overrides:
        - name: perses.serviceMonitor.labels
          valueFrom:
            ref:
              kind: Plugin
              name: kube-monitoring-obs-eu-de-1
              expression: |
                ${spec.optionValues.filter(v, v.name == "kubeMonitoring.prometheus.prometheusSpec.serviceMonitorSelector")[0].value.matchLabels}
```

See [What a reference returns](./../../../reference/components/pluginpreset#what-a-reference-returns) for the difference between the two kinds.

## 1:n integration

A Plugin collects a value from multiple Plugins. The Plugins are selected by their labels, and the results are collected into a list.

For example, a central Thanos Query needs the gRPC endpoint of the Thanos on every cluster.

The `thanos-regional` PluginPreset deploys Thanos to all clusters with the label `cluster-type: observability` and exposes its Thanos Query via a gRPC ingress. The host is built from the cluster name:

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: PluginPreset
metadata:
  name: thanos-regional
  namespace: example-organization
spec:
  clusterSelector:
    matchLabels:
      cluster-type: observability
  plugin:
    pluginDefinitionRef:
      kind: ClusterPluginDefinition
      name: thanos
    releaseName: thanos
    releaseNamespace: kube-monitoring
    optionValues:
      - name: thanos.query.ingress.grpc.enabled
        value: true
      - name: thanos.query.ingress.grpc.hosts
        expression: |
          - host: thanos-grpc.${global.greenhouse.clusterName}.example.com
            paths:
              - path: /
                pathType: ImplementationSpecific
```

The `thanos-global` PluginPreset deploys the central Thanos Query to the cluster `obs-eu-de-1`. It selects all Plugins created by `thanos-regional` via the label `greenhouse.sap/pluginpreset` and reads the host from each of them:

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: PluginPreset
metadata:
  name: thanos-global
  namespace: example-organization
spec:
  clusterSelector:
    matchLabels:
      greenhouse.sap/cluster: obs-eu-de-1
  plugin:
    pluginDefinitionRef:
      kind: ClusterPluginDefinition
      name: thanos
    releaseName: thanos-global
    releaseNamespace: kube-monitoring
    optionValues:
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

The expression is evaluated for each matching Plugin. The `value` of the option is added in front of the list. The created Plugin contains:

```yaml
spec:
  optionValues:
    - name: thanos.query.stores
      value:
        - thanos-sidecar.monitoring:10901
        - thanos-grpc.obs-eu-de-1.example.com:443
        - thanos-grpc.obs-eu-nl-1.example.com:443
```

When another cluster is labelled with `cluster-type: observability`, `thanos-regional` creates a Plugin for it and Greenhouse adds its host to the `thanos.query.stores` of the central Thanos Query. When a Plugin is deleted, its host is removed again.

> :information_source: The reference has to use `kind: Plugin` here. A reference to the PluginPreset `thanos-regional` is resolved for the cluster of the central Thanos Query only and returns a single host.

## n:1 integration

Multiple Plugins read a value from the same Plugin. This is a reference to a Plugin by name in a PluginPreset that is deployed to more than one cluster. Every created Plugin gets the same value.

For example, the Prometheus on every cluster sends its alerts to one central Alertmanager. The Alertmanager is deployed by the Plugin `alerts`, which sets its hostnames in the option `alerts.alertmanager.ingress.hosts`. The `kube-monitoring` PluginPreset reads them from this Plugin:

```yaml
spec:
  plugin:
    optionValues:
      - name: alerts.enabled
        value: true
      - name: alerts.alertmanagers.hosts
        valueFrom:
          ref:
            kind: Plugin
            name: alerts
            expression: |
              ${spec.optionValues.filter(v, v.name == "alerts.alertmanager.ingress.hosts")[0].value}
```

## m:n integration

Multiple Plugins each collect a value from a different group of Plugins. This is a PluginPreset deployed to more than one cluster, with a reference with a selector set for each cluster in `clusterOptionOverrides`.

For example, the clusters are split into domains, such as compute and storage. Each domain has its own Thanos Query that needs the gRPC endpoints of the Thanos in its domain only.

The Thanos in each domain is deployed by its own PluginPreset, `thanos-compute` for the clusters with the label `domain: compute` and `thanos-storage` for the clusters with the label `domain: storage`. Both are set up like `thanos-regional` in the 1:n integration.

The `thanos-domain` PluginPreset deploys a Thanos Query to the clusters `obs-compute` and `obs-storage`. Each of them selects the Plugins of its domain:

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: PluginPreset
metadata:
  name: thanos-domain
  namespace: example-organization
spec:
  clusterSelector:
    matchLabels:
      thanos-domain: "true"
  plugin:
    pluginDefinitionRef:
      kind: ClusterPluginDefinition
      name: thanos
    releaseName: thanos-domain
    releaseNamespace: kube-monitoring
  clusterOptionOverrides:
    - clusterName: obs-compute
      overrides:
        - name: thanos.query.stores
          valueFrom:
            ref:
              kind: Plugin
              selector:
                matchLabels:
                  greenhouse.sap/pluginpreset: thanos-compute
              expression: |
                ${spec.optionValues.filter(v, v.name == "thanos.query.ingress.grpc.hosts")[0].value[0].host + ":443"}
    - clusterName: obs-storage
      overrides:
        - name: thanos.query.stores
          valueFrom:
            ref:
              kind: Plugin
              selector:
                matchLabels:
                  greenhouse.sap/pluginpreset: thanos-storage
              expression: |
                ${spec.optionValues.filter(v, v.name == "thanos.query.ingress.grpc.hosts")[0].value[0].host + ":443"}
```

The Plugin on `obs-compute` gets the hosts of all compute clusters, the Plugin on `obs-storage` the hosts of all storage clusters. The selector has to be set for each cluster, as `ref.selector` is used as it is. See [What a reference returns](./../../../reference/components/pluginpreset#what-a-reference-returns) for more details.

## Reading the status of a Plugin

An expression can also read the `status` of a Plugin, for example the URLs of its exposed services. See [URLs for exposed services and ingresses](./../plugin-deployment#urls-for-exposed-services-and-ingresses) for more details on these URLs.

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

`exposedServices` is a map and has no defined order, so the result has to be sorted. See [What an expression can read](./../../../reference/components/pluginpreset#what-an-expression-can-read) for more details.

## Troubleshooting

If a reference can't be resolved for a cluster, the Plugin for this cluster is not created or updated and the PluginPreset is not Ready. The error is shown in the `PluginFailed` condition of the PluginPreset:

```bash
kubectl describe pluginpreset thanos-global --namespace=example-organization
```

| Error | Cause |
|-------|-------|
| `failed to get Plugin <name>: ... not found` or `failed to get PluginPreset <name>: ... not found` | The referenced Plugin or PluginPreset does not exist (yet). |
| `no Plugins found matching selector ...` or `no PluginPresets found matching selector ...` | The selector matches no Plugin or PluginPreset in the namespace. |
| `index out of bounds: 0` | The expression filters for an option that is not set on the referenced Plugin or PluginPreset. |
| `no such key: <field>` | The expression reads a field the referenced object does not have. If the message lists `options with no plain value to read`, the option takes its value from a Secret or another reference, which an expression can't read. |
| `option <name> has valueFrom.ref but integrationEnabled is disabled ...` | References are not enabled in this Greenhouse installation. |

See [When a reference fails](./../../../reference/components/pluginpreset#when-a-reference-fails) for more details.

## Next Steps

- [PluginPreset reference](./../../../reference/components/pluginpreset)
- [Using Metadata Labels and CEL Expressions](./../metadata-expressions)
- [Managing Plugins for multiple clusters](./../plugin-management)
