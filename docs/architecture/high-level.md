---
title: "Architecture"
weight: 1
---

## Greenhouse components

```mermaid
---
title: Greenhouse High-Level Architecture
---

flowchart LR

  kubectl[kubectl]
  dashboard[Greenhouse Dashboard]

  api[Greenhouse API endpoint]

  subgraph "Greenhouse Central Cluster"
    direction TB
    operator[Greenhouse Operators]
  end

  subgraph "Remote Cluster"
    direction TB
    rbac[Kubernetes RBAC]
    helmRelease[Helm Releases]
  end

  kubectl --> api
  dashboard --> api
  api --> operator
  operator --> rbac
  operator --> helmRelease
```

From a high-level perspective, the Greenhouse platform consists of these main components:

1. **Greenhouse Central Cluster**: The API Server of the central Greenhouse cluster serves as the API endpoint that acts as the primary interface for users to interact with the Greenhouse platform. The API consists of the Greenhouse Custom Resource Definitions (CRDs). Interactions with the API are possible both via the Greenhouse dashboard and the `kubectl` command line tool. The Greenhouse operators run inside this central cluster.
2. **Remote Clusters**: The are the clusters that are onboarded into Greenhouse, so that the Greenhouse operators can manage the lifecycle of Greenhouse resources in these clusters.

## Greenhouse API

```mermaid
---
title: Greenhouse API
---

flowchart TD

api[Greenhouse API]
organization[Organization]
team[Team]
cluster[Cluster]
clusterKubeConfig[ClusterKubeConfig]
teamRole[TeamRole]
teamRoleBinding[TeamRoleBinding]
pluginDefinition[PluginDefinition]
pluginPreset[PluginPreset]
plugin[Plugin]

api --- organization
api --- team
api --- cluster
api --- clusterKubeConfig
api --- teamRole
api --- teamRoleBinding
api --- pluginDefinition
api --- pluginPreset
api --- plugin
api --- ...
```

The Greenhouse API is the core of the Greenhouse, providing a familiar interface to interact with the platform. It is deliberately not exposing all the Kubernetes APIs to the users, but uses RBAC to limit the available resources to the ones that are relevant for the use of Greenhouse. For example it is not permitted to run arbitrary workload resources inside of the Greenhouse central clusters.

## Greenhouse Clusters

When talking about clusters in the context of Greenhouse, we are referring to two different types of clusters:

1) **Central cluster**

    This is the cluster where all of the Greenhouse components are running. It is the central point of access for users to interact with the platform. The dashboard, Kubernetes API and the operator are all running in this cluster. It is possible to share this central cluster between mutliple organizations with an isolation on Kubernetes namespace level. This multi-tenancy is part of the reason that the access to the Kubernetes API is limited to the users, so that they can only access those resources required to interact with Greenhouse.

    Users of the Greenhouse cloud operations platform, depending on their roles, can perform tasks such as managing Organizations,
    configuring and using Plugins, monitoring resources, developing custom PluginDefinitions, and conducting audits to ensure compliance with standards.
    Greenhouse's flexibility allows users to efficiently manage cloud infrastructure and applications while tailoring their experience to organizational needs.
    The configuration and metadata is persisted in Kubernetes custom resource definitions (CRDs), acted upon by the Greenhouse operator and managed in the customer cluster.

2) **Remote cluster**

    When referring to a Remote cluster or Customer cluster we are talking about the clusters that are onboarded into Greenhouse. Onboarding means that a valid KubeConfig is provided so that the Greenhouse operator can access the cluster and manage the resources in it.
    Managing and operating Kubernetes clusters can be challenging due to the complexity of tasks related to orchestration, scaling, and ensuring high availability in containerized environments.  
    By onboarding their Kubernetes clusters into Greenhouse, users can centralize cluster management, streamlining tasks like resource monitoring and access control.
    This integration enhances visibility, simplifies operational tasks, and ensures better compliance, enabling users to efficiently manage and optimize their Kubernetes infrastructure.
    While the central cluster contains the user configuration and metadata, all workloads of user-selected Plugins are run in the customer cluster and managed by Greenhouse.

