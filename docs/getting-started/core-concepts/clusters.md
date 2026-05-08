---
title: "Clusters"
weight: 3
---

## What are Clusters?

In the context of Greenhouse a Cluster represents a Kubernetes cluster that is onboarded to Greenhouse. Onboarded in this context means that Greenhouse can handle the management of role-based access control ([RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)) and the provisioning of operating tools (e.g. logging, monitoring, ingress etc.).
The Greenhouse dashboard provides an overview of all onboarded clusters. Throughout Greenhouse the reference to a Cluster is used to target it for configuration and deployments.

## Cluster access

During the [initial onboarding](./../../../user-guides/cluster/onboarding) of a cluster, Greenhouse will create a dedicated [ServiceAccount](https://kubernetes.io/docs/concepts/security/service-accounts/) inside the onboarded cluster. This ServiceAccount's token is rotated automatically by Greenhouse.

## Cluster registry (coming soon)

Once a Cluster is onboarded to Greenhouse a ClusterKubeConfig is generated for the Cluster based on the OIDC configuration of the Organization. This enables members of an Organization to access the fleet of onboarded Clusters via the common Identity Provider. on the respective Clusters can be managed via [Greenhouse Team RBAC](./../../../user-guides/team/rbac).

In order to make it convenient to use these ClusterKubeConfigs and to easily switch between multiple context locally there will be a CLI provided by Greenhouse.
