---
title: "Overview"
weight: 1
---

## What is Greenhouse?

![](../greenhouse.svg) 

Greenhouse is a cloud operations platform designed to streamline and simplify the management of a large-scale, distributed infrastructure.  

It offers a unified interface for organizations to manage various operational aspects efficiently and transparently and operate their cloud infrastructure in compliance with industry standards.  
The platform addresses common challenges such as the fragmentation of tools, visibility of application-specific permission concepts and the management of organizational groups.
It also emphasizes the harmonization and standardization of authorization concepts to enhance security and scalability.
With its operator-friendly dashboard, features and extensive automation capabilities, Greenhouse empowers organizations to optimize their cloud operations, reduce manual efforts, and achieve greater operational efficiency.

## Value Propositions

1. **Holistic dashboard** <br>
   Unified dashboard for infrastructure, alert, security, compliance, and organizational management. ([Juno](https://github.com/cloudoperators/juno))
2. **Organization management** <br>
   Greenhouse allows to manage organizational groups as Teams. Teams can be provided with fine-grained access control to resources and tools. (e.g. Github Repositories, Kubernetes RBAC, etc.)
3. **Automation** <br>
   Greenhouse allows to configure tools and access control in a declarative way, that is auto-deployed across a fleet of Kubernetes clusters.
4. **Security & Compliance** <br>
   With [Heureka](https://github.com/cloudoperators/heureka), Greenhouse integrates a Security Posture Management tool that focuses on remediation of security issues (vulnerabilities, security events, policy violations), while ensuring compliance and auditability.
5. **Extensibility** <br>
   Greenhouse provides a plugin system that provides a curated [catalog of plugins](https://github.com/cloudoperators/greenhouse-extensions/) with sane defaults. Furthermore, it is possible to extend the platform with self-developed plugins.

## Roadmap

The [Roadmap Kanban board](https://github.com/orgs/cloudoperators/projects/1) provides an overview of ongoing and planned efforts.

## Architecture & Design

The [Greenhouse design and architecture document](../../architecture/product_design) describes the various use-cases and user stories.
