---
title: "TeamRoleBindingNotReady"
linkTitle: "TeamRoleBindingNotReady"
landingSectionIndex: false
weight: 2
description: >
  Playbook for the TeamRoleBindingNotReady Alert
---

## Alert Description

This alert fires when a TeamRoleBinding has not been ready for more than 15 minutes.

## What does this alert mean?

A TeamRoleBinding in Greenhouse maps a Team to a TeamRole on one or more clusters. When a TeamRoleBinding is not ready, it means that the RBAC resources (RoleBindings or ClusterRoleBindings) could not be properly created on the target clusters, preventing team members from accessing the clusters with the intended permissions.

This could be due to:

- Cluster access issues (cluster not ready or inaccessible)
- Permission issues on the target Cluster
- Referenced Team or TeamRole does not exist
- Cluster selector not matching any clusters

## Diagnosis

### Get the TeamRoleBinding Resource

Retrieve the TeamRoleBinding resource to view its current status:

```bash
kubectl get teamrolebinding <trb-name> -n <namespace> -o yaml
```

Or use the shortname:

```bash
kubectl get trb <trb-name> -n <namespace> -o yaml
```

### Check the Status Conditions

Look at the `status.statusConditions` section. Pay special attention to:

- **Ready**: The main indicator of TeamRoleBinding health
- **RBACReady**: Indicates if the RBAC resources are ready on the clusters

Common failure reasons:

- **RBACReconcileFailed**: Not all RBAC resources have been successfully reconciled
- **EmptyClusterList**: The clusterSelector and clusterName do not match any existing clusters
- **TeamNotFound**: The referenced Team does not exist
- **ClusterConnectionFailed**: Cannot connect to the target cluster
- **ClusterRoleFailed**: ClusterRole could not be created on the remote cluster
- **RoleBindingFailed**: RoleBinding could not be created on the remote cluster
- **CreateNamespacesFailed**: Namespaces could not be created (when createNamespaces is enabled)

### Check Propagation Status

The `status.clusters` field shows the propagation status per cluster:

```bash
kubectl get trb <trb-name> -n <namespace> -o jsonpath='{.status.clusters}' | jq
```

This will show which specific clusters are failing and why.

### Verify Referenced Resources

Check if the referenced Team and TeamRole exist:

```bash
# Check Team
kubectl get team <team-name> -n <namespace>

# Check TeamRole
kubectl get teamrole <teamrole-name> -n <namespace>
```

### Check Cluster Availability

If the issue is cluster connectivity, check the target cluster status:

```bash
kubectl get cluster <cluster-name> -n <namespace>
```

See the [ClusterNotReady playbook](../cluster/cluster-not-ready.md) for cluster troubleshooting.

### Check Controller Logs

Review the Greenhouse controller logs for detailed error messages:

```bash
kubectl logs -n greenhouse -l app=greenhouse --tail=200 | grep "<trb-name>" | grep "error" # requires permissions on the greenhouse namespace
```

Or access your logs sink for Greenhouse logs.

## Additional Resources

- [Greenhouse Team(RoleBinding) Documentation](../../../reference/components/team.md#teamrolebinding)
- [Greenhouse Team RBAC User Guide](../../../user-guides/team/rbac.md)
