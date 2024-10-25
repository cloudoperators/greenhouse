---
title: "Organizations"
weight: 1
---

## What are Organizations?

Organizations are the top-level entities in Greenhouse. Each Organization gets a dedicated Namespace, that contains all resources bound to the Organization.
Greenhouse expects an Organization to provide it's own Identity Provider and currently supports _OIDC_ Identity Providers. Greenhouse also supports SCIM for syncing users and groups from an Identity Provider.

See [creating an Organization](./../../user-guides/organization/creation.md) for more details.

## Organization Namespace and Permissions

The Organization's Namespace in the Greenhouse cluster contains all resources bound to the Organization. This Namespace is automatically provisioned when a new Organization is created and shares the Organization's name.
Once the Namespace is created, Greenhouse will automatically seed RBAC [Roles](./../../../pkg/rbac/role.go) and [ClusterRoles](./../../../pkg/rbac/clusterrole.go) for the Organization. These are used to grant permissions for the Organization's resources to Teams.

- The Administrators of an Organization are specified via a identity provider (IDP) group during the creation of the Organization.
- The Administrators for Plugins and Clusters need to be defined by the Organization Admins via `RoleBindings` for the seeded Roles `role:<org-name>:cluster-admin` and `role:<org-name>:plugin-admin`.
- All authenticated users are considered members of the Organization and are granted the `organization:<org-name>` Role.

The following roles are seeded for each Organization:

| Name                            | Description                                                | ApiGroups                 | Resources                                                                                            | Verbs                       | Cluster scoped |
| ------------------------------- | ---------------------------------------------------------- | ------------------------- | ---------------------------------------------------------------------------------------------------- | --------------------------- | ---- |
| `role:<org-name>:admin`         | An admin of a Greenhouse `Organization`. This entails the permissions of `role:<org-name>:cluster-admin` and `role:<org-name>:plugin-admin`                    | `greenhouse.sap/v1alpha1` | \*                                                                                                   | \*                          | - |
|                                 |                                                            | `v1`                      | `secrets`                                                                                            | \*                          | - |
|                                 |                                                            | `""`                      | `pods`, `replicasets`, `deployments`, `statefulsets`, `daemonsets`, `cronjobs`, `jobs`, `configmaps` | `get`, `list`, `watch`      | - |
|                                 |                                                            | `monitoring.coreos.com`   | `alertmanagers`, `alertmanagerconfigs`                                                               | `get`, `list`, `watch`      | - |
| `role:<org-name>:cluster-admin` | An admin of Greenhouse `Clusters` within an `Organization` | `greenhouse.sap/v1alpha1` | `clusters`, `teamrolebindings`                                                                       | \*                          | - |
|                                 |                                                            | `v1`                      | `secrets`                                                                                            | `create`, `update`, `patch` | - |
| `role:<org-name>:plugin-admin`  | An admin of Greenhouse `Plugins` within an `Organization`  | `greenhouse.sap/v1alpha1` | `plugins`, `pluginpresets`                                                                           | \*                          | - |
|                                 |                                                            | `v1`                      | `secrets`                                                                                            | `create`, `update`, `patch` | - |
| `organization:<org-name>`        | A member of a Greenhouse `Organization`                    | `greenhouse.sap/v1alpha1` | \*                                                                                                   | `get`, `list`, `watch`      | - |
| `organization:<org-name>`       | A member of a Greenhouse `Organization`                    | `greenhouse.sap/v1alpha1` | `organizations`, `plugindefinitions`                                                                 | `get`, `list`, `watch`      | x |

## OIDC

Each Organization must specify the OIDC configuration for the Organization's IDP. This configuration is used together with [DEXIDP](https://dexidp.io/) to authenticate users in the Organization.

## SCIM

Each Organization can specify SCIM credentials which are used to syncronize users and groups from an Identity Provider. This makes it possible to view the members of a Team in the Greenhouse dashboard.
