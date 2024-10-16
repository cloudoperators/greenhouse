---
title: "Organizations"
weight: 1
---

## What are Organizations?

Organizations are the top-level entities in Greenhouse. Based on the name of the Organization a namespace is automatically created. This namespace contains all resources created by the  Organization, such as Teams, Clusters, and Plugins. As well as some seeded RBAC roles and Team RBAC resources.
Furthermore, the Organization is specced with the authentication details for OIDC and SCIM. These are used for authentication and user management respectively.

## Organization Namespace

This is the namespace in the Greenhouse cluster where all resources of the Organization are stored. The namespace is created automatically when the Organization is created and is named after the Organization.
Once the namespace is created, Greenhouse will automatically seed RBAC [Roles](./../../../pkg/rbac/role.go) and [ClusterRoles](./../../../pkg/rbac/clusterrole.go) for the Organization, which are used to grant permissions for these organizations resources to users and teams.

- The Administrators of an Organization are defined via a IDP Group specified on the Organization resource.
- The Administrators for Plugins and Clusters need to be defined by the Organization Admins via `RoleBindings` for the seeded Roles `role:<org-name>:cluster-admin` and `role:<org-name>:plugin-admin`.
- All authenticated users are considered members of the Organization and are granted the `organization:<org-name>` Role.

The following roles are seeded for each Organization:

| Name                            | Description                                                | ApiGroups                 | Resources                                                                                            | Verbs                       | Cluster scoped |
| ------------------------------- | ---------------------------------------------------------- | ------------------------- | ---------------------------------------------------------------------------------------------------- | --------------------------- | ---- |
| `role:<org-name>:admin`         | An admin of a Greenhouse `Organization`                    | `greenhouse.sap/v1alpha1` | \*                                                                                                   | \*                          | - |
|                                 |                                                            | `v1`                      | `secrets`                                                                                            | \*                          | - |
|                                 |                                                            | `""`                      | `pods`, `replicasets`, `deployments`, `statefulsets`, `daemonsets`, `cronjobs`, `jobs`, `configmaps` | `get`, `list`, `watch`      | - |
|                                 |                                                            | `monitoring.coreos.com`   | `alertmanagers`, `alertmanagerconfigs`                                                               | `get`, `list`, `watch`      | - |
| `role:<org-name>:cluster-admin` | An admin of Greenhouse `Clusters` within an `Organization` | `greenhouse.sap/v1alpha1` | `clusters`, `teamrolebindings`                                                                       | \*                          | - |
|                                 |                                                            | `v1`                      | `secrets`                                                                                            | `create`, `update`, `patch` | - |
| `role:<org-name>:plugin-admin`  | An admin of Greenhouse `Plugins` within an `Organization`  | `greenhouse.sap/v1alpha1` | `plugins`, `pluginpresets`                                                                           | \*                          | - |
|                                 |                                                            | `v1`                      | `secrets`                                                                                            | `create`, `update`, `patch` | - |
| `role:<org-name>:member`        | A member of a Greenhouse `Organization`                    | `greenhouse.sap/v1alpha1` | \*                                                                                                   | `get`, `list`, `watch`      | - |
| `organization:<org-name>`       | A member of a Greenhouse `Organization`                    | `greenhouse.sap/v1alpha1` | `organizations`, `plugindefinitions`                                                                 | `get`, `list`, `watch`      | x |

## OIDC

The Organization resources contains the OIDC configuration for the Organization. This configuration is used togethe r with DEXIDP to authenticate users in the Organization. The OIDC configuration is stored in the `oidc` field of the Organization resource.

## SCIM

The Organization resources contains the SCIM configuration for the Organization. The SCIM configuration is stored in the `scim` field of the Organization resource. This configuration is used syncronize the members of a Team from the SCIM API and allow to show the members of a Team in the Greenhouse dashboard.
