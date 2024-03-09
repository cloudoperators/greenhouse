---
title: "Plugin deployment"
linkTitle: "deploying"
description: >
  Deploy a Greenhouse plugin to an existing Kubernetes cluster.
---

## Before you begin

This guides describes how to configure and deploy a Greenhouse plugin.  

```yaml
    apiVersion: greenhouse.sap/v1alpha1
    kind: PluginConfig
    metadata:
      name: kube-monitoring-martin
      namespace: <organization namespace> # same namespace in remote cluster for resources
    spec:
      clusterName: <name of the remote cluster > # k get cluster 
      disabled: false
      displayName: <any human readable name>
      plugin: <plugin name> # k get plugin
      optionValues:
        - name: <from the plugin options>
          value: <from the plugin options> 
        - ...
```

## Deploy plugin

Create the plugin config resource via the command:
```
   kubectl --namespace=<organization name> create -f plugin_config.yaml
```
## After deployment

1. Check with `kubectl --namespace=<organization name> get pluginconfig` has been properly created. When all components of the plugin are successfully created, the plugonconfig should show the state **configured**.  

2. Check in the remote cluster that all plugin resources are created in the organization namespace.