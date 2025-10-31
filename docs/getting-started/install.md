---
title: "Installation"
weight: 3
---

This section provides a step-by-step guide to install Greenhouse on a Gardner shoot cluster.

## Prerequisites

Before you start the installation, make sure you have the following prerequisites:

- Helm & Kubernetes CLI
- OAuth2/OpenID provider (see Authentik)
- Gardener Shoot Cluster configured to use the OIDC provider
- nginx-ingress deployed in the cluster

## Installation

To install Greenhouse on your Gardener shoot cluster, follow these steps:

1. Create a values file called `values.yaml` with the following content:

    ```yaml
      global:
        dnsDomain: tld.domain # Shoot.spec.dns.domain
        kubeAPISubDomain: myapi # api is already used by Gardener
        oidc:
          enabled: true
          issuer: <issuer-url>
          clientID: <client-ID>
          clientSecret: <top-secret>
        dex:   # DEX configuration for Greenhouse.
          backend: kubernetes
      
      postgresqlng:
        enabled: false # disable, because the dex backend is kubernetes
      
      organization:
        enabled: false # disable, because the greenhouse webhook is not running yet
      
      teams:
        admin:
          mappedIdPGroup: greenhouse-admins

      # gardener specifics
      dashboard:
        ingress:
          annotations:
            dns.gardener.cloud/dnsnames: "*"
            dns.gardener.cloud/ttl: "600"
            dns.gardener.cloud/class: garden
            cert.gardener.cloud/purpose: managed

      idproxy:
        enabled: false # disable because no organization is created yet
        ingress:
          annotations:
            dns.gardener.cloud/dnsnames: "*"
            dns.gardener.cloud/ttl: "600"
            dns.gardener.cloud/class: garden
            cert.gardener.cloud/purpose: managed

      cors-proxy:
        ingress:
          annotations:
            dns.gardener.cloud/dnsnames: "*"
            dns.gardener.cloud/ttl: "600"
            dns.gardener.cloud/class: garden
            cert.gardener.cloud/purpose: managed

      # disable Plugins for the greenhouse organization, PluginDefinitions are missing
      plugins:
        enabled: false

      # disable, Prometheus CRDs are missing
      manager:
        alerts:
          enabled: false
    ```

2. Install the Greenhouse Helm chart:

    ```bash
    helm install greenhouse oci://ghcr.io/cloudoperators/greenhouse/charts/greenhouse --version <greenhouse-release-version> -f values.yaml
    ```

3. Enable Greenhouse OIDC

    Now set `organization.enabled` and `idproxy.enabled` to `true` in the `values.yaml` file and upgrade the Helm release:

    ```bash
    helm upgrade greenhouse oci://ghcr.io/cloudoperators/greenhouse/charts/greenhouse --version <greenhouse-release-version> -f values.yaml
    ```

    This will create the initial Greenhouse Organization and the Greenhouse Admin Team. This Organization will receive the `greenhouse` namespace, which is used to manage the Greenhouse installation and allows to administer other organizations.
    Enabling the idproxy will deploy the idproxy service which handles the OIDC authentication.
