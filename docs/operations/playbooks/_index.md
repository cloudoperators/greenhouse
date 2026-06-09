---
title: "Playbooks"
linkTitle: "Playbooks"
landingSectionIndex: false
weight: 2
description: >
  Playbooks for the alerts produced by PrometheusRules deployed with the
  Greenhouse manager chart, grouped by which team is paged.
---

<!-- This page is auto-generated from charts/manager/alerts/. Do not edit by hand — run `make generate-alerts-doc` to regenerate. -->

This page is auto-generated from `charts/manager/alerts/`. Do not edit by hand — run `make generate-alerts-doc` to regenerate.

## Greenhouse admin team

Alerts that page the Greenhouse platform team (`greenhouse-admin`). These cover Greenhouse itself.

- [`GreenhouseOperatorReconcileErrorsHigh`](https://cloudoperators.github.io/greenhouse/docs/operations/playbooks/operator/operator-reconcile-errors-high/)  
  Errors while reconciling &lcub;&lcub;$labels.controller}}
- [`GreenhouseOperatorReconcileDurationHigher10Min`](https://cloudoperators.github.io/greenhouse/docs/operations/playbooks/operator/operator-reconcile-duration-higher-10min/)  
  Reconcile duration higher than 10m while reconciling &lcub;&lcub; $labels.controller }}
- [`GreenhouseOperatorWorkqueueNotDrained`](https://cloudoperators.github.io/greenhouse/docs/operations/playbooks/operator/operator-workqueue-not-drained/)  
  Greenhouse Operator controller - &lcub;&lcub; $labels.name }}'s backlog is not being drained.
- [`GreenhouseWebhookLatencyHigh`](https://cloudoperators.github.io/greenhouse/docs/operations/playbooks/operator/webhook-latency-high/)  
  Greenhouse Operator webhook - &lcub;&lcub; $labels.webhook }}'s latency is high.
- [`GreenhouseWebhookErrorsHigh`](https://cloudoperators.github.io/greenhouse/docs/operations/playbooks/operator/webhook-errors-high/)  
  Errors while reconciling &lcub;&lcub; $labels.webhook }}
- [`GreenhouseProxyRequestErrorsHigh`](https://cloudoperators.github.io/greenhouse/docs/operations/playbooks/proxy/proxy-request-errors-high/)  
  HTTP 5xx errors high for proxy &lcub;&lcub;$labels.service}}
- [`GreenhouseProxyRequestDurationHigh`](https://cloudoperators.github.io/greenhouse/docs/operations/playbooks/proxy/proxy-request-duration-high/)  
  Greenhouse proxy service - &lcub;&lcub; $labels.service }}s latency is high.
- [`GreenhouseIDProxyErrorsHigh`](https://cloudoperators.github.io/greenhouse/docs/operations/playbooks/proxy/idproxy-errors-high/)  
  Greenhouse id-proxy service - HTTP 5xx errors are high.

## Organization admin team

Alerts that page a tenant organization's admin team (`<org>-admin`). These cover tenant-organization-level resources.

- [`GreenhouseOrganizationNotReady`](https://cloudoperators.github.io/greenhouse/docs/operations/playbooks/organization/organization-not-ready/)  
  Greenhouse Organization is not ready
- [`GreenhouseSCIMAccessNotReady`](https://cloudoperators.github.io/greenhouse/docs/operations/playbooks/organization/scim-access-not-ready/)  
  Greenhouse SCIM Access is not ready
- [`GreenhouseResourceOwnedByLabelMissing`](https://cloudoperators.github.io/greenhouse/docs/operations/playbooks/resource/resource-owned-by-label-missing/)  
  The greenhouse.sap/owned-by label is missing on resource
- [`GreenhouseTeamMembershipCountDrop`](https://cloudoperators.github.io/greenhouse/docs/operations/playbooks/team/team-membership-count-drop/)  
  Team members count drop detected

## Support groups

Alerts that page the team owning the affected resource via its `owned_by` label. These cover tenant-managed resources — Plugins, Clusters, Catalogs, TeamRoleBindings.

- `GreenhouseCatalogNotReady`  
  Catalog not ready for over 15 minutes
- [`GreenhouseClusterNotReady`](https://cloudoperators.github.io/greenhouse/docs/operations/playbooks/cluster/cluster-not-ready/)  
  Cluster not ready
- [`GreenhouseClusterTokenExpiry`](https://cloudoperators.github.io/greenhouse/docs/operations/playbooks/cluster/cluster-token-expiry/)  
  The kubeconfig token is not refreshed.
- [`GreenhouseClusterKubernetesVersionOutOfMaintenance`](https://cloudoperators.github.io/greenhouse/docs/operations/playbooks/cluster/cluster-kubernetes-version-out-of-maintenance/)  
  Kubernetes version out of maintenance
- [`GreenhousePluginNotReady`](https://cloudoperators.github.io/greenhouse/docs/operations/playbooks/plugin/plugin-not-ready/)  
  Plugin not ready for over 15 minutes
- `GreenhousePluginPresetNotReconciled`  
  PluginPreset cannot reconcile all Plugins for over 15 minutes
- [`GreenhousePluginConstantlyFailing`](https://cloudoperators.github.io/greenhouse/docs/operations/playbooks/plugin/plugin-constantly-failing/)  
  Plugin reconciliation is constantly failing
- [`GreenhouseTeamRoleBindingNotReady`](https://cloudoperators.github.io/greenhouse/docs/operations/playbooks/team/team-role-binding-not-ready/)  
  TeamRoleBinding not ready for over 15 minutes
