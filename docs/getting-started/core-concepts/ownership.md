---
title: "Ownership and Labels"
weight: 3
---

## What is Ownership with Greenhouse

Ownership in Greenhouse is the combination of two of the core features:
- User and Organization management via [Teams](teams.md)
- Resource deployment via [Plugins](plugins.md) to remote [Clusters](clusters.md)


Greenhouse provides a `1:1` relationship for
- `PluginPresets` & `Plugins`
- `Clusters`
- `TeamRoleBindings`

to a `Team`. We call this relationship `Ownership`.

## Why Ownership of Resources
By identifying an owner of a resource it is possible to route operational tasks on the resource to the owner.
Examples for these operational tasks might be:

- Alert routing on Prometheus metrics for resources deployed by a `Plugin`
- Lifecycly management of k8s `Clusters`
- Security posture and vulnerability patch management for running resources
- `Secret` rotation

## How is Ownership achieved
Greenhouse expects a `label` with the key `greenhouse.sap/owned-by` with a value matching an existing `Team` on 
- `PluginPresets`
- `Plugins`
- `Clusters`
- `TeamRoleBindings`
- `Secrets`



### Label Transport

#### On Greenhouse Cluster
The Greenhouse controller [transports labels from a source resource to a target resource](https://github.com/cloudoperators/greenhouse/blob/main/internal/lifecycle/propagation.go) on the Greenhouse cluster.
This is currently active for:
- `Secrets` that are used to bootsrap a `Cluster`
- `PluginPresets` creating `Plugins`

The transport works via an `metadat.annotation` on the source:
```yaml
metadata:
  ...
  labels:
    foo: bar
    qux: baz
    owned_by: foo-team
    ...
  annotations:
    greenhouse.sap/propagate-labels: "foo, owned_by"
  ...
```
which results in `metadata.labels` and a state `metadata.annotation` added in the target:
```yaml
metadata:
  annotations:
   ...
    greenhouse.sap/last-applied-propagator: '{"labelKeys":["foo","owned_by"]}'
  labels:
    foo: bar
    owned_by: foo-team
   ...
```

#### To resources on remote `Clusters`

Greenhouse will provide the automation to label all resources created by a `Plugin` on the remote `Cluster` in the future: 
https://github.com/cloudoperators/greenhouse-extensions/issues/704

Currently Greenhouse provides the `owned-by` label as a `OptionValue` [to be consumed by the underlying helm chart of the `Plugin`](./../../contribute/plugins.md#development).

### Enforcing Labels
Greenhouse denies creation of resources without `owned-by` label. The value needs to be a valid `Team`.





