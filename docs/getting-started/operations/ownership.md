---
title: "Ownership"
weight: 3
---

## What is Ownership within Greenhouse

`Ownership` in Greenhouse is the combination of two of the core features:

- User and Organization management via [Teams](./../core-concepts/teams.md)
- Deployment of resources ([Plugins](./../core-concepts/plugins.md), [TeamRoleBindings](./../core-concepts/teams/#team-rbac)) to remote [Clusters](./../core-concepts/clusters.md)

Greenhouse provides a `1:1` relationship between a Team and

- PluginPresets
- Plugins
- Clusters
- TeamRoleBindings
- Secrets

Within the context of Greenhouse this relationship is called `Ownership`.

## Why Ownership of Resources

[Operational processes](processes.md) facilitated via Greenhouse rely on `Ownership`:

By identifying the owner of a resource it is possible to route operational tasks on the resource to the owner.

## How is Ownership achieved

Greenhouse expects a `label` with the key `greenhouse.sap/owned-by` with a value matching an existing Team on the following resources in the Greenhouse central cluster:

- PluginPresets
- Plugins
- Clusters
- TeamRoleBindings
- Secrets

> Missing `greenhouse.sap/owned-by` label results in a `StatusCondition` called `OwnerLabelSetCondition` set to `false`. A `greenhouse_owned_by_label_missing` metric on missing owner labels is exposed and alerted on.

The owner label is also expected on k8s resources (e.g. Deployments, Pods, ...) exposing metrics on the remote clusters.

### Label Transport

#### On Greenhouse central cluster

The Greenhouse controller [transports labels from a source resource to a target resource](https://github.com/cloudoperators/greenhouse/blob/main/internal/lifecycle/propagation.go) on the Greenhouse cluster.
This is currently active for:

- Secrets that are used to bootstrap a Cluster
- PluginPresets creating Plugins

The transport works via an `metadata.annotation` on the source:

```yaml
metadata:
  ...
  labels:
    foo: bar
    qux: baz
    greenhouse.sap/owned_by: foo-team
    ...
  annotations:
    greenhouse.sap/propagate-labels: "foo, greenhouse.sap/owned_by"
  ...
```

which results in `metadata.labels` and a state in `metadata.annotations` added to the target:

```yaml
metadata:
  annotations:
   ...
    greenhouse.sap/last-applied-propagator: '{"labelKeys":["foo","greenhouse.sap/owned_by"]}'
  labels:
    foo: bar
    greenhouse.sap/owned_by: foo-team
   ...
```

#### On Resources on Remote Clusters

Greenhouse will provide the automation to label all resources created by a Plugin on the remote Cluster in the future:
<https://github.com/cloudoperators/greenhouse-extensions/issues/704>

Currently Greenhouse provides the `owned-by` label as a `OptionValue` [to be consumed by the underlying helm chart of the Plugin](./../../contribute/plugins.md#development).
