---
title: "Plugin development"
linkTitle: "development"
description: >
  Develop a new Greenhouse Plugin against a local development environment.
---

## Introduction

Let's illustrate how to leverage Greenhouse _Plugins_ to deploy a Helm Chart into a remote cluster within the local development environment.

This guide will walk you through the process of spinning up the local development environment, creating a new Greenhouse _PluginDefinition_ and deploying it to a local kind cluster.

At the end of the guide you will have spun up the local development environment, onboarded a Cluster, created a _PluginDefinition_ and deployed it as a _Plugin_ to the onboarded Cluster.

> [!NOTE]
> This guide assumes you already have a working Helm chart and will not cover how to create a Helm Chart from scratch. For more information on how to create a Helm Chart, please refer to the [Helm documentation](https://helm.sh/docs/topics/charts/).

## Requirements

- [git](https://git-scm.com/downloads)
- [Docker](https://docs.docker.com/engine/install/)
- [kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
- [kubectl](https://kubernetes.io/docs/reference/kubectl/)
- [Greenhousectl](https://github.com/cloudoperators/greenhouse/releases)

### Starting the local develoment environment

Follow the [Local Development](./../../contribute/local-dev.md) documentation to spin up the local Greenhouse development environment.

This will provide you with a local Greenhouse instance running, filled with some example Greenhouse resources and the Greenhouse UI running on `http://localhost:3000`.

### Onboarding a Cluster

In this step we will create and onboard a new Cluster to the local Greenhouse instance. The local cluster will be created utilizing kind.

In order to onboard a kind cluster follow the [onboarding a cluster](https://github.com/cloudoperators/greenhouse-extensions/tree/main/dev-env#onboard-kind-cluster) secton of the dev-env README.

After onboarding the cluster you should see the new Cluster in the Greenhouse UI.

## Prepare Helm Chart

For this example we will use the [bitnami nginx](https://artifacthub.io/packages/helm/bitnami/nginx) Helm Chart.
The packaged chart can be downloaded with:

```bash
helm pull oci://registry-1.docker.io/bitnamicharts/nginx --destination ./
```

After unpacking the `*.tgz` file there is a folder named `nginx` containing the Helm Chart.

## Generating a PluginDefinition from a Helm Chart

Using the files of the Helm Chart we will create a new Greenhouse _PluginDefinition_ using the `greenhousectl` CLI.

```bash
greenhousectl plugin generate ./nginx ./nginx-plugin
```

This will create a new folder `nginx-plugin` containing the _PluginDefinition_ in a nested structure.

## Modifying the PluginDefinition

The generated _PluginDefinition_ contains a `plugindefinition.yaml` file which defines the _PluginDefinition_. But there are still a few steps required to make it work.

### Specify the Helm Chart repository

After generating the _PluginDefinition_ the `.spec.helmChart.repository` field in the `plugindefinition.yaml` contains a TODO comment. This field should be set to the repository where the Helm Chart is stored.
For the bitnami nginx Helm Chart this would be `oci://registry-1.docker.io/bitnamicharts`.

#### Specify a local Helm Chart

Instead of using a chart repository hosted on a server, you can also utilize a local chart in a `*.tgz` format and place it under `dev-env/helm-charts`. You then only need to run `docker compose up` again to mount the chart to the respective folder and source it as following: 
```yaml
 helmChart:
        name: helm-charts/{filename}.tgz
        repository: 
```


### Specify the UI application

A _PluginDefinition_ may specify a UI application that will be integrated into the Greenhouse UI. This tutorial does not cover how to create a UI application. Therefore the section `.spec.uiApplication` in the `plugindefinition.yaml` should be removed.

> [!INFORMATION]
> The [UI](https://github.com/cloudoperators/greenhouse-extensions/tree/main/dev-env#ui) section of the dev-env readme provides a brief introduction developing a frontend application for Greenhouse.

### Modify the Options

The _PluginDefinition_ contains a section `.spec.options` which defines options that can be set when deploying the _Plugin_ to a Cluster. These options have been generated based on the Helm Chart values.yaml file. You can modify the options to fit your needs.

In general the options are defined as follows:

```yaml
options:
  - default: true
    value: abcd123
    description: automountServiceAccountToken
    name: automountServiceAccountToken
    required: false
    type: ""
```

_default_ specifies if the option should provide a default value. If this is set to true, the value specified will be used as the default value. The _Plugin_ can still provide a different value for this option.
_description_ provides a description for the option.
_name_ specifies the Helm Chart value name, as it is used within the Chart's template files.
_required_ specifies if the option is required. This will be used by the Greenhouse Controllers to determine if a _Plugin_ is valid.
_type_ specifies the type of the option. This can be any of `[string, secret, bool, int, list, map]`. This will be used by the Greenhouse Controllers to validate the provided value.

For this tutorial we will remove all options.

## Deploying a Plugin to the Kind Cluster

After modifying the _PluginDefinition_ we can deploy it to the local Greenhouse cluster and create a _Plugin_ that will deploy the `nginx` to the onboarded cluster.

```bash
  kubectl --kubeconfig=./envtest/kubeconfig apply -f ./nginx-plugin/nginx/17.3.2/plugindefinition.yaml
  plugindefinition.greenhouse.sap/nginx-17.3.2 created
```

The _Plugin_ can be configured using the Greenhouse UI running on `http://localhost:3000`.
Follow the following steps to deploy a _Plugin_ for the created _PluginDefinition_ into the onboarded kind cluster:

1. Navigate on the Greenhouse UI to `Organization>Plugins`.
2. Click on the `Add Plugin` button.
3. Select the `nginx-17.3.2` _PluginDefinition_.
4. Click on the `Configure Plugin` button.
5. Select the cluster in the drop-down.
6. Click on the `Create Plugin` button.

After the _Plugin_ has been created the Plugin Overview page will show the status of the plugin.

Theh deployment can also be verified in the onboarded cluster by checking the pods in the `test-org` namespace of the kind cluster.

```bash
kind export kubeconfig --name remote-cluster
Set kubectl context to "kind-remote-cluster"

k get pods -n test-org
NAME                                    READY   STATUS    RESTARTS   AGE
nginx-remote-cluster-758bf47c77-pz72l   1/1     Running   0          2m11s
```