## Organizations & Authentication

The Greenhouse platform is designed to support multiple organizations, each with its own set of users and permissions. Each organization can have multiple teams, and each team can have its own set of roles and permissions. The Greenhouse API provides a way to manage these organizations, teams, and roles.
The Organization is a cluster-scoped resource for which a namespace with the same name will be created. The members of the organization only have access to the resources in this namespace. The Organization CR specifies the IdP that is used to authenticate the users of the organization. While Greenhose will automatically provision a set of RBAC roles, it is possible for the organization to create additional roles and bind users to them.
In order to make the Kubernetes API available for multiple organizations, Greenhouse provides an idproxy build on-top of [dex](https://dexidp.io/), which allows to handle different identity providers (IdP) when authenticating users against the Kubernetes API.

```mermaid
---
title: Greenhouse Platform Simplified Architecture
---
flowchart TB
  
  enduser(End-user)
  kubectl[kubectl]
  dashboard[Greenhouse Dashboard]

  
  subgraph "Greenhouse Central Cluster"
    k8s[Kubernetes API]
    organization[Organization CR]
    subgraph greenhouse ["Greenhouse Namespace"]
    idproxy[IDProxy]

    end
    subgraph org ["Organization Namespace"]
      team[Team CRs]
      rbac[Roles & RoleBindings]
    end
  end
  idp["Identity Provider <br> (via OIDC)"]


  organization -. specifies .- idp
  team -- maps to IdP Group --> idp
  idproxy -- AuthN & AuthZ -->idp

  enduser <--> dashboard
  enduser -- manages Greenhouse resources --> kubectl
  --> k8s
  dashboard -- watches & manages Greenhouse resources --> k8s

  k8s --> idproxy

```

## Remote Cluster management with Plugins & Team RBAC

Greenhouse provides a way to manage access, observability and compliance of a Kubernetes cluster through the use of Plugins and TeameRoleBindings. Plugins are used to deploy and manage workloads (e.g. Prometheus, Open Telemetry, Cert-Manager, etc.) in the remote clusters, while TeamRoleBindings are used to manage access to the remote clusters through Kubernetes RBAC.

Plugins are a key feature of Greenhouse, allowing users to extend the dashboard with custom functionality and to provide best practices for deploying observability, operations and security tools in the remote clusters. PluginDefinitions define the Helm chart and optional default values. With a PluginPreset, users can configure Plugins for a set of clusters or with a Plugin for one specific cluster.

Access in the remote clusters is managed via TeamRole and TeamRoleBinding resources. TeamRoles define the permissions, similar to RBAC Roles & ClusterRoles. The Greenhouse operator will create the necessary Kubernetes RBAC resources in the remote clusters based on the TeamRoleBinding. The TeamRoleBinding combines the TeamRole with a Team and defines the target Clusters and Namespaces.

```mermaid
---
title: Greenhouse Plugin & Team RBAC
---
flowchart TB

  subgraph "Greenhouse Central Cluster"
    direction TB
    pluginDefinition[PluginDefinition]
    pluginPreset[PluginPreset]
    plugin[Plugin]
    cluster[Cluster]
    team[Team]
    teamRole[TeamRole]
    teamRoleBinding[TeamRoleBinding]
  end

  subgraph "Remote Cluster"
    direction TB
    helmRelease[Helm Release]
    rbac[Kubernetes RBAC]
  end

  pluginDefinition --> pluginPreset
  pluginPreset --> plugin
  cluster --> plugin
  plugin --> helmRelease
  team --> teamRoleBinding
  teamRole --> teamRoleBinding
  cluster --> teamRoleBinding
  teamRoleBinding --> rbac
```
