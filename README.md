Greenhouse
==========
[![REUSE status](https://api.reuse.software/badge/github.com/cloudoperators/greenhouse)](https://api.reuse.software/info/github.com/cloudoperators/greenhouse)
<a href="https://github.com/cloudoperators/greenhouse"><img align="left" width="150" height="170" src="./docs/assets/greenhouse.svg"></a>

Greenhouse is an operations platform focussing on providing a set of tools & processes for managing cloud native infrastructure. Among others it provides the building blocks to enable the configuration and deployment of tools and fine-grained access control to a fleet of Kubernetes clusters.

It provides out-of-the-box integration and processes on top of cloud native tools like Prometheus, Perses, Alertmanager, and Thanos. The platform allows to extend the functionality with self-developed plugins.

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

## Community

Greenhouse is a community-driven project. We welcome contributions and feedback. Please see our [Contributing Guidelines](CONTRIBUTING.md) for more information.

## Roadmap

The [Roadmap Kanban board](https://github.com/orgs/cloudoperators/projects/9) provides an overview of ongoing and planned efforts.

## Documentation

User guides, links and references are available in the official [Greenhouse documentation](https://cloudoperators.github.io/greenhouse/).

### Architecture & Design

The [Greenhouse design and architecture document](https://cloudoperators.github.io/greenhouse/docs/architecture/product_design/) describes the various use-cases and user stories.

### API resources

Greenhouse extends Kubernetes using Custom Resource Definitions(CRD).
See the [API resources documentation](https://cloudoperators.github.io/greenhouse/docs/reference/api/) for more details.

## Code of Conduct

We as members, contributors, and leaders pledge to make participation in our community a harassment-free experience for everyone. By participating in this project, you agree to abide by its [Code of Conduct](https://github.com/SAP/.github/blob/main/CODE_OF_CONDUCT.md) at all times.

## Licensing

Copyright 2025 SAP SE or an SAP affiliate company and Greenhouse contributors. Please see our [LICENSE](LICENSE) for copyright and license information. Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/cloudoperators/greenhouse).
