---
title: "Cluster onboarding"
linkTitle: "Onboarding"
description: >
  Onboard an existing Kubernetes cluster to Greenhouse.
---

This guides describes how to onboard an existing Kubernetes cluster to your Greenhouse organization.  
If you don't have an organization yet please reach out to the Greenhouse administrators.

While all members of an organization can see existing clusters, their management requires [`org-admin` or `cluster-admin` privileges](./../../getting-started/core-concepts/organizations.md).

```
NOTE: The UI is currently in development. For now this guide describes the onboarding workflow via command line.
```

### Preparation

Download the latest `greenhousectl` binary from [here](https://github.com/cloudoperators/greenhouse/releases).

Onboarding a `Cluster` to Greenhouse will require you to authenticate to two different kubernetes clusters via repsective `kubeconfig` files:

- `greenhouse`: The cluster your Greenhouse installation is running on. You need `organization-admin` or `cluster-admin` priviledges.
- `bootstrap`: The cluster you want to onboard. You need `system:masters` priviledges.

For consistency we will refer to those two clusters by their names from now on.

You need to have the `kubeconfig` files for both the `greenhouse` and the `bootstrap` cluster at hand. The `kubeconfig` file for the `greenhouse` cluster can be downloaded via the Greenhouse dashboard:

_Organization_ > _Clusters_ > _Access greenhouse cluster_.

### Onboard

For accessing the `greenhouse` cluster, the `greenhousectl` will expect your default kubernetes [`kubeconfig` file](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/) and [`context`](https://kubernetes.io/docs/reference/kubectl/generated/kubectl_config/kubectl_config_use-context/) to be set to `greenhouse`.
The easiest way for doing so is passing the `--kubeconfig` (and optionally `--kubecontext`) flag to your `greenhousectl` command.

The location of the `kubeconfig` file to the `bootstrap` cluster is passed via the `--bootstrap-kubeconfig` flag.

```commandline
greenhousectl cluster bootstrap --kubeconfig=<path/to/greenhouse-kubeconfig-file> --bootstrap-kubeconfig <path/to/bootstrap-kubeconfig-file> --org <greenhouse-organization-name> --cluster-name <name>
```

Since Greenhouse generates URLs which contain the cluster name, we highly recommend to choose a **short** cluster name.
In particular for <span style="color:red">Gardener Clusters</span> setting a short name is mandatory, because Gardener has very long cluster names, e.g. `garden-greenhouse--monitoring-external`.

A typical output when you ran the command looks like

```commandline
2024-02-01T09:34:55.522+0100	INFO	setup	Loaded kubeconfig	{"context": "default", "host": "https://api.greenhouse-qa.eu-nl-1.cloud.sap"}
2024-02-01T09:34:55.523+0100	INFO	setup	Loaded client kubeconfig	{"host": "https://api.monitoring.greenhouse.shoot.canary.k8s-hana.ondemand.com"}
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
    greenhouse.sap/cluster
  generation: 1
  name: monitoring
  namespace: ccloud
  resourceVersion: "282792586"
  uid: 0db6e464-ec36-459e-8a05-4ad668b57f42
spec:
  accessMode: direct
status:
  bearerTokenExpirationTimestamp: "2024-02-09T06:28:57Z"
  kubernetesVersion: v1.27.8
  statusConditions:
    conditions:
    - lastTransitionTime: "2024-02-09T06:28:57Z"
      status: "True"
      type: Ready
...
```

When the `status.kubernetesVersion` field shows the correct version of the Kubernetes cluster, the cluster was successfully bootstrapped in Greenhouse.
Then `status.conditions` will contain a `Condition` with `type=ready` and `status="true""` 


In the remote cluster, a new namespace is created and contains some resources managed by Greenhouse.
The namespace has the same name as your organization in Greenhouse.

If the bootstrapping failed, you can delete the Kubernetes cluster from Greenhouse with `kubectl --namespace=<greenhouse-organization-name> delete cluster <name>` and run the bootstrap command again.
