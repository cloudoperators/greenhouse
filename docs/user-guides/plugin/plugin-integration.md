---
title: "Connecting Plugins with References"
linkTitle: "Plugin Integration"
weight: 6
description: >
  Use values of other Plugins in a PluginPreset.
---

## Overview

A central Thanos Query has to know the endpoint of every regional Thanos. Written into its PluginPreset by hand, that list needs an edit for every new region.

Instead, a PluginPreset option can take its value from other Plugins in the same namespace with `valueFrom.ref`. The reference names a Plugin or selects several by label, and a CEL expression picks the value out of each of them. Greenhouse resolves the reference for every cluster the PluginPreset deploys to, writes the result into the generated Plugin as a plain value and resolves it again whenever a referenced Plugin changes.

This guide sets up the Thanos example. All fields are described in [ValueFrom References](./../../../reference/components/pluginpreset#valuefrom-references).

## A regional Thanos on every cluster

The `thanos-regional` PluginPreset deploys Thanos to every cluster labelled `cluster-type: observability` and builds the host of its gRPC ingress from the cluster label `metadata.greenhouse.sap/region`:

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
          - host: thanos-grpc.${global.greenhouse.metadata.region}.example.com
            paths:
              - path: /
                pathType: ImplementationSpecific
```

Greenhouse creates one Plugin per matching cluster, for example `thanos-regional-obs-eu-de-1`, with the expression already resolved to the host. Every Plugin created by a PluginPreset carries the label `greenhouse.sap/pluginpreset: <PluginPreset name>`, the next section selects on it.

## A central Thanos Query for all regions

The `thanos-global` PluginPreset deploys a central Thanos Query to the cluster `obs-eu-de-1`. Its `thanos.query.stores` holds a fixed Thanos sidecar and the gRPC host of every regional Thanos:

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

The selector matches every Plugin created by `thanos-regional`. The expression runs once per Plugin and the results are collected into one list, ordered by Plugin name. The option's own `value` goes first, and an entry that shows up twice is kept once. The generated Plugin only holds the result:

```yaml
spec:
  optionValues:
    - name: thanos.query.stores
      value:
        - thanos-sidecar.monitoring:10901
        - thanos-grpc.eu-de-1.example.com:443
        - thanos-grpc.eu-nl-1.example.com:443
```

The reference uses `kind: Plugin`. With `kind: PluginPreset` and `name: thanos-regional`, Greenhouse would resolve `thanos-regional` for the cluster of `thanos-global` only and return a single host.

## Adding a region

To add a region, label its cluster:

```bash
kubectl label cluster obs-ap-jp-1 \
  cluster-type=observability \
  metadata.greenhouse.sap/region=ap-jp-1 \
  --namespace=example-organization
```

Nothing else has to change. `thanos-regional` deploys Thanos to the new cluster, and as soon as its Plugin exists Greenhouse resolves `thanos-global` again and adds `thanos-grpc.ap-jp-1.example.com:443` to `thanos.query.stores`. When a regional Plugin is deleted, for example because its cluster no longer matches, its entry goes away.

## Reading the status

The gRPC host is set before Thanos is deployed, so the expression reads it from `spec`. Values that only exist once a Plugin is deployed are in its `status`. `status.exposedServices` maps the URL of every Service and Ingress Greenhouse exposes for the Plugin to its details, see [URLs for exposed services and ingresses](./../plugin-deployment#urls-for-exposed-services-and-ingresses). This reference collects the URLs of every regional Thanos:

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

`exposedServices` is a map and a map has no order. Without `.sort()` the URLs come back in a different order on every resolution, so the generated Plugin is written again each time and never settles.

Greenhouse also resolves the reference again when the status of a referenced Plugin changes.

## A different Plugin on each cluster

`${global.greenhouse.clusterName}` is resolved in an `expression`, but not in `ref.name` or `ref.selector`. To read a different Plugin on each cluster, set the reference per cluster in `clusterOptionOverrides`. For example, to give a Plugin the gRPC host of the Thanos in its own region:

```yaml
spec:
  clusterOptionOverrides:
    - clusterName: obs-eu-de-1
      overrides:
        - name: app.thanosHost
          valueFrom:
            ref:
              kind: Plugin
              name: thanos-regional-obs-eu-de-1
              expression: |
                ${spec.optionValues.filter(v, v.name == "thanos.query.ingress.grpc.hosts")[0].value[0].host}
```

Every cluster needs its own override, so a new region is not picked up on its own here.

## Troubleshooting

If a reference can't be resolved for a cluster, Greenhouse leaves the Plugin for that cluster as it is. An existing Plugin keeps its last resolved values, a new one is not created, and the other clusters are not affected. The PluginPreset is not Ready until the reference resolves, and the `PluginFailed` condition in its `statusConditions` holds the error for every affected Plugin:

```bash
kubectl describe pluginpreset thanos-global --namespace=example-organization
```

| Error | Cause |
|-------|-------|
| `failed to get Plugin <name>: ... not found` | The referenced Plugin does not exist (yet). |
| `no Plugins found matching selector ...` | The selector matches no Plugin in the namespace. |
| `no such key: <field>` | The expression reads a field the referenced object does not have. If the message lists `options with no plain value to read`, the option takes its value from a Secret or another reference, which an expression can't read. |
| `option <name> has valueFrom.ref but integrationEnabled is disabled ...` | References are not enabled in this Greenhouse installation. |

## Next Steps

- [PluginPreset reference](./../../../reference/components/pluginpreset)
- [Using Metadata Labels and CEL Expressions](./../metadata-expressions)
- [Managing Plugins for multiple clusters](./../plugin-management)
