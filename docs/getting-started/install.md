---
title: "Installation"
weight: 3
---

This section provides a step-by-step guide to install Greenhouse on your Kubernetes cluster.

## Prerequisites

Before you start the installation, make sure you have the following prerequisites:

- Helm & Kubernetes CLI
- An OIDC provider to authenticate users

## Installation

To install Greenhouse on your Kubernetes cluster, follow these steps:

1. Create a values file called `values.yaml` with the following content:

    ```yaml
    global:
      dnsDomain: <my-greenhouse.test.cloud>
      oidc:
        enabled: true
        issuer: <https://top.secret>
        redirectURL: <https://top.secret/redirect>
        clientID: <topSecret!>
        clientSecret: <topSecret!>

    teams:
      admin:
        mappedIdPGroup: <MyGreenhouseAdmins>

    plugins:
      enabled: false

    organization:
      enabled: false

    idproxy:
      enabled: false
    ```

2. Install the Greenhouse Helm chart:

    ```bash
    helm install greenhouse oci://ghcr.io/cloudoperators/greenhouse/charts/greenhouse --version <greenhouse-release-version> -f values.yaml
    ```

3. Enable Greenhouse OIDC

    Now set `organization.enabled` and `idproxy.enabled` to `true` in the `values.yaml` file and upgrade the helm release:

    ```bash
    helm upgrade greenhouse oci://ghcr.io/cloudoperators/greenhouse/charts/greenhouse --version <greenhouse-release-version> -f values.yaml
    ```

    This will create the Greenhouse Organization and the admin Team used to manage the Greenhouse installation. Also it will deploy the idproxy service to handle the OIDC authentication.
