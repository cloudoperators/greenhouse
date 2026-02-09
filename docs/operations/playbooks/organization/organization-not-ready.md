---
title: "OrganizationNotReady"
linkTitle: "OrganizationNotReady"
landingSectionIndex: false
weight: 1
description: >
  Playbook for the OrganizationNotReady Alert
---

## Alert Description

This alert fires when a Greenhouse Organization has not been ready for more than 15 minutes.

## What does this alert mean?

An Organization in Greenhouse represents a tenant and serves as the primary namespace for all resources belonging to that organization. When an Organization is not ready, it indicates that Greenhouse cannot properly initialize or manage the organization's resources.

This could be due to:

- Issues with the organization's namespace creation or configuration
- RBAC setup failures
- IdP (Identity Provider) configuration problems
- Service proxy provisioning issues
- Default team role configuration problems

## Diagnosis

### Get the Organization Resource

Retrieve the organization resource to view its current status:

```bash
kubectl get organization <organization-name> -o yaml
```

Or use kubectl describe for a more readable output:

```bash
kubectl describe organization <organization-name>
```

### Check the Status Conditions

Look at the `status.statusConditions` section in the organization resource. Pay special attention to:

- **Ready**: The main indicator of organization health
- **NamespaceCreated**: Indicates if the organization namespace was successfully created
- **OrganizationRBACConfigured**: Shows if RBAC for the organization is properly configured
- **OrganizationDefaultTeamRolesConfigured**: Indicates if default team roles are configured
- **ServiceProxyProvisioned**: Shows if the service proxy is provisioned
- **OrganizationOICDConfigured**: Indicates if OIDC is configured correctly
- **OrganizationAdminTeamConfigured**: Shows if the admin team is configured for the organization

### Check Controller Logs

Review the Greenhouse controller logs for more detailed error messages:

```bash
kubectl logs -n greenhouse -l app=greenhouse --tail=100 | grep "<organization-name>" | grep "error"
```

## Additional Resources

- [Greenhouse Organization Documentation](../../../reference/components/organization.md)
- [Getting Started with Organizations](../../../getting-started/core-concepts/organizations.md)
