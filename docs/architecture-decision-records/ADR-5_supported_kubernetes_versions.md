# ADR-4 Supported Kubernetes Versions

## Decision Contributors

- Arno Uhlig
- David Gogl
- Ivo Gosemann
- Uwe Mayer

## Status

- Proposed

## Context and Problem Statement

Greenhouse is interacting with remote Kubernetes clusters via [kubernetes/client-go](https://github.com/kubernetes/client-go) and [helm/helm](https://github.com/helm/helm). This includes creating Kubernetes resources and interacting with Helm releases. To ensure a consistent and transparent behavior it is important to define which Kubernetes versions are officially supported by Greenhouse.

This ADR addresses the following concerns:

1. guideline which Kubernetes versions are officially supported
2. how to communicate/ inform Organisation Admins about EOL Kubernetes versions

## Decision Drivers

- Stability:
  - Greenhouse should be stable and reliable for any of the officially supported Kubernetes Versions.

- Transparency:
  - Consumers should be informed about the officially supported Kubernetes Versions.
  - Consumers should be informed about the EOL Date of their Kubernetes Version.

- Compatibility:
  - Greenhouse should be compatible with the Kubernetes Versions officially supported by Helm and the Kubernetes client.

## Decision

The Kubernetes project [supports](https://kubernetes.io/releases/version-skew-policy/#supported-versions) the most recent three minor releases. The [Kubernetes Release Cycle](https://kubernetes.io/releases/release/#the-release-cycle) for a new minor version is roughly every 4 months.

Kubernetes is backward compatible with clients. This means that client-go will work with [many different Kubernetes versions](https://github.com/kubernetes/client-go?tab=readme-ov-file#compatibility-client-go---kubernetes-clusters).

The Helm project officially supports the most recent n-3 Kubernetes minor releases. This means that Helm 3.14.x is supporting Kubernetes 1.29.x, 1.28.x, 1.27.x and 1.26.x. The Helm project follows the release cadence of Kubernetes, with a [minor release every 4 months](https://helm.sh/docs/topics/release_policy/#minor-releases). The Helm minor version release does not follow directly with the Kubernetes version release but with a small offset.

The official release date for Kubernetes 1.30 is 17.04.2024. The corresponding Helm minor release 3.15 has been announced for 08.05.2024.

Greenhouse should support the latest Kubernetes Version soon after the Helm project has released the corresponding minor version release. Since Helm supports the latest n-3 Kubernetes Versions, this allows for a grace period of roughly 4 months for Organisation Admins to upgrade their Clusters to an officially supported Kubernetes version.

With this decision, Greenhouse will follow the version skew policy of the Helm project to support the most recent n-3 Kubernetes minor releases. This both ensures that Organisations can use the latest Kubernetes version soon after it has been released. Also this gives Organisation Admins time to upgrade their Clusters to an officially supported Kubernetes version, if they are running Clusters on a Kubernetes version that has reached EOL.

It must also be clear how Greenhouse interacts with Clusters running on a Kubernetes Version not yet supported, or on a Kubernetes Version that is no longer supported.

Greenhouse should not reconcile Clusters that are running on a newer Kubernetes Version than currently supported by the pulled in dependencies for Kubernetes and Helm. It should be made clear in the Status of the Cluster CRD and in the Plugin CRD, that the Cluster is running an unsupported version. The Greenhouse UI should also visualise that a Cluster is running on a version that is not yet supported. Organisation Admins should also be informed about this situation.

The other case is when a Cluster is running on a Kubernetes Version that is no longer supported by the Helm dependencies. In this case the reconciliation of this Cluster should not be stopped. The UI should however visualise that the Cluster is running on an EOL Kubernetes release. Prior to the EOL Date, Organisation Admins should be informed about the upcoming EOL Date and the implications for their Clusters.

The documentation will show the currently supported Kubernetes releases. The documentation should also describe the reconciliation behavior for clusters running on Kubernetes releases that are not yet supported and those no longer supported.
