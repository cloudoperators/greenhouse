---
title: "Operational Processes"
weight: 3
---

## Operational Processes in Greenhouse

Greenhouse provides a couple of predefined operational processes.

Operational processes facilitated via Greenhouse heavily rely on the `Ownership` principle. It is used to route operational tasks to [Support Groups](./../../core-concepts/teams#support-groups).

Examples for these operational tasks are:

- Alert routing based on metrics
- Lifecycle management of k8s Clusters
- Security posture and vulnerability patch management
- Secret rotation and management

## Labels Used

Greenhouse focuses on `labels` in three different places:

- On resources (e.g. PluginPresets, Clusters but also k8s Deployments, Pods, etc.)
- On metrics exposed by those resources
- On Prometheus alerts based on metrics

The following `labels` are used by Greenhouse automation:

| label                                 | description                                                      | used on                | used by                |
|---------------------------------------|------------------------------------------------------------------|------------------------|------------------------|
| `greenhouse.sap/owned-by`             | Identifies the owning team of a resource                         | Resources, metrics    | Security management, lifecycle management, secret rotation |
| `support_group`     | Specifies the support group responsible for the alert          |  Alerts    | Alert routing   |
| `service` | Groups resources belonging to a service      | Resources, metrics, alerts             | Security management, alert routing            |
| `region`           | Indicates the region an alert is firing in                     | Metrics, alerts   | Alert routing  |
| `severity`         | Indicates the importance or urgency of an alert                  | Alerts         | Alert routing         |
| Cluster          | Specifies the cluster a metric is exposed on     | Metrics, alerts        | Alert routing       |

## Alert Routing

Monitoring and alert routing is achieved through a combination of Plugins running on the remote Clusters and the Greenhouse central cluster.

All alerts processed with Greenhouse need the `support_group` label that can be extracted from the `greenhouse.sap/owned-by`.

With the [Alerts Plugin](https://github.com/cloudoperators/greenhouse-extensions/tree/main/alerts) a holistic alerts dashboard is integrated to the Greenhouse UI. This dashboard is prefiltered on the support group a user is member of. It directly displays alerts by `region` and `severity`. Also `service` is prominently displayed.

It is good practice to also route alerts by `support_group` and/or `severity` to specific Alertmanager receivers (e.g. Slack channels).

## Lifecycle management of k8s Clusters

All Cluster related alerts, including version expiration and other maintenance tasks are routed to the owning `support_group` of the Cluster.

## Security Management

Security posture and vulnerability management is achieved through the [heureka Plugin](https://github.com/cloudoperators/heureka). It scans for security violations in running k8s `containers` and displays these by `support_group` and `service`.

## Secret Management

With secret management Greenhouse wants to have alerts on expiring Secrets in need of rotation. These alerts will be routed to the respective `support_groups`. See [roadmap item](https://github.com/cloudoperators/greenhouse/issues/1211) for further information.
