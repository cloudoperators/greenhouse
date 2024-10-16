---
title: "Clusters"
weight: 3
---

## What are Clusters?

In the context of Greenhouse a Cluster represents a Kubernetes cluster which is managed by Greenhouse. Managed in this context means that Greenhouse can take care of RBAC permissions and deploying operational tooling (e.g. logging, monitoring, ingress etc.).
The Greenhouse dashboard provides an overview of all onboarded clusters. Throughout Greenhouse the reference to a Cluster is used to target it for configuration and deployments.

## Cluster access

During the [initial onboarding](../../user-guides/cluster/onboarding.md) of a cluster, Greenhouse will create a ServiceAccount inside the onboarded cluster. Greenhouse uses this ServiceAccount to deploy resources such as Helm releases and RBAC into the cluster. The token for this ServiceAccount is rotated automatically by Greenhouse.

## Cluster registry (coming soon)

Once a Cluster is onboarded to Greenhouse a KubeConfig is generated for the cluster based on the OIDC configuration of the organization. In order to make it convenient to use these KubeConfigs and to manage multiple context locally there will be a CLI provided by Greenhouse.
