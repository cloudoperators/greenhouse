---
title: "Team creation"
linkTitle: "Team creation"
description: >
  Create a team within your organization
---

## Before you begin

This guides describes how to create a team in your Greenhouse organization. 

While all members of an organization can see existing teams, their management requires **organization admin privileges**. 

## Creating a team

The team resource is used to structure members of your organization and assign fine-grained access and permission levels.

Each Team must be backed by a group in the identity provider (IdP) of the Organization.
   * IdP group should be set on the `mappedIdPGroup` field in Team configuration.
   * This, along with SCIM API configured in the Organization, allows for synchronization of Team members with Greenhouse.

```
NOTE: The UI is currently in development. For now this guides describes the onboarding workflow via command line.
```

1. To onboard a new cluster provide the kubeconfig file with a static, short-lived token.  
   It should look similar to this example:
   ```
   cat <<EOF | kubectl apply -f -
      apiVersion: greenhouse.sap/v1alpha1
      kind: Team
      metadata:
      name: <name>
      spec:
         description: My new team
         mappedIdPGroup: <IdP group name>
   EOF
   ```

## Managing resources as a Team

The `greenhouse.sap/owned-by` label on a Greenhouse resource (Plugin, PluginPreset, Cluster, TeamRoleBinding) identifies which Team owns it. The [Authorization Webhook](./../../../getting-started/operations/authorization-webhook) uses this label to grant team members elevated access (get, update, patch, delete) on resources they own, without requiring organization-wide admin permissions.

### Working with owned resources as a team member

To make a resource manageable by your team, set the `greenhouse.sap/owned-by` label to your team name:

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: Plugin
metadata:
  name: my-plugin
  namespace: my-organization
  labels:
    greenhouse.sap/owned-by: my-team
spec:
  # ...
```

Or label an existing resource:

```bash
kubectl label plugin my-plugin greenhouse.sap/owned-by=my-team -n my-organization
```

Your Team must also be marked as a support-group:

```bash
kubectl label team my-team greenhouse.sap/support-group=true -n my-organization
```

For the webhook to recognise you as a team member, your IdP token must include a group claim with the `support-group:` prefix matching your team name — for example `support-group:my-team`. Contact your IdP administrator if this claim is missing.

### Automating resource management with a ServiceAccount

When a Team has the `greenhouse.sap/support-group: "true"` label, Greenhouse automatically creates a ServiceAccount named `<team-name>-sa` in the organization namespace. This ServiceAccount carries the `greenhouse.sap/owned-by` label pre-set to the team name, so it has the same elevated access to team-owned resources as human team members.

Use this ServiceAccount for CI/CD pipelines, custom controllers, or scheduled jobs:

```bash
# Verify the ServiceAccount exists
kubectl get serviceaccount my-team-sa -n my-organization
```

> **Note**: The ServiceAccount is created during Team reconciliation, which requires `spec.mappedIdPGroup` to be set and the Organization's SCIM integration to be configured. See [Setting up Team members synchronization](./../../organization/creation#setting-up-team-members-synchronization-with-greenhouse) for SCIM configuration details.

> **Note**: The `greenhouse.sap/owned-by` label on the ServiceAccount is immutable once set — it cannot be changed or removed.
