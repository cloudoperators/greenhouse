---
title: "WebhookErrorsHigh"
linkTitle: "WebhookErrorsHigh"
landingSectionIndex: false
weight: 5
description: >
  Playbook for the WebhookErrorsHigh Alert
---

## Alert Description

This alert fires when more than 10% of webhook operations fail for a webhook for 15 minutes.

## What does this alert mean?

Webhooks validate or mutate resources before they are persisted. When a webhook's error rate exceeds 10%, it indicates that many API requests for the affected resources are being rejected or failing.

This could be due to:

- Invalid resource configurations being submitted
- External dependencies being unavailable (e.g., clusters, teams, secrets)
- Permission issues in webhook operations
- Bugs in the webhook logic
- Network issues preventing webhook from accessing required resources

## Diagnosis

### Identify the Affected Webhook and Resource

The alert label `webhook` identifies which webhook has high error rates. Extract the resource type from the webhook path (e.g., `Plugin`, `Cluster`, `Organization`) to use in log filtering.

Common webhook paths:

- `/mutate-greenhouse-sap-v1alpha1-plugin` → Plugin resource
- `/validate-greenhouse-sap-v1alpha1-plugin` → Plugin resource
- `/mutate-greenhouse-sap-v1alpha1-cluster` → Cluster resource
- `/validate-greenhouse-sap-v1alpha1-cluster` → Cluster resource

### Check Webhook Metrics

Access the Prometheus instance monitoring your Greenhouse cluster and query the webhook request metrics using the following PromQL queries:

```promql
# Total webhook requests by status code
controller_runtime_webhook_requests_total{webhook="<webhook-path>"}

# Successful requests (200)
controller_runtime_webhook_requests_total{webhook="<webhook-path>",code="200"}

# Failed requests (non-200)
controller_runtime_webhook_requests_total{webhook="<webhook-path>",code!="200"}

# Error rate
rate(controller_runtime_webhook_requests_total{webhook="<webhook-path>",code!="200"}[5m]) / rate(controller_runtime_webhook_requests_total{webhook="<webhook-path>"}[5m])
```

Replace `<webhook-path>` with the actual webhook path from the alert.

### Check Webhook Logs

Review webhook logs for error messages using the resource type:

```bash
kubectl logs -n greenhouse -l app=greenhouse,app.kubernetes.io/component=webhook --tail=500 | grep '"kind":"<Resource>"' | grep -i error
```

For example, for the plugin webhook:

```bash
kubectl logs -n greenhouse -l app=greenhouse,app.kubernetes.io/component=webhook --tail=500 | grep '"kind":"Plugin"' | grep -i error
```

Look for:

- Validation errors indicating why resources are being rejected
- Missing referenced resources (Teams, Secrets, PluginDefinitions, Clusters)
- Permission errors
- Network errors when accessing external systems

### Check Recent Resource Submissions

List recent resources of the affected type to see if there are patterns:

```bash
kubectl get <resource-type> --all-namespaces --sort-by=.metadata.creationTimestamp
```

Check if recently created or updated resources have issues:

```bash
kubectl get <resource-type> --all-namespaces -o json | jq -r '.items[] | select(.status.statusConditions.conditions[]? | select(.type=="Ready" and .status!="True")) | "\(.metadata.namespace)/\(.metadata.name)"'
```

### Check Webhook Pod Resource Usage

Verify the webhook pod has sufficient resources:

```bash
kubectl top pod -n greenhouse -l app=greenhouse,app.kubernetes.io/component=webhook

kubectl describe pod -n greenhouse -l app=greenhouse,app.kubernetes.io/component=webhook
```

## Additional Resources

- [Greenhouse Architecture](../../../architecture/components.md)
- [Kubernetes Admission Webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/)
- [Controller Runtime Metrics](https://book.kubebuilder.io/reference/metrics-reference.html)
