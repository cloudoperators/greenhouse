---
title: "Cluster offboarding"
linkTitle: "Offboarding"
weight: 3
description: >
  Offboarding an existing Kubernetes cluster in Greenhouse.
---

## Content Overview

- [Pre-requisites](#pre-requisites)
- [Off-boarding](#off-boarding)
- [Troubleshooting](#troubleshooting)

This guides describes how to off-board an existing Kubernetes cluster in your Greenhouse organization.  

While all members of an organization can see existing clusters, their management requires `org-admin` or
`cluster-admin` privileges. See [RBAC with the Organization namespace](./../../../reference/components/organization/#role-based-access-control-within-the-organization-namespace) for details.

```
:information_source: The UI is currently in development. For now this guide describes the onboarding workflow via command line.
```

### Pre-requisites

Off-boarding a `Cluster` in Greenhouse requires authenticating to the `greenhouse` cluster via `kubeconfig` file:

- `greenhouse`: The cluster where Greenhouse installation is running on.
- `organization-admin` or `cluster-admin` privileges is needed for deleting a `Cluster` resource.


## Off-boarding

Off-board a `Cluster` in Greenhouse is initiated by calling the command:

```shell
kubectl --namespace=<greenhouse-organization-name> delete cluster <cluster-name>
```

| :exclamation: Deleting the `Cluster` resource automatically uninstalls any deployed plugins as part of the cleanup process. |
|-----------------------------------------------------------------------------------------------------------------------------|

## Troubleshooting

If the cluster deletion has failed, you can troubleshoot the issue by inspecting -

1. `Cluster` resource status conditions, specifically the `KubeConfigValid` condition.
2. status conditions of the `Plugin` resources associated with the `Cluster` resource. There will be a clear indication of the issue in the `HelmReleaseDeployed` condition (set to `false` when Helm reconciliation fails).
