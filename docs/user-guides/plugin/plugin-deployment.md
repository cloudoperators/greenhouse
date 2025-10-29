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
  clusterName: <name of the remote cluster >
  disabled: false
  displayName: <any human readable name>
  pluginDefinition: <pluginDefinition name>
  releaseNamespace: <namespace> # namespace in remote cluster where the plugin is deployed
  releaseName: <helm release name> # name of the helm release that will be created
  optionValues:
    - name: <from the plugin options>
      value: <from the plugin options>
    - ...
```

### Exposed services and ingresses

Plugins deploying Helm Charts into remote clusters support exposing their services and ingresses in two ways:

#### Service exposure via service-proxy

Services can be exposed through Greenhouse's central service-proxy by adding this annotation:

```yaml
annotations:
  greenhouse.sap/expose: "true"
```

For services with multiple ports, you can specify which port to expose:

```yaml
annotations:
  greenhouse.sap/expose: "true"
  greenhouse.sap/exposed-named-port: "https"  # optional, defaults to first port
```

#### Direct ingress exposure

Ingresses can be exposed directly using their external URLs:

```yaml
annotations:
  greenhouse.sap/expose: "true"
  greenhouse.sap/exposed-host: "api.example.com"  # optional, for multi-host ingresses
```

Both types of exposures appear in the Plugin's `status.exposedServices` with different types: `service` or `ingress`.

## Deploying a Plugin

Create the Plugin resource via the command:

```bash
kubectl --namespace=<organization name> create -f plugin.yaml
```

## After deployment

1. Check with `kubectl --namespace=<organization name> get plugin` has been properly created. When all components of the plugin are successfully created, the plugin should show the state **configured**.

2. Check in the remote cluster that all plugin resources are created in the organization namespace.

### URLs for exposed services and ingresses

After deploying the plugin to a remote cluster, the ExposedServices section in Plugin's status provides an overview of the exposed resources. It maps URLs to both services and ingresses found in the manifest.

#### Service-proxy URLs (for services)

- Services exposed through service-proxy use the pattern: `https://$cluster--$hash.$organization.$basedomain`
- The `$hash` is computed from `service--$namespace`

#### Direct ingress URLs (for ingresses)

- Ingresses are exposed using their actual hostnames: `https://api.example.com` or `http://internal.service.com`
- Protocol (http/https) is automatically detected from the ingress TLS configuration
- The host is taken from `greenhouse.sap/exposed-host` annotation or defaults to the first host rule

Both types are listed together in `status.exposedServices` with their respective types for easy identification.

## Next Steps

- [Plugin reference](./../../../reference/components/plugin)
- [PluginPreset reference](./../../../reference/components/pluginpreset)
- [PluginDefinition reference](./../../../reference/components/plugindefinition)
