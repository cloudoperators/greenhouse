---
title: "Role-based access control on remote clusters"
description: >
  Creating and managing RBAC for Teams on Greenhouse-managed remote clusters.
---
## Greenhouse Team RBAC user guide

Role-Based Access Control (RBAC) in Greenhouse allows Organization administrators to manage the access of Teams on Clusters. TeamRole and TeamRoleBindings are used to manage the RBAC on remote Clusters. These two Custom Resource Definitions allow for fine-grained control over the permissions of each Team within each Cluster and Namespace.

## Contents

- [Before you begin](#before-you-begin)
- [Overview](#overview)
- [Defining TeamRoles](#defining-teamroles)
  - [Example](#example)
- [Seeded default TeamRoles](#seeded-default-teamroles)
- [Defining TeamRoleBindings](#defining-teamrolebindings)
  - [Assigning TeamRoles to Teams on a single Cluster](#assigning-teamroles-to-teams-on-a-single-cluster)
  - [Assigning TeamRoles to Teams on multiple Clusters](#assigning-teamroles-to-teams-on-multiple-clusters)
  - [Aggregating TeamRoles](#aggregating-teamroles)

## Before you begin

This guide is intended for users who want to manage Role-Based Access Control (RBAC) for Teams on remote clusters managed by Greenhouse. It assumes you have a basic understanding of [Kubernetes RBAC concepts](https://kubernetes.io/docs/reference/access-authn-authz/rbac/) and the Greenhouse platform.

🔑 Permissions

 1. Create/Update TeamRoles and TeamRoleBindings in the Organization namespace.
 2. View Teams and Clusters in the Organization namespace

By default the necessary authorizations are provieded via the `role:<organization>:admin` RoleBinding that is granted to members of the Organizations Admin Team.
You can check the permissions inside the Organization namespace by running the following command:

```bash
kubectl auth can-i --list --namespace=<organization-namespace>
```

💻 Software

1. [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl): The Kubernetes command-line tool which allows you to manage Kubernetes cluster resources.

## Overview

- **TeamRole**: Defines a set of permissions that can be assigned to teams and individual users.
- **TeamRoleBinding**: Assigns a TeamRole to a specific Team and/or a list of users fo Clusters and (optionally) Namespaces.

## Defining TeamRoles

TeamRoles define the actions a Team can perform on a Kubernetes cluster. For each Organziation a set of TeamRoles is [seeded](#seeded-default-teamroles). The syntax of the TeamRole's `.spec` is following the Kubernetes RBAC API.

### Example

This TeamRole named `pod-read` grants read access to Pods.

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: TeamRole
metadata:
  name: pod-read
spec:
  rules:
    - apiGroups:
        - ""
      resources:
        - "pods"
      verbs:
        - "get"
        - "list"
```

## Seeded default TeamRoles

Greenhouse provides a set of [default TeamRoles](./../../../pkg/controllers/organization/teamrole_seeder_controller.go) that are seeded to all clusters:

| TeamRole                | Description                                                                                                                         | APIGroups | Resources                                                                             | Verbs                                                         |
| ----------------------- | ----------------------------------------------------------------------------------------------------------------------------------- | --------- | ------------------------------------------------------------------------------------- | ------------------------------------------------------------- |
| `cluster-admin`         | Full privileges                                                                                                                     | \*        | \*                                                                                    | \*                                                            |
| `cluster-viewer`        | `get`, `list` and `watch` all resources                                                                                             | \*        | \*                                                                                    | `get`, `list`, `watch`                                        |
| `cluster-developer`     | Aggregated role. Greenhouse aggregates the `application-developer` and the `cluster-viewer`. Further TeamRoles can be aggregated. |           |                                                                                       |                                                               |
| `application-developer` | Set of permissions on `pods`, `deployments` and `statefulsets` necessary to develop applications on k8s                             | `apps`    | `deployments`, `statefulsets`                                                         | `patch`                                                       |
|                         |                                                                                                                                     | ""        | `pods`, `pods/portforward`, `pods/eviction`, `pods/proxy`, `pods/log`, `pods/status`, | `get`, `list`, `watch`, `create`, `update`, `patch`, `delete` |
| `node-maintainer`       | `get` and `patch` `nodes`                                                                                                           | ""        | `nodes`                                                                               | `get`, `patch`                                                |
| `namespace-creator`     | All permissions on `namespaces`                                                                                                     | ""        | `namespaces`                                                                          | \*                                                            |

## Defining TeamRoleBindings

TeamRoleBindings link a Team, a TeamRole, one or more Clusters and optionally one or more Namespaces together. Once the TeamRoleBinding is created, the Team will have the permissions defined in the TeamRole within the specified Clusters and Namespaces. This allows for fine-grained control over the permissions of each Team within each Cluster.
The TeamRoleBinding Controller within Greenhouse deploys RBAC resources to the targeted Clusters. The referenced TeamRole is created as a rbacv1.ClusterRole. In case the TeamRoleBinding references a Namespace, it is considered to be namespace-scoped. Hence, the controller will create a rbacv1.RoleBinding which links the Team with the rbacv1.ClusterRole. In case no Namespace is referenced, the Controller will create a cluster-scoped rbacv1.ClusterRoleBinding instead.

### Assigning TeamRoles to Teams on a single Cluster

Roles are assigned to Teams through the TeamRoleBinding configuration, which links Teams to their respective roles within specific clusters.

This TeamRoleBinding assigns the `pod-read` TeamRole to the Team named `my-team` in the Cluster named `my-cluster`.

Example: `team-rolebindings.yaml`

```yaml
apiVersion: greenhouse.sap/v1alpha2
kind: TeamRoleBinding
metadata:
  name: my-team-read-access
spec:
  teamRef: my-team
  roleRef: pod-read
  clusterSelector:
    clusterName: my-cluster
```

### Assigning TeamRoles to Teams on multiple Clusters

A LabelSelector can be used to assign a TeamRoleBinding to multiple Clusters.

This TeamRoleBinding assigns the `pod-read` TeamRole to the Team named `my-team` in all Clusters that have the label `environment: production` set.

```yaml
apiVersion: greenhouse.sap/v1alpha2
kind: TeamRoleBinding
metadata:
  name: production-cluster-admins
spec:
  teamRef: my-team
  roleRef: pod-read
  clusterSelector:
    labelSelector:
      matchLabels:
        environment: production
```

### Aggregating TeamRoles

It is possible with Kubernetes RBAC to aggregate rbacv1.ClusterRoles. This is also supported for TeamRoles. All label specified on a TeamRole's `.spec.Labels` will be set on the rbacv1.ClusterRole created on the target cluster. This makes it possible to aggregate multiple rbacv1.ClusterRole resources by using a rbacv1.AggregationRule. This can be specified on a TeamRole by setting `.spec.aggregationRule`.

More details on the concept of Aggregated ClusterRoles can be found in the Kubernetes documentation: [Aggregated ClusterRoles](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#aggregated-clusterroles)

[!NOTE] A TeamRole is only created on a cluster if it is referenced by a TeamRoleBinding. If a TeamRole is not referenced by a TeamRoleBinding it will not be created on any target cluster. A TeamRoleBinding referencing a TeamRole with an aggregationRule will only provide the correct access, if there is at least one TeamRoleBinding referencing a TeamRole with the corresponding label deployed to the same cluster.

The following example shows how an AggregationRule can be used with TeamRoles and TeamRoleBindings.

This TeamRole specifies `.spec.Labels`. The labels will be applied to the resulting ClusterRole on the target cluster.

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: TeamRole
metadata:
  name: pod-read
spec:
  labels:
    aggregate: "true"
  rules:
    - apiGroups:
        - ""
      resources:
        - "pods"
      verbs:
        - "get"
        - "list"
```

This TeamRoleBinding assigns the `pod-read` TeamRole to the Team named `my-team` in all Clusters with the label `environment: production`.

```yaml
apiVersion: greenhouse.sap/v1alpha2
kind: TeamRoleBinding
metadata:
  name: production-pod-read
spec:
  teamRef: my-team
  roleRef: pod-read
  clusterSelector:
    labelSelector:
      matchLabels:
        environment: production
```

Access granted by TeamRoleBinding can also be restricted to specified Namespaces. This can be achieved by specifying the `.spec.namespaces` field in the TeamRoleBinding.

Setting dedicated Namespaces results in RoleBindings being created in the specified Namespaces. The Team will then only have access to the Pods in the specified Namespaces. The TeamRoleBinding controller will create a non-existing Namespace, only if the field `.spec.createNamespaces` is set to `true` on the TeamRoleBinding. If this field is not set, the TeamRoleBinding controller will not create the Namespace or the RBAC resources.
Deleting a TeamRoleBinding will only result in the deletion of the RBAC resources but will never result in the deletion of the Namespace.

```yaml
apiVersion: greenhouse.sap/v1alpha2
kind: TeamRoleBinding
metadata:
  name: production-pod-read
spec:
  teamRef: my-team
  roleRef: pod-read
  clusterSelector:
    labelSelector:
      matchLabels:
        environment: production
  namespaces:
    - kube-system
  # createNamespaces: true # optional, if set the TeamRoleBinding will create the namespaces if they do not exist
```

This TeamRole has a `.spec.aggregationRule` set. This aggregationRule will be added to the ClusterRole created on the target clusters. With the aggregationRule set it will aggregate the ClusterRoles created by the TeamRoles with the label `aggregate: "true"`. The Team will have the permissions of both TeamRoles and will be able to `get`, `list`, `update` and `patch` Pods.

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: TeamRole
metadata:
  name: aggregated-role
spec:
  aggregationRule:
    clusterRoleSelectors:
    - matchLabels:
        "aggregate": "true"
```

```yaml
apiVersion: greenhouse.sap/v1alpha2
kind: TeamRoleBinding
metadata:
  name: aggregated-rolebinding
spec:
  teamRef: operators
  roleRef: aggregated-role
  clusterSelector:
    labelSelector:
      matchLabels:
        environment: production
```

### Updating TeamRoleBindings

Updating the RoleRef of a ClusterRoleBinding and RoleBinding is not allowed, but requires recreating the TeamRoleBinding resources. See [ClusterRoleBinding docs](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#clusterrolebinding-example) for more information.
This is to allow giving out permissions to update the subjects, while avoiding that privileges are changed. Furthermore, changing the TeamRole can change the extent of a binding significantly. Therefore it needs to be recreated.

After the TeamRoleBinding has been created, it can be updated with some limitations. Similarly to RoleBindings, the `.spec.roleRef` and `.spec.teamRef` can not be changed.
The TeamRoleBinding's `.spec.namespaces` can be amended to include more namespaces. However, the scope of the TeamRoleBinding cannot be changed. If a TeamRoleBinding has been created with `.spec.namespaces` specified, it is namespace-scoped, and cannot be changed to cluster-scoped by removing the `.spec.namespaces`. The reverse is true for a cluster-scoped TeamRoleBinding, where it is not possible to add `.spec.namespaces` once created.
