---
title: "Core Concepts"
linkTitle: "Core Concepts"
weight: 1
landingSectionIndex: false
description: >
  Overview of some core concepts of Greenhouse
---

| Feature | Description | API | UI | Comments |
| --- | --- | --- | --- | --- |
| [Organizations](/docs/getting-started/core-concepts/organizations) | Organizations are the top-level entities in Greenhouse. Based on the name of the Organization a namespace is automatically created. This namespace contains all resources that are managed by Greenhouse for the Organization, such as Teams, Clusters, and Plugins. | 🟢 | 🟢 | |
| [Teams](/docs/getting-started/core-concepts/teams.md) | Teams are used to manage access to resources in Greenhouse. | 🟢 | 🟡 | Read-only access to Teams via the UI |
| [Clusters(/docs/getting-started/core-concepts/clusters.md)] | Clusters represent a Kubernetes cluster that is managed by Greenhouse. | 🟡 | 🟡 | Limited editability of Clusters via UI, CLI for KubeConfig registry planned |
| [Plugin Definitions & Plugins](/docs/getting-started/core-concepts/plugins.md) | PluginDefinitions are the extensibility features used to deploy & configure UIs and backend components. | 🟡 | 🟡 | Read-only access via UI, kube native plugin catalog planned |
