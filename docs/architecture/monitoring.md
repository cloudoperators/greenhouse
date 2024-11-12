---
title: "Building Observability Architectures"
weight: 3
---

The main terminologies used in this document can be found in [core-concepts](https://cloudoperators.github.io/greenhouse/docs/getting-started/core-concepts).

## Introduction to Observability

Observability in software and infrastructure has become essential for operation the complexity of modern Clouds. The concept centers on understanding the internal states of systems based on the data they produce, enabling teams to:

	•	Detect and troubleshoot issues quickly,
	•	Maintain performance and reliability,
	•	Make data-driven improvements.

## Core Components and Tools

Key pillars of observability are **metrics**, **logs**, and **traces**, each providing unique insights that contribute to a comprehensive view of a system’s health.

- **Metrics:**
	•	Metrics are numerical data points representing system health over time (e.g., CPU usage, memory usage, request latency).
	•	**Prometheus** is a widely used tool for collecting and querying metrics. It uses a time-series database optimized for real-time data, making it ideal for gathering system health data, enabling alerting, and visualizing trends.

- **Logs:**
	•	Logs capture detailed, structured/unstructured records of system events, which are essential for post-incident analysis and troubleshooting.
	•	**OpenSearch** provides a robust, scalable platform for log indexing, search, and analysis, enabling teams to sift through large volumes of logs to identify issues and understand system behavior over time.

- **Traces:**
	•	Traces follow a request’s journey through the system, capturing latency and failures across microservices. Traces are key for understanding dependencies and diagnosing bottlenecks.

**OpenTelemetry** is a vendor-neutral standard for instrumenting applications and collecting metrics, logs and traces. By providing a unified approach, **OpenTelemetry** makes it easier to integrate with backends like Prometheus, OpenSearch, or other APM tools.

## Observability in Greenhouse

Greenhouse provides a suite of Plugins to help customers build observability architectures for their Greenhouse-onboarded Kubernetes clusters. These components are designed to collect metrics and logs with a proven default configuration, enabling customers to monitor, visualize, and alert on system health. The following Plugins are available currently:

- [Kubernetes Monitoring](https://cloudoperators.github.io/greenhouse/docs/reference/catalog/kube-monitoring): Prometheus, to collect metrics from Kubernetes components and provides a default set of alerting rules.
- [Thanos](https://cloudoperators.github.io/greenhouse/docs/reference/catalog/thanos): Thanos, to store and query Prometheus metrics at scale.
- [Plutono](https://cloudoperators.github.io/greenhouse/docs/reference/catalog/plutono): Visualisation tool, to query Prometheus metrics and visualize them in dynamic dashboards.
- [Alerts](https://cloudoperators.github.io/greenhouse/docs/reference/catalog/alerts): Prometheus Alertmanager and Supernova, to manage and visualize alerts from Prometheus.
- [OpenTelemetry](https://cloudoperators.github.io/greenhouse/docs/reference/catalog/opentelemetry): OpenTelemetry Collector, to collect metrics and logs from applications and forward them to backends like Prometheus and OpenSearch.

## Example Architectures
![Monitoring architecture](./monitoring-architecture.png)
