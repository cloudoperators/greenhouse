---
title: "ClusterNotReady"
linkTitle: "ClusterNotReady"
landingSectionIndex: false
weight: 1
description: >
  Playbook for the ClusterNotReady Alert
---

## Alert Description

**Severity:** Warning  
**Alert Name:** GreenhouseClusterNotReady

This alert fires when a Greenhouse-managed cluster has not been ready for more than 15 minutes.

**Alert Message:**

```
Cluster {{ $labels.cluster }} in namespace {{ $labels.namespace }} has not been ready for more than 15 minutes.
```

## What does this alert mean?

The Greenhouse controller monitors the health of all registered clusters. When a cluster is not ready, it indicates that the Greenhouse operator cannot properly communicate with or manage resources on that cluster. This could be due to:

- Network connectivity issues between Greenhouse and the cluster
- Invalid or expired kubeconfig credentials
- The cluster API server being unavailable
- Insufficient permissions for Greenhouse to access the cluster
- Node issues preventing the cluster from being operational

## Diagnosis

### Get the Cluster Resource

Retrieve the cluster resource to view its current status:

```bash
kubectl get cluster <cluster-name> -n <namespace> -o yaml
```

Or use kubectl describe for a more readable output:

```bash
kubectl describe cluster <cluster-name> -n <namespace>
```

### Check the Status Conditions

Look at the `status.statusConditions` section in the cluster resource. Pay special attention to:

- **Ready**: The main indicator of cluster health
- **KubeConfigValid**: Indicates if credentials are valid
- **AllNodesReady**: Shows if all nodes in the cluster are ready
- **PermissionsVerified**: Confirms Greenhouse has required permissions
- **ManagedResourcesDeployed**: Indicates if Greenhouse resources were deployed

### Check Controller Logs

Review the Greenhouse controller and webhook logs for more detailed error messages:

```bash
kubectl logs -n greenhouse -l app=greenhouse
 --tail=100 | grep "<cluster-name>"
```

## Additional Resources

- [Greenhouse Cluster Documentation](../../reference/api/cluster.md)
- [Kubernetes Cluster Troubleshooting](https://kubernetes.io/docs/tasks/debug/)
