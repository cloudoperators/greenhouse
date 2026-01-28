---
title: "ClusterKubernetesVersionOutOfMaintenance"
linkTitle: "ClusterKubernetesVersionOutOfMaintenance"
landingSectionIndex: false
weight: 3
description: >
  Playbook for the ClusterKubernetesVersionOutOfMaintenance Alert
---

## Alert Description

**Severity:** Warning  
**Alert Name:** GreenhouseClusterKubernetesVersionOutOfMaintenance

This alert fires when a cluster is running a Kubernetes version that is out of maintenance.

**Alert Message:**

```
Cluster {{ $labels.cluster }} in namespace {{ $labels.namespace }} is running Kubernetes version {{ $labels.version }} which is out of maintenance.
```

## What does this alert mean?

Kubernetes versions have a limited support lifecycle. When a version goes out of maintenance, it no longer receives security patches or bug fixes. Running clusters on unsupported versions poses security risks and may lead to compatibility issues with newer features and tools.

This alert fires when a cluster is detected running Kubernetes version which are out of the official Kubernetes maintenance window.

## Fix

Update the kubernetes version of the target Cluster.

## Diagnosis

### Get the Cluster Resource

Check the detected Kubernetes version:

```bash
kubectl get cluster <cluster-name> -n <namespace> -o yaml
```

Look for the `status.kubernetesVersion` field to see the current version.

### Verify the Version

Check the version directly on the target cluster:

```bash
kubectl --kubeconfig=<target-cluster-kubeconfig> version --short
```

## Additional Resources

- [Greenhouse Cluster Documentation](../../../reference/api/cluster.md)
- [Kubernetes Version Skew Policy](https://kubernetes.io/releases/version-skew-policy/)
- [Kubernetes Release Information](https://kubernetes.io/releases/)
