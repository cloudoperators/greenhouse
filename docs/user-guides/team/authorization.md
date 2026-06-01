---
title: "Managing team-owned resources"
linkTitle: "Team-owned resources"
description: >
  Elevated access to Greenhouse resources owned by your team.
---

## Overview

Greenhouse resources (Plugin, PluginPreset, Cluster, TeamRoleBinding) can be owned by a Team. Team members receive **elevated access** (get, update, patch, delete) on resources their Team owns — without requiring organization-wide admin permissions.

This is enforced by the [Authorization Webhook](../../../getting-started/operations/authorization-webhook), which checks that the requesting user's IdP token contains a `support-group:` claim matching the resource's `greenhouse.sap/owned-by` label.

## Prerequisites

Before your Team can use elevated access, two things must be in place:

1. **The Team must be marked as a support-group:**

   ```bash
   kubectl label team my-team greenhouse.sap/support-group=true -n my-organization
   ```

2. **Your IdP token must include a matching group claim.** The claim must have the `support-group:` prefix, for example `support-group:my-team`. Contact your IdP administrator if this claim is missing from your token.

## Claiming ownership of a resource

To make a resource manageable by your Team, set the `greenhouse.sap/owned-by` label to your Team name at creation time:

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

> **Note**: Organization admins must grant your Team the `create` verb on the resource type via RBAC before you can create new resources. Ownership-based elevated access only applies to **existing named resources** — it does not cover create or list/watch operations.

## Automating resource management with a ServiceAccount

When a Team has the `greenhouse.sap/support-group: "true"` label, Greenhouse automatically creates a ServiceAccount named `<team-name>-sa` in the organization namespace. This ServiceAccount carries the `greenhouse.sap/owned-by` label pre-set to the Team name, so it has the same elevated access to Team-owned resources as human team members.

Use this ServiceAccount for CI/CD pipelines, custom controllers, or scheduled jobs:

```bash
# Verify the ServiceAccount exists
kubectl get serviceaccount my-team-sa -n my-organization
```

### Requesting a token

Greenhouse creates a Role and RoleBinding that allow both support-group members and the ServiceAccount itself to request tokens. Use `kubectl create token` to obtain a credential for CI/CD use:

```bash
kubectl create token my-team-sa -n my-organization --duration=2160h
```

The actual token lifetime is capped by the API server's `--service-account-max-token-expiration` setting. The token can be used as a `Bearer` token when authenticating against the Greenhouse API server.

> **Note**: The ServiceAccount is created during Team reconciliation, which requires `spec.mappedIdPGroup` to be set and the Organization's SCIM integration to be configured. See [Setting up Team members synchronization](../../organization/creation#setting-up-team-members-synchronization-with-greenhouse) for SCIM configuration details.

> **Note**: The `greenhouse.sap/owned-by` label on the ServiceAccount is immutable once set — it cannot be changed or removed.

## Related Documentation

- [Authorization Webhook](../../../getting-started/operations/authorization-webhook) - How the webhook evaluates access
- [Team creation](../create) - Creating a Team and setting up IdP group mapping
