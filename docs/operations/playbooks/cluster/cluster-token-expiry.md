---
title: "ClusterTokenExpiry"
linkTitle: "ClusterTokenExpiry"
landingSectionIndex: false
weight: 2
description: >
  Playbook for the ClusterTokenExpiry Alert
---

## Alert Description

**Severity:** Warning  
**Alert Name:** GreenhouseClusterTokenExpiry

This alert fires when the kubeconfig token for a cluster will expire in less than 20 hours.

**Alert Message:**

```
The kubeconfig token for {{ $labels.cluster }} in {{ $labels.namespace }} will expire in less than 20 hours.
```

## What does this alert mean?

Greenhouse has two ways of authenticating to Clusters:

- [OIDC trust](../../user-guides/cluster/oidc_connectivity.md) - Preferred method, credential-less authentication
- [Initial kubeconfig](../../user-guides/cluster/onboarding.md) - Traditional method using kubeconfig credentials

This alert only fires when initially a kubeconfig was provided. Greenhouse will create a Service Account on the target Cluster and keep a kubeconfig with a token scoped to this SA on the Greenhouse Cluster. These tokens have a limited validity period and are auto-rotated by the Greenhouse controller. When a token is about to expire, this alert fires. Since Greenhouse auto-rotates these tokens, this alert indicates that Greenhouse cannot (or could not) interact properly with the Cluster.

If the token expires without being refreshed:

- Greenhouse will lose access to the cluster
- The cluster will become NotReady
- Plugin deployments and updates will fail
- RBAC synchronization will stop working

## Quick fix

With this first attempt to fix you will delete the `.data.greenhousekubeconfig` entry of the Secret holding the authentication credentials for the Cluster. This will trigger reconciliation of the Cluster by the Greenhouse controller.

> Important! Please make sure you have a valid `.data.kubeconfig` entry (base64 encoded) in the Secret.

### Step 1: Verify the Secret exists

```bash
kubectl get secret <cluster-name> -n <namespace>
```

### Step 2: Update the kubeconfig (if needed)

If you need to replace the kubeconfig with a new one:

```bash
# Base64 encode your kubeconfig
KUBECONFIG_BASE64=$(cat <path-to-new-kubeconfig> | base64)

# Patch the secret to update the kubeconfig
kubectl patch secret <cluster-name> -n <namespace> \
  --type='json' \
  -p='[{"op": "replace", "path": "/data/kubeconfig", "value":"'$KUBECONFIG_BASE64'"}]'
```

### Step 3: Remove the greenhousekubeconfig to trigger token refresh

```bash
kubectl patch secret <cluster-name> -n <namespace> \
  --type='json' \
  -p='[{"op": "remove", "path": "/data/greenhousekubeconfig"}]'
```

This will trigger the Greenhouse controller to:

1. Detect the missing `greenhousekubeconfig`
2. Use the `kubeconfig` to authenticate to the remote cluster
3. Create/verify the ServiceAccount on the remote cluster
4. Generate a new token and update the `greenhousekubeconfig` entry

## Further Diagnosis

You might want to find out, why Greenhouse could not auto-rotate the token in the first place:

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

Look for messages about token refresh operations or authentication issues.

## Additional Resources

- [Greenhouse Cluster Documentation](../../../reference/api/cluster.md)
- [Cluster Onboarding](../../../user-guides/cluster/onboarding.md)
- [Cluster OIDC connectivity](../../../user-guides/cluster/oidc_connectivity.md)
