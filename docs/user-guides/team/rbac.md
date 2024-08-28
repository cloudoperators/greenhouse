---
title: "Role-based access control"
description: >
   Creating and managing roles and permissions in Greenhouse.
---

## Before you begin

This guides describes how to manage roles and permissions in Greenhouse with the help of TeamRoles and TeamRoleBindings.

While all members of an organization can see the permissions configured with TeamRoles & TeamRoleBindings, configuration of these requires **OrganizationAdmin privileges**.

## Greenhouse Team RBAC user guide

Role-Based Access Control (RBAC) in Greenhouse allows organization administrators to regulate access to Kubernetes resources in onboarded Clusters based on the roles of individual users within an Organization.
Within Greenhouse the RBAC on remote Clusters is managed using `TeamRole` and `TeamRoleBinding`. These two Custom Resource Defintions allow for fine-grained control over the permissions of each Team within each Cluster and Namespace.

## Overview

- **TeamRole**: Defines a set of permissions that can be assigned to teams.
- **TeamRoleBinding**: Assigns a `TeamRole` to a specific `Team` for certain `Clusters` and (optionally) `Namespaces`.

## Defining TeamRoles

`TeamRoles` define what actions a team can perform within the Kubernetes cluster.
Common roles including the below `cluster-admin` are pre-defined within each organization.

### Cluster administrator

This TeamRole named cluster-admin grants full access to all resources in all API groups.

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: TeamRole
metadata:
  name: cluster-admin
spec:
  rules:
    - apiGroups:
        - "*"
      resources:
        - "*"
      verbs:
        - "*"
```

## Defining TeamRoleBindings

`TeamRoleBindings` define the permissions of a Greenhouse Team within Clusters by linking to a specific `TeamRole`.
TeamRoleBindings have a simple specification that links a Team, a TeamRole, one or more Clusters and optionally a one or more Namespaces together. Once the TeamRoleBinding is created, the Team will have the permissions defined in the TeamRole within the specified Clusters and Namespaces. This allows for fine-grained control over the permissions of each Team within each Cluster.
The TeamRoleBinding Controller within Greenhouse deploys rbacv1 resources to the targeted Clusters. The referenced TeamRole is created as a rbacv1.ClusterRole. In case the TeamRoleBinding references a Namespace, the Controller will create a rbacv1.RoleBinding which links the Team with the rbacv1.ClusterRole. In case no Namespace is referenced, the Controller will create a rbacv1.ClusterRoleBinding instead.

### Assigning TeamRoles to Teams on a single Cluster

Roles are assigned to teams through the TeamRoleBinding configuration, which links teams to their respective roles within specific clusters.

This TeamRoleBinding assigns the `cluster-admin` TeamRole to the Team named `my-team` in the Cluster named `my-cluster`.

Example: `team-rolebindings.yaml`

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: TeamRoleBinding
metadata:
  name: my-cluster-admin
spec:
  teamRef: my-team
  roleRef: cluster-admin
  clusterName: my-cluster
```

### Assigning TeamRoles to Teams on multiple Clusters

It is also possible to use a LabelSelector to assign TeamRoleBindings to multiple Clusters at once.

This TeamRoleBinding assigns the `cluster-admin` TeamRole to the Team named `my-team` in all Clusters with the label `environment: production`.

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: TeamRoleBinding
metadata:
  name: production-cluster-admins
spec:
  teamRef: my-team
  roleRef: cluster-admin
  clusterSelector:
    matchLabels:
      environment: production
```

### Aggregating TeamRoles

It is possible with RBAC to aggregate rbacv1.ClusterRoles. This is also supported for TeamRoles. By specifying `.spec.Labels` on a TeamRole the resulting ClusterRole on the target cluster will have the same labels set. Then it is possible to aggregate multiple ClusterRole resources by using a rbacv1.AggregationRule. This can be specified on a TeamRole by setting `.spec.aggregationRule`.

[!NOTE] A TeamRole is only created on a cluster if it is referenced by a TeamRoleBinding. If a TeamRole is not referenced by a TeamRoleBinding it will not be created on any target cluster. A TeamRoleBinding referencing a TeamRole with an aggregationRule will only provide the correct access, if there is at least one TeamRoleBinding referencing a TeamRole with the corresponding label deployed to the same cluster.

The following example shows how to an AggregationRule can be used with TeamRoles and TeamRoleBindings.

This TeamRole has a label set. This label will be set on the resulting ClusterRole on the target cluster.

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: TeamRole
metadata:
  name: cluster-admin
spec:
  labels:
    aggregate: "true"
  rules:
    - apiGroups:
        - "*"
      resources:
        - "*"
      verbs:
        - "*"
```

This TeamRoleBinding assigns the `cluster-admin` TeamRole to the Team named `my-team` in all Clusters with the label `environment: production`.

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: TeamRoleBinding
metadata:
  name: production-cluster-admins
spec:
  teamRef: my-team
  roleRef: cluster-admin
  clusterSelector:
    matchLabels:
      environment: production
```

This TeamRole has an aggregationRule set. This aggregationRule will be added to the ClusterRole created on the target clusters.

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
apiVersion: greenhouse.sap/v1alpha1
kind: TeamRoleBinding
metadata:
  name: aggregated-rolebinding
spec:
  teamRef: my-team
  roleRef: aggregated-role
  clusterSelector:
    matchLabels:
      environment: production
```
