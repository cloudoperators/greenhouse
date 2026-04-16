---
title: "Clusters"
weight: 2
---

A Cluster represents an external Kubernetes cluster that is managed by Greenhouse. Clusters are onboarded to an Organization and can be targeted by Plugins and PluginPresets to deploy infrastructure components.

## Example Cluster

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: Cluster
metadata:
  name: example-cluster
  namespace: example-organization
  labels:
    metadata.greenhouse.sap/region: europe
    metadata.greenhouse.sap/environment: production
spec:
  accessMode: direct
```

## Working with Clusters

### Setting Metadata Labels

Cluster metadata is stored as Kubernetes resource labels with the `metadata.greenhouse.sap/` prefix. Add or update metadata labels on an existing Cluster using `kubectl`:

```bash
kubectl label cluster example-cluster \
  metadata.greenhouse.sap/region=europe \
  metadata.greenhouse.sap/environment=production \
  --namespace=example-organization
```

### Disabling service-proxy for a Cluster

By default, Greenhouse allows exposing Kubernetes Ingress and Service resources on a remote cluster via [annotations](./../../../user-guides/plugin/plugin-deployment/#urls-for-exposed-services-and-ingresses).
This feature can be disabled for a Cluster by annotating the Cluster resource with `greenhouse.sap/service-proxy-disabled=true`. This will disable the creation of service-proxy endpoints for Service resources. Ingress resources will still be collected and added to the Plugin status.

## Next Steps

- [Cluster Onboarding](./../../../user-guides/cluster/onboarding)
- [Cluster Offboarding](./../../../user-guides/cluster/offboarding)
- [Using Metadata Labels and Expressions](./../../../user-guides/plugin/metadata-expressions)
