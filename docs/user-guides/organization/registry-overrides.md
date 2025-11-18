---
title: "Registry Overrides"
linkTitle: "Registry Overrides"
weight: 3
description: >
  Configure registry mirrors for Helm charts and container images in your organization.
---

## Overview

Greenhouse supports overriding registry sources for both Helm charts and container images. This allows pulling resources from alternative registries instead of their original sources.

Registry overrides operate at two levels:

- **Helm chart repositories**: Configured per Catalog
- **Container image registries**: Configured per Organization

## Helm Chart Registry Overrides

Helm chart repositories can be overridden by configuring the `repository` field in the `overrides` array of a Catalog resource.

### Configuration

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: Catalog
metadata:
  name: my-catalog
  namespace: my-organization
spec:
  sources:
    - repository: https://github.com/cloudoperators/greenhouse-extensions
      resources:
        - kube-monitoring/plugindefinition.yaml
        - perses/plugindefinition.yaml
      ref:
        branch: main
      overrides:
        - name: kube-monitoring
          repository: oci://my-registry.example.com/charts/kube-monitoring
        - name: perses
          alias: perses-mirror
          repository: oci://my-registry.example.com/charts/perses
```

Fields:
- `name`: Name of the PluginDefinition to override (required)
- `repository`: The alternative Helm chart repository URL (optional)
- `alias`: The alternative name of the PluginDefinition (optional)

### Example

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: Catalog
metadata:
  name: production-catalog
  namespace: production-org
spec:
  sources:
    - repository: https://github.com/cloudoperators/greenhouse-extensions
      resources:
        - "*/plugindefinition.yaml"
      ref:
        branch: main
      overrides:
        - name: kube-monitoring
          repository: oci://internal-registry.company.com/greenhouse/kube-monitoring
        - name: alerts
          repository: oci://internal-registry.company.com/greenhouse/alerts
        - name: dashboards
          repository: oci://internal-registry.company.com/greenhouse/dashboards
```

## Container Image Registry Overrides

Container image registries are overridden using a ConfigMap referenced by the Organization resource. This applies to all plugins deployed within the organization.

### Configuration

Create a ConfigMap with a `containerRegistryConfig` key:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: registry-mirrors
  namespace: my-organization
data:
  containerRegistryConfig: |
    registryMirrors:
      ghcr.io:
        baseDomain: "mirror.company.com"
        subPath: "ghcr-mirror"
      docker.io:
        baseDomain: "mirror.company.com"
        subPath: "dockerhub-mirror"
      quay.io:
        baseDomain: "mirror.company.com"
        subPath: "quay-mirror"
      registry.k8s.io:
        baseDomain: "mirror.company.com"
        subPath: "k8s-mirror"
```

Reference the ConfigMap in the Organization:

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: Organization
metadata:
  name: my-organization
spec:
  description: My Organization with Registry Mirrors
  configMapRef: registry-mirrors
```

Fields:
- `baseDomain`: The hostname of the mirror registry (required)
- `subPath`: The path within the mirror where images are stored (required)

### Image Transformation

When deploying plugins, image references are automatically rewritten to use the configured mirror registries.

**Example: GitHub Container Registry**

Original image:
```
ghcr.io/cloudoperators/greenhouse:v1.0.0
```

Mirror configuration:
```yaml
registryMirrors:
  ghcr.io:
    baseDomain: "mirror.company.com"
    subPath: "ghcr-mirror"
```

Transformed image:
```
mirror.company.com/ghcr-mirror/cloudoperators/greenhouse:v1.0.0
```

**Example: Docker Hub official images**

Original image:
```
nginx:latest
```

Mirror configuration:
```yaml
registryMirrors:
  docker.io:
    baseDomain: "mirror.company.com"
    subPath: "dockerhub-mirror"
```

Transformed image:
```
mirror.company.com/dockerhub-mirror/library/nginx:latest
```

Note: Docker Hub official images are automatically prefixed with `library/`.

**Example: Docker Hub user repositories**

Original image:
```
bitnami/postgresql:15
```

Mirror configuration:
```yaml
registryMirrors:
  docker.io:
    baseDomain: "mirror.company.com"
    subPath: "dockerhub-mirror"
```

Transformed image:
```
mirror.company.com/dockerhub-mirror/bitnami/postgresql:15
```

### Complete Example

```yaml
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-organization-config
  namespace: my-org
data:
  containerRegistryConfig: |
    registryMirrors:
      ghcr.io:
        baseDomain: "registry.internal.company.com"
        subPath: "external/ghcr"
      docker.io:
        baseDomain: "registry.internal.company.com"
        subPath: "external/dockerhub"
      quay.io:
        baseDomain: "registry.internal.company.com"
        subPath: "external/quay"
      registry.k8s.io:
        baseDomain: "registry.internal.company.com"
        subPath: "external/kubernetes"
      my-public-registry.com:
        baseDomain: "registry.internal.company.com"
        subPath: "external/custom"

---
apiVersion: greenhouse.sap/v1alpha1
kind: Organization
metadata:
  name: my-org
spec:
  description: My Organization with Internal Registry Mirrors
  configMapRef: my-organization-config
```
