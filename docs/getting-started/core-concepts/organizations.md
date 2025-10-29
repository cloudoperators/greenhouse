---
title: "Organizations"
weight: 1
---

## What are Organizations?

Organizations are the top-level entities in Greenhouse. Each Organization gets a dedicated Namespace, that contains all resources bound to the Organization.
Greenhouse expects an Organization to provide its own Identity Provider and currently supports _OIDC_ Identity Providers. Greenhouse also supports SCIM for syncing users and groups from an Identity Provider.

See [Creating an Organization](./../../../user-guides/organization/creation) for more details.

## Organization Namespace and Permissions

The Organization's Namespace in the Greenhouse cluster contains all resources bound to the Organization. This Namespace is automatically provisioned when a new Organization is created and shares the Organization's name.
Once the Namespace is created, Greenhouse will automatically seed RBAC [Roles](./../../../pkg/rbac/role.go) and [ClusterRoles](./../../../pkg/rbac/clusterrole.go) for the Organization. These are used to grant permissions for the Organization's resources to Teams.

- The Administrators of an Organization are specified via a identity provider (IDP) group during the creation of the Organization. Greenhouse automatically creates a Team called `<org-name>-admin`. This Team is also a [support group](teams.md#support-groups) and all alerts created by the Greenhouse controller are routed to this admin Team, if no other ownership is provided. See [operational processes](./../operations/processes.md) and [ownership](./../operations/ownership.md) for details.
- The Administrators for Plugins and Clusters need to be defined by the Organization Admins via `RoleBindings` for the seeded Roles `role:<org-name>:cluster-admin` and `role:<org-name>:plugin-admin`.
- All authenticated users are considered members of the Organization and are granted the `organization:<org-name>` Role.

See [Working with Organizations](./../../../reference/components/organization#role-based-access-control-within-the-organization-namespace) for details on the seeded Roles and ClusterRoles.

## OIDC

Each Organization must specify the OIDC configuration for the Organization's IDP. This configuration is used together with [DEXIDP](https://dexidp.io/) to authenticate users in the Organization.

## SCIM

Each Organization can specify SCIM credentials which are used to syncronize users and groups from an Identity Provider. This makes it possible to view the members of a Team in the Greenhouse dashboard.

## Next Steps

- [Creating an Organization](./../../../user-guides/organization/creation)
- [Organization reference](./../../reference/components/organization)
