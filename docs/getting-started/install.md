---
title: "Installation"
weight: 3
---

This section provides a step-by-step guide to install Greenhouse on a Gardener shoot cluster.

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

## Installing the Authorization Webhook

The Greenhouse Authorization Webhook is a separate Helm chart (`charts/authz/`) that deploys the authorization server responsible for support-group-scoped access control on Greenhouse resources. It must be reachable from the Kubernetes API server's seed so the API server can forward SubjectAccessReview requests to it.

See [Authorization Webhook](./../operations/authorization-webhook) for an explanation of how it works.

### Requirements

- [cert-manager](https://cert-manager.io/) deployed in the cluster — required for TLS certificate management
- The webhook endpoint must be reachable from the Kubernetes API server seed (see Gardener section below)

### Install

```bash
helm upgrade --install greenhouse-authz oci://ghcr.io/cloudoperators/greenhouse/charts/authz \
  --namespace greenhouse \
  --create-namespace
```

This deploys the authz server with TLS certs managed by cert-manager. For all available values see [`charts/authz/README.md`](https://github.com/cloudoperators/greenhouse/blob/main/charts/authz/README.md).

### Gardener Clusters

On Gardener shoot clusters the API server runs on the seed, so the webhook ClusterIP is not directly reachable. Use [shoot-dns-service](https://gardener.cloud/docs/extensions/others/gardener-extension-shoot-dns-service/) and [reversed VPN tunnel](https://gardener.cloud/contribute/gardener/reversed-vpn-tunnel/) to bridge the network gap:

1. **Enable HA VPN** on the shoot — makes ClusterIPs reachable from the seed.
2. **Pick a static ClusterIP** from the shoot's service CIDR (`spec.networking.services` in the shoot resource).
3. **Configure the chart** with the static IP, a DNS name, and a DNSEntry annotation so shoot-dns-service registers the record:

    ```yaml
    service:
      clusterIP: "100.104.1.10"   # static IP from shoot service CIDR

    tls:
      extraDNSNames:
        - "greenhouse-authz.your-shoot.example.tld"

    dnsEntry:
      enabled: true
      dnsName: "greenhouse-authz.your-shoot.example.tld"
      annotations:
        dns.gardener.cloud/class: garden
    ```

4. **Register the webhook** with the API server by configuring it as an authorization webhook, pointing to the DNS name above with the matching TLS CA bundle.
