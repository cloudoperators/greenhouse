---
title: "Plugin deployment"
linkTitle: "Plugin Deployment"
weight: 3
description: >
  Deploy a Greenhouse plugin to an existing Kubernetes cluster.
---

## Before you begin

This guides describes how to configure and deploy a Greenhouse plugin.

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: Plugin
metadata:
  name: kube-monitoring-martin
  namespace: <organization namespace> # same namespace in remote cluster for resources
spec:
  clusterName: <name of the remote cluster > # k get cluster
  disabled: false
  displayName: <any human readable name>
  pluginDefinition: <plugin name> # k get plugin
  optionValues:
    - name: <from the plugin options>
      value: <from the plugin options>
    - ...
```

### Exposed services

Plugins deploying Helm Charts into remote clusters support exposed services.

By adding the following label to a service in helm chart it will become accessible from the central greenhouse system via a service proxy:

`greenhouse.sap/expose: "true"`

## Deploying a Plugin

Create the Plugin resource via the command:

```bash
kubectl --namespace=<organization name> create -f plugin.yaml
```

## After deployment

1. Check with `kubectl --namespace=<organization name> get plugin` has been properly created. When all components of the plugin are successfully created, the plugin should show the state **configured**.

2. Check in the remote cluster that all plugin resources are created in the organization namespace.

### URLs for exposed services

After deploying the plugin to a remote cluster, ExposedServices section in Plugin's status provides an overview of the Plugins services that are centrally exposed. It maps the exposed URL to the service found in the manifest.

- The URLs for exposed services are created in the following pattern: `$https://$cluster--$hash.$organisation.$basedomain`. The `$hash` is computed from `service--$namespace`.
- When deploying a plugin to the central cluster, the exposed services won't have their URLs defined, which will be reflected in the Plugin's Status.
