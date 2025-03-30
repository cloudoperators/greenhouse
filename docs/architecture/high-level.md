---
title: "High-level architecture"
weight: 1
---

This section provides a high-level overview of the Greenhouse concepts.

## Greenhouse components

Conceptually, the Greenhouse platform consists of 2 types of Kubernetes cluster fulfilling specific purposes.

1) **Central cluster**

    The central cluster accommodates the core components of the Greenhouse platform,
    providing the holistic API and dashboard to let users manage their entire cloud infrastructure from a single control point.
    Users of the Greenhouse cloud operations platform, depending on their roles, can perform tasks such as managing Organizations,
    configuring and using Plugins, monitoring resources, developing custom PluginDefinitions, and conducting audits to ensure compliance with standards.
    Greenhouse's flexibility allows users to efficiently manage cloud infrastructure and applications while tailoring their experience to organizational needs.
    The configuration and metadata is persisted in Kubernetes custom resource definitions (CRDs), acted upon by the Greenhouse operator and managed in the customer cluster.

2) **Customer cluster**

    Managing and operating Kubernetes clusters can be challenging due to the complexity of tasks related to orchestration, scaling, and ensuring high availability in containerized environments.  
    By onboarding their Kubernetes clusters into Greenhouse, users can centralize cluster management, streamlining tasks like resource monitoring and access control.
    This integration enhances visibility, simplifies operational tasks, and ensures better compliance, enabling users to efficiently manage and optimize their Kubernetes infrastructure.
    While the central cluster contains the user configuration and metadata, all workloads of user-selected Plugins are run in the customer cluster and managed by Greenhouse.

A simplified architecture of the Greenhouse platform is illustrated below.

```mermaid
---
title: Greenhouse Platform Simplified Architecture
---
flowchart TD
  
  enduser(End-user)

  idp["Identity Provider <br> (via OIDC)"]

  subgraph "Greenhouse Central Cluster"
    operator[Greenhouse Operator]
    dashboard[Greenhouse Dashboard]
    idproxy[IDProxy]
    k8s[Kubernetes API]
    organization[Organizations]
    pluginDefinitions[PluginDefinitions]
    subgraph org ["Organization Namespace"]
      pluginPresets[PluginPresets]
      plugins[Plugins]
      teams[Teams]
      teamRoles[TeamRoles]
      teamRoleBindings[TeamRoleBindings]
      clusters[Clusters]
      clusterKubeConfig[ClusterKubeConfig]

      teams ~~~ teamRoles
      teamRoleBindings ~~~ clusters
      pluginPresets ~~~ plugins
    end
  end


  subgraph "Customer Cluster"
    helmRelease[Helm Releases]
    rbac[RBAC]
  end

  organization -. specifies .- idp
  idproxy -- AuthN & AuthZ -->idp

  enduser <--> dashboard
  enduser -- manages Greenhouse resources --> k8s
  operator -- watches & manages --> k8s
  dashboard -- watches & manages Greenhouse resources --> k8s

  k8s <--> organization
  k8s <--> pluginDefinitions
  k8s <--> org

  org -- manages --> helmRelease
  org -- manages --> rbac

```
