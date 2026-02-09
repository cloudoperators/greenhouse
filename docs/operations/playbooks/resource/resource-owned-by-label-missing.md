---
title: "ResourceOwnedByLabelMissing"
linkTitle: "ResourceOwnedByLabelMissing"
landingSectionIndex: false
weight: 1
description: >
  Playbook for the ResourceOwnedByLabelMissing Alert
---

## Alert Description

This alert fires when resources exist without the required `greenhouse.sap/owned-by` label for 15 minutes.

## What does this alert mean?

The `greenhouse.sap/owned-by` label is used to track resource ownership by Teams. This label should reference a Team with the `greenhouse.sap/support-group=true` label. Missing ownership labels make it difficult to:

- Track responsibility for resources
- Audit resource ownership
- Contact support teams for issues
- Enforce access control policies

## Diagnosis

### Identify the Affected Resource

The alert provides:

- `resource`: The type of resource (e.g., Plugin, Cluster, TeamRoleBinding)
- `namespace`: The namespace where the resource exists
- `name`: The name of the resource (in alert labels)

### Check the Resource

Retrieve the resource to inspect its labels:

```bash
kubectl get <resource-type> <resource-name> -n <namespace> -o yaml
```

Check the `metadata.labels` section for the `greenhouse.sap/owned-by` label.

### List All Resources Missing the Label

Find all resources of the same type missing the ownership label:

```bash
kubectl get <resource-type> --all-namespaces -o json | jq -r '.items[] | select(.metadata.labels["greenhouse.sap/owned-by"] == null) | "\(.metadata.namespace)/\(.metadata.name)"'
```

### Identify the Appropriate Owner Team

List support group teams in the namespace:

```bash
kubectl get teams -n <namespace> -l greenhouse.sap/support-group=true
```

### Add the Missing Label

Once you've identified the appropriate owner team, add the label:

```bash
kubectl label <resource-type> <resource-name> -n <namespace> greenhouse.sap/owned-by=<team-name>
```

For example:

```bash
kubectl label plugin my-plugin -n my-org greenhouse.sap/owned-by=platform-team
```

### Verify Webhooks are Working

Check if webhook validation is functioning properly:

```bash
# Check webhook pod status
kubectl get pods -n greenhouse -l app.kubernetes.io/component=webhook

# Check webhook logs for errors
kubectl logs -n greenhouse -l app.kubernetes.io/component=webhook --tail=100 | grep -i "owned-by"
```

### Prevent Future Occurrences

Ensure that:

- Webhooks are enabled and functioning
- Users are aware of the label requirement
- Resource creation processes include the ownership label

## Additional Resources

- [Greenhouse Team Documentation](../../../getting-started/core-concepts/teams.md)
- [Resource Ownership](../../../getting-started/operations/ownership.md)
