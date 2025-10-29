---
title: "Cluster offboarding"
linkTitle: "Offboarding"
weight: 3
description: >
  Offboarding an existing Kubernetes cluster in Greenhouse.
---

## Content Overview

- [Pre-requisites](#pre-requisites)
- [Schedule Deletion](#schedule-deletion)
- [Impact](#impact)
- [Immediate Deletion](#immediate-deletion)
- [Troubleshooting](#trouble-shooting)

This guides describes how to off-board an existing Kubernetes cluster in your Greenhouse organization.  

While all members of an organization can see existing clusters, their management requires [`org-admin` or `cluster-admin` privileges](./../../getting-started/core-concepts/organizations).

```
NOTE: The UI is currently in development. For now this guide describes the onboarding workflow via command line.
```

### Pre-requisites

Offboarding a `Cluster` in Greenhouse requires authenticating to the `greenhouse` cluster via `kubeconfig` file:

- `greenhouse`: The cluster where Greenhouse installation is running on.
- `organization-admin` or `cluster-admin` privileges is needed for deleting a `Cluster` resource.

### Schedule Deletion

By default `Cluster` resource deletion is blocked by `ValidatingWebhookConfiguration` in Greenhouse. 
This is done to prevent accidental deletion of cluster resources.

List the clusters in your Greenhouse organization:

```shell
kubectl --namespace=<greenhouse-organization-name> get clusters
```

A typical output when you run the command looks like

```shell
NAME          AGE    ACCESSMODE   READY
mycluster-1   15d    direct       True
mycluster-2   35d    direct       True
mycluster-3   108d   direct       True
```

Delete a `Cluster` resource by annotating it with `greenhouse.sap/delete-cluster: "true"`.

Example:

```shell
kubectl annotate cluster mycluster-1 greenhouse.sap/delete-cluster=true --namespace=my-org
```

Once the `Cluster` resource is annotated, the `Cluster` will be scheduled for deletion in 48 hours (UTC time). 
This is reflected in the `Cluster` resource annotations and in the status conditions.

View the deletion schedule by inspecting the `Cluster` resource:

```shell
kubectl get cluster mycluster-1 --namespace=my-org -o yaml
````

A typical output when you run the command looks like

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: Cluster
metadata:
  annotations:
    greenhouse.sap/delete-cluster: "true"
    greenhouse.sap/deletion-schedule: "2025-01-17 11:16:40"
  finalizers:
  - greenhouse.sap/cleanup
  name: mycluster-1
  namespace: my-org
spec:
  accessMode: direct
  kubeConfig:
    maxTokenValidity: 72
status:
  ...
  statusConditions:
    conditions:
    ...
    - lastTransitionTime: "2025-01-15T11:16:40Z"
      message: deletion scheduled at 2025-01-17 11:16:40
      reason: ScheduledDeletion
      status: "False"
      type: Delete
```

In order to cancel the deletion, you can remove the `greenhouse.sap/delete-cluster` annotation:

```shell
kubectl annotate cluster mycluster-1 greenhouse.sap/delete-cluster- --namespace=my-org
```

> the `-` at the end of the annotation name is used to remove the annotation.

### Impact

When a `Cluster` resource is scheduled for deletion, all `Plugin` resources associated with the `Cluster` resource will skip the reconciliation process.

When the deletion schedule is reached, the `Cluster` resource will be deleted and all associated resources `Plugin` resources will be deleted as well.


### Immediate Deletion

In order to delete a `Cluster` resource immediately - 

1. annotate the `Cluster` resource with `greenhouse.sap/delete-cluster`. (see [Schedule Deletion](#schedule-deletion))
2. update the `greenhouse.sap/deletion-schedule` annotation to the current date and time.

You can also annotate the `Cluster` resource with `greenhouse.sap/delete-cluster` and `greenhouse.sap/deletion-schedule` at the same time and set the current date and time for deletion.

> The time and date should be in `YYYY-MM-DD HH:MM:SS` format or golang's `time.DateTime` format.
> The time should be in UTC timezone.


## Troubleshooting

If the cluster deletion has failed, you can troubleshoot the issue by inspecting -

1. `Cluster` resource status conditions, specifically the `KubeConfigValid` condition.
2. status conditions of the `Plugin` resources associated with the `Cluster` resource. There will be a clear indication of the issue in `HelmReconcileFailed` condition.
