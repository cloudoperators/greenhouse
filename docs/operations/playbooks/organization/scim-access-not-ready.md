---
title: "SCIMAccessNotReady"
linkTitle: "SCIMAccessNotReady"
landingSectionIndex: false
weight: 2
description: >
  Playbook for the SCIMAccessNotReady Alert
---

## Alert Description

**Severity:** Warning  
**Alert Name:** GreenhouseSCIMAccessNotReady

This alert fires when the SCIM access for an organization is not ready for more than 15 minutes.

**Alert Message:**

```
The SCIM access for organization '{{ $labels.organization }}' is not ready for more than 15 minutes. Team member sync may not be operational.
```

## What does this alert mean?

SCIM (System for Cross-domain Identity Management) is used by Greenhouse to synchronize team members from external identity providers. When SCIM access is not ready, it indicates that Greenhouse cannot properly communicate with the SCIM API to fetch and synchronize user and group information.

This could be due to:

- Invalid or missing SCIM credentials in the referenced secret
- Network connectivity issues to the SCIM API endpoint
- SCIM API authentication failures
- Incorrect SCIM configuration in the Organization spec

While SCIM access is not crucial:

- Team member synchronization will not work
- New members added in the IdP will not appear in Greenhouse teams
- Member removals in the IdP will not be reflected in Greenhouse

## Diagnosis

### Get the Organization Resource

Retrieve the organization resource to check SCIM configuration:

```bash
kubectl get organization <organization-name> -o yaml
```

Look for the `spec.authentication.scim` section to see the SCIM configuration.

### Check the Status Conditions

Look at the `status.statusConditions` section in the organization resource. Pay special attention to:

- **Ready**: The main indicator of organization health
- **SCIMAPIAvailable**: Indicates if there is a connection to the SCIM API

Check for specific reasons:

- **SecretNotFound**: The secret with SCIM credentials is not found
- **SCIMRequestFailed**: A request to SCIM API failed
- **SCIMConfigErrorReason**: SCIM config is missing or invalid

### Verify the SCIM Secret

Check if the referenced secret exists and contains the correct credentials:

```bash
# Check if the secret exists
kubectl get secret <scim-secret-name> -n <organization-name>

# View the secret keys (not the values)
kubectl get secret <scim-secret-name> -n <organization-name> -o jsonpath='{.data}' | jq 'keys'
```

### Check Controller Logs

Review the Greenhouse controller logs for SCIM-related errors:

```bash
kubectl logs -n greenhouse -l app=greenhouse --tail=100 | grep "<organization-name>" | grep -i "scim\|error"
```

## Additional Resources

- [Greenhouse Organization Documentation](../../../reference/components/organization.md)
