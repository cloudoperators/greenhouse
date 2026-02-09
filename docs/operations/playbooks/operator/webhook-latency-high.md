---
title: "WebhookLatencyHigh"
linkTitle: "WebhookLatencyHigh"
landingSectionIndex: false
weight: 4
description: >
  Playbook for the WebhookLatencyHigh Alert
---

## Alert Description

This alert fires when the 90th percentile latency of a Greenhouse webhook exceeds 200ms for 15 minutes.

## What does this alert mean?

Webhooks are admission controllers that validate or mutate resources before they are persisted to etcd. High webhook latency can slow down all API requests for the resources the webhook handles, affecting user operations and controller reconciliations.

This could be due to:

- Complex validation or mutation logic
- External API calls from the webhook (e.g., checking clusters, teams)
- Resource constraints on the webhook pod
- High rate of requests to the webhook
- Network latency within the cluster

## Diagnosis

### Identify the Affected Webhook and Resource

The alert label `webhook` identifies which webhook has high latency. The webhook path indicates the resource type:

- `/mutate-greenhouse-sap-v1alpha1-plugin` → Plugin resource
- `/validate-greenhouse-sap-v1alpha1-plugin` → Plugin resource
- `/mutate-greenhouse-sap-v1alpha1-cluster` → Cluster resource
- `/validate-greenhouse-sap-v1alpha1-cluster` → Cluster resource
- And similar patterns for other resources

Extract the resource type from the webhook path (e.g., `Plugin`, `Cluster`, `Organization`) to use in log filtering.

### Check Webhook Metrics

Access the Prometheus instance monitoring your Greenhouse cluster and query the webhook latency metrics using the following PromQL queries:

```promql
# Webhook latency distribution
controller_runtime_webhook_latency_seconds{webhook="<webhook-path>"}

# 90th percentile latency
histogram_quantile(0.90, rate(controller_runtime_webhook_latency_seconds_bucket{webhook="<webhook-path>"}[5m]))
```

Replace `<webhook-path>` with the actual webhook path from the alert.

### Check Webhook Request Rate

High request rates can contribute to latency. Query Prometheus:

```promql
# Request rate
rate(controller_runtime_webhook_requests_total{webhook="<webhook-path>"}[5m])
```

### Check Webhook Logs

Review webhook logs for slow operations or errors. Use the resource type extracted from the webhook path:

```bash
kubectl logs -n greenhouse -l app=greenhouse,app.kubernetes.io/component=webhook --tail=500 | grep '"kind":"<Resource>"'
```

For example, for the plugin webhook:

```bash
kubectl logs -n greenhouse -l app=greenhouse,app.kubernetes.io/component=webhook --tail=500 | grep '"kind":"Plugin"'
```

Look for:

- Long-running validation or mutation operations
- External API call timeouts
- Error messages
- Repeated webhook calls for the same resources

### Check Webhook Pod Resource Usage

Verify the webhook pod has sufficient resources:

```bash
kubectl top pod -n greenhouse -l app=greenhouse,app.kubernetes.io/component=webhook

kubectl describe pod -n greenhouse -l app=greenhouse,app.kubernetes.io/component=webhook
```

### Check for Resource Contention

Check if the webhook pod is being throttled:

```bash
kubectl describe pod -n greenhouse -l app=greenhouse,app.kubernetes.io/component=webhook | grep -i throttl
```

## Additional Resources

- [Greenhouse Architecture](../../../architecture/components.md)
- [Kubernetes Admission Webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/)
- [Controller Runtime Metrics](https://book.kubebuilder.io/reference/metrics-reference.html)
