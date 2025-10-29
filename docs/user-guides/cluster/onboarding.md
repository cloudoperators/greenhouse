---
title: "Cluster onboarding"
linkTitle: "Onboarding"
weight: 1
description: >
  Onboard an existing Kubernetes cluster to Greenhouse.
---

## Content Overview

- [Preparation](#preparation)
- [Onboard](#onboard)
- [After onboarding](#after-onboarding)
- [Troubleshooting](#troubleshooting)

This guides describes how to onboard an existing Kubernetes cluster to your Greenhouse organization.  
If you don't have an organization yet please reach out to the Greenhouse administrators.

While all members of an organization can see existing clusters, their management requires `org-admin` or
`cluster-admin` privileges. See [RBAC with the Organization namespace](./../../../reference/components/organization/#role-based-access-control-within-the-organization-namespace) for details.

```
:information_source: The UI is currently in development. For now this guide describes the onboarding workflow via command line.
```

### Preparation

Download the latest `greenhousectl` binary from the Greenhouse [releases](https://github.com/cloudoperators/greenhouse/releases).

Onboarding a `Cluster` to Greenhouse will require you to authenticate to two different Kubernetes clusters via respective `kubeconfig` files:

- `greenhouse`: The cluster your Greenhouse installation is running on. You need `organization-admin` or `cluster-admin` privileges.
- `bootstrap`: The cluster you want to onboard. You need `system:masters` privileges.

For consistency, we will refer to those two clusters by their names from now on.

You need to have the `kubeconfig` files for both the `greenhouse` and the `bootstrap` cluster at hand. The `kubeconfig` file for the `greenhouse` cluster can be downloaded via the Greenhouse dashboard:

_Organization_ > _Clusters_ > _Access Greenhouse cluster_.

### Onboard

For accessing the `bootstrap` cluster, the `greenhousectl` will expect your default Kubernetes [`kubeconfig` file](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/) and [`context`](https://kubernetes.io/docs/reference/kubectl/generated/kubectl_config/kubectl_config_use-context/) to be set to `bootstrap`. This can be achieved by passing the `--kubeconfig` flag or by setting the `KUBECONFIG` env var.

The location of the `kubeconfig` file to the `greenhouse` cluster is passed via the `--greenhouse-kubeconfig` flag.

```commandline
greenhousectl cluster bootstrap --kubeconfig=<path/to/bootstrap-kubeconfig-file> --greenhouse-kubeconfig <path/to/greenhouse-kubeconfig-file> --org <greenhouse-organization-name> --cluster-name <name>
```

Since Greenhouse generates URLs which contain the cluster name, we highly recommend to choose a **short** cluster name.
In particular for <span style="color:red">Gardener Clusters</span> setting a short name is mandatory, because Gardener has very long cluster names, e.g. `garden-greenhouse--monitoring-external`.

A typical output when you run the command looks like

```commandline
2024-02-01T09:34:55.522+0100	INFO	setup	Loaded kubeconfig	{"context": "default", "host": "https://api.greenhouse.tld"}
2024-02-01T09:34:55.523+0100	INFO	setup	Loaded client kubeconfig	{"host": "https://api.remote.tld"}
2024-02-01T09:34:56.579+0100	INFO	setup	Bootstraping cluster	{"clusterName": "monitoring", "orgName": "ccloud"}
2024-02-01T09:34:56.639+0100	INFO	setup	created namespace	{"name": "ccloud"}
2024-02-01T09:34:56.696+0100	INFO	setup	created serviceAccount	{"name": "greenhouse"}
2024-02-01T09:34:56.810+0100	INFO	setup	created clusterRoleBinding	{"name": "greenhouse"}
2024-02-01T09:34:57.189+0100	INFO	setup	created clusterSecret	{"name": "monitoring"}
2024-02-01T09:34:58.309+0100	INFO	setup	Bootstraping cluster finished	{"clusterName": "monitoring", "orgName": "ccloud"}
```

### After onboarding

1. List all clusters in your Greenhouse organization:

```
   kubectl --namespace=<greenhouse-organization-name> get clusters
```

2. Show the details of a cluster:

```
   kubectl --namespace=<greenhouse-organization-name> get cluster <name> -o yaml
```

Example:

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: Cluster
metadata:
  creationTimestamp: "2024-02-07T10:23:23Z"
  finalizers:
    - greenhouse.sap/cleanup
  generation: 1
  name: monitoring
  namespace: ccloud
  resourceVersion: "282792586"
  uid: 0db6e464-ec36-459e-8a05-4ad668b57f42
spec:
  accessMode: direct
  maxTokenValidity: 72h
status:
  bearerTokenExpirationTimestamp: "2024-02-09T06:28:57Z"
  kubernetesVersion: v1.27.8
  statusConditions:
    conditions:
      - lastTransitionTime: "2024-02-09T06:28:57Z"
        status: "True"
        type: Ready
```

When the `status.kubernetesVersion` field shows the correct version of the Kubernetes cluster, the cluster was successfully bootstrapped in Greenhouse.
Then `status.conditions` will contain a `Condition` with `type=Ready` and `status="true""`

In the remote cluster, a new namespace is created and contains some resources managed by Greenhouse.
The namespace has the same name as your organization in Greenhouse.

## Troubleshooting

If bootstrapping fails, you can inspect the `Cluster.statusConditions` for more details. The `type=KubeConfigValid` condition may contain hints in the `message` field. Additional insights can be found in the `type=PermissionsVerified` and `type=ManagedResourcesDeployed` conditions, which indicate whether `ServiceAccount` has valid permissions and whether required resources were successfully deployed. These conditions are also visible in the UI on the `Cluster` details view.
Reruning the onboarding command with an updated `kubeConfig` file will fix these issues.
