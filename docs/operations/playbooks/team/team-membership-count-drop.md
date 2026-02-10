---
title: "TeamMembershipCountDrop"
linkTitle: "TeamMembershipCountDrop"
landingSectionIndex: false
weight: 1
description: >
  Playbook for the TeamMembershipCountDrop Alert
---

## Alert Description

This alert fires when the number of members for a team has dropped by more than 5 in the last 5 minutes.

## What does this alert mean?

This alert detects sudden drops in team membership that could indicate:

- Accidental bulk removal of team members in the IdP
- SCIM synchronization issues causing member data loss
- IdP group configuration changes
- Potential security incidents (unauthorized access removal)

A drop of more than 5 members in 5 minutes is unusual and warrants investigation.

## Diagnosis

### Get the Team Resource

Retrieve the team resource to view current membership:

```bash
kubectl get team <team-name> -n <namespace> -o yaml
```

Check the current Team members in `.status.members`.

Check `.status.statusConditions`:

- **SCIMAccessReady**: Indicates if there is a connection to SCIM
- **SCIMAllMembersValid**: Shows if all members are valid (no invalid or inactive members)

### Check Organization SCIM Status

Verify that the organization's SCIM connection is working:

```bash
kubectl get organization <namespace> -o jsonpath='{.status.statusConditions.conditions[?(@.type=="SCIMAPIAvailable")]}'
```

### Check IdP Group Membership

Verify the current membership in the IdP group directly to confirm if the drop is legitimate or a sync issue:

1. Access your IdP (Identity Provider) console
2. Navigate to the group specified in `spec.mappedIdPGroup`
3. Compare the member list with what's shown in Greenhouse

### Check Controller Logs

Review the Greenhouse controller logs for SCIM synchronization errors:

```bash
kubectl logs -n greenhouse -l app=greenhouse --tail=200 | grep "<team-name>" | grep -E "scim|member|error" # requires permissions on the greenhouse namespace
```

Or access your logs sink for Greenhouse logs.

## Additional Resources

- [Greenhouse Team Documentation](../../../getting-started/core-concepts/teams.md)
- [Greenhouse Organization Documentation](../../../reference/components/organization)
