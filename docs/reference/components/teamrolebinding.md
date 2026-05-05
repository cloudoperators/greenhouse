---
title: "TeamRoleBinding"
linkTitle: "TeamRoleBinding"
landingSectionIndex: false
description: >
  Reference documentation for the TeamRoleBinding resource.
---

A `TeamRoleBinding` assigns a `TeamRole` to one or more Teams (and/or individual usernames) for a set of Clusters and optionally a set of Namespaces. The Greenhouse controller translates a TeamRoleBinding into native Kubernetes `rbacv1` resources on each targeted remote cluster.

## Example TeamRoleBinding Spec

```yaml
apiVersion: greenhouse.sap/v1alpha2
kind: TeamRoleBinding
metadata:
  name: example-team-pod-read
  namespace: example-organization
spec:
  teamRoleRef: pod-read
  teamRefs:
    - example-team
    - dev-team
  clusterSelector:
    labelSelector:
      matchLabels:
        environment: production
  namespaces:
    - monitoring
    - logging
  createNamespaces: false
```

## Writing a TeamRoleBinding Spec

### `.spec.teamRoleRef`

**Required.** The name of the `TeamRole` resource (in the same namespace) that defines the set of RBAC permissions to grant.

```yaml
spec:
  teamRoleRef: pod-read
```

Changing `teamRoleRef` after creation requires deleting and re-creating the TeamRoleBinding (the field is effectively immutable for the underlying `rbacv1` binding objects).

### `.spec.teamRefs`

**Recommended.** A list of `Team` resource names (in the same namespace) to grant access. Each team's `spec.mappedIdPGroup` is used as the `Group` subject in the generated `rbacv1.ClusterRoleBinding` or `rbacv1.RoleBinding` on the remote cluster.

```yaml
spec:
  teamRefs:
    - example-team
    - dev-team
```

The subjects list on the remote cluster is the union of the IDP groups from all referenced teams plus any entries in `spec.usernames`.

**Partial failure**: if one or more teams do not exist, Greenhouse still creates RBAC for the teams that do exist. The `RBACReady` status condition will contain the names of any missing teams. Only if _all_ referenced teams are missing will RBAC not be applied at all and `RBACReady` will be set to `False` with reason `TeamNotFound`.

### `.spec.teamRef` (deprecated)

> [!WARNING]
> `spec.teamRef` is **deprecated**. Use `spec.teamRefs` instead.

The singular `teamRef` field accepts a single team name for backwards compatibility. The mutating webhook automatically merges `teamRef` into `teamRefs` on every create or update, so existing resources are migrated lazily without any manual intervention required.

If both `teamRef` and `teamRefs` are present, `teamRef` is appended to `teamRefs` (deduplicated) and then `teamRef` is cleared.

### `.spec.usernames`

An optional list of individual Kubernetes usernames to add as `User` subjects alongside the Team IDP groups.

```yaml
spec:
  usernames:
    - jane@example.com
    - bot-ci-user
```

### `.spec.clusterSelector`

Specifies which Clusters to target. Accepts either a direct cluster name or a label selector.

**By name** (single cluster):

```yaml
spec:
  clusterSelector:
    clusterName: example-cluster
```

**By label** (one or more clusters):

```yaml
spec:
  clusterSelector:
    labelSelector:
      matchLabels:
        environment: production
```

When a cluster's labels change so that it no longer matches the selector, Greenhouse removes the RBAC resources from that cluster. When a new cluster gains matching labels, RBAC is automatically applied.

### `.spec.namespaces`

An optional list of Kubernetes namespace names on the remote cluster. When set, the controller creates a `rbacv1.RoleBinding` per namespace (namespace-scoped). When empty, a single `rbacv1.ClusterRoleBinding` is created (cluster-scoped).

```yaml
spec:
  namespaces:
    - monitoring
    - logging
```

> [!NOTE]
> The scope of a TeamRoleBinding (cluster-scoped vs. namespace-scoped) is determined at creation time and cannot be changed afterwards. A cluster-scoped binding cannot gain namespaces, and a namespace-scoped binding cannot be made cluster-scoped, without deleting and re-creating the resource.

### `.spec.createNamespaces`

When `true`, the controller creates any namespaces listed in `.spec.namespaces` that do not already exist on the remote cluster. Defaults to `false`. Deleting the TeamRoleBinding never deletes the created namespaces.

```yaml
spec:
  createNamespaces: true
```

## Status

### `.status.statusConditions`

Contains a `RBACReady` condition and an overall `Ready` condition.

| Condition | Status | Reason | Meaning |
| --- | --- | --- | --- |
| `RBACReady` | `True` | `RBACReconciled` | All RBAC resources applied successfully on all target clusters |
| `RBACReady` | `False` | `RBACReconcileFailed` | One or more clusters could not be reconciled |
| `RBACReady` | `False` | `TeamNotFound` | All referenced teams are missing; no RBAC was applied |
| `RBACReady` | `False` | `TeamRoleNotFound` | The referenced TeamRole does not exist |
| `RBACReady` | `False` | `EmptyClusterList` | The cluster selector matched no clusters |
| `RBACReady` | `False` | `ClusterConnectionFailed` | Could not connect to a target cluster |

### `.status.clusters`

A per-cluster propagation status list. Each entry records the cluster name and the `RBACReady` condition for that specific cluster, enabling operators to see at a glance which clusters succeeded or failed.

```yaml
status:
  clusters:
    - clusterName: cluster-a
      condition:
        type: RBACReady
        status: "True"
        reason: RBACReconciled
    - clusterName: cluster-b
      condition:
        type: RBACReady
        status: "False"
        reason: RBACReconcileFailed
        message: "Failed to reconcile RoleBindings: ..."
```

## What gets applied to the remote cluster

| Condition | Remote resource created |
| --- | --- |
| `spec.namespaces` is empty | One `rbacv1.ClusterRole` + one `rbacv1.ClusterRoleBinding` named `greenhouse:<TRB-name>` |
| `spec.namespaces` is set | One `rbacv1.ClusterRole` + one `rbacv1.RoleBinding` per namespace, each named `greenhouse:<TRB-name>` |

The `ClusterRole` is created from the `TeamRole`'s `spec.rules` (and `spec.aggregationRule` if present). The subjects of the binding are the union of the `MappedIDPGroup` values of all referenced Teams plus any `spec.usernames`.

## Next Steps

- [Team RBAC user guide](./../../../user-guides/team/rbac)
