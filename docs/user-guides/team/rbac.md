---
title: "Role-based access control"
description: >
   Creating and managing roles and permissions in Greenhouse.
---

## Before you begin

This guides describes how to manage roles and permissions in Greenhouse.  

While all members of an organization can see the configured permissions, configuration of these requires **organization admin privileges**.

# Greenhouse RBAC user guide

Role-Based Access Control (RBAC) in Greenhouse allows organization administrators to regulate access to Kubernetes resources in onboarded clusters based on the roles of individual users within an organization.    
Greenhouse utilizes custom RBAC configurations with `TeamRole` and `TeamRoleBinding` to manage access controls effectively.

## Overview

- **TeamRole**: Defines a set of permissions that can be assigned to teams.
- **TeamRoleBinding**: Assigns a `TeamRole` to a specific team for certain clusters.

## Defining Team Roles

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

## Assigning Roles to Teams

Roles are assigned to teams through the TeamRoleBinding configuration, which links teams to their respective roles within specific clusters.

This TeamRoleBinding assigns the `cluster-admin` role to the team named `my-team` in the cluster named `my-cluster`.

Example: `team-rolebindings.yaml`

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: TeamRoleBinding
metadata:
  name: my team
spec:
  teamRef: my-team
  roleRef: cluster-admin
  clusterName: my-cluster
```
