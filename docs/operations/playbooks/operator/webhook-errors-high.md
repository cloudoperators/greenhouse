---
title: "WebhookErrorsHigh"
linkTitle: "WebhookErrorsHigh"
landingSectionIndex: false
weight: 5
description: >
  Playbook for the WebhookErrorsHigh Alert
---

## Alert Description

**Severity:** Warning  
**Alert Name:** GreenhouseWebhookErrorsHigh

This alert fires when more than 10% of webhook operations fail for a webhook for 15 minutes.

**Alert Message:**

```
{{ $value | humanizePercentage }} of webhook operations failed for {{ $labels.webhook }} webhook
```

## What does this alert mean?

Webhooks validate or mutate resources before they are persisted. When a webhook's error rate exceeds 10%, it indicates that many API requests for the affected resources are being rejected or failing.

This could be due to:

- Invalid resource configurations being submitted
- Webhook validation logic rejecting malformed resources
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

View the current webhook error rate. Either on the Prometheus instance monitoring your Greenhouse controller or directly in cluster:

```bash
# Port-forward to the metrics service
kubectl port-forward -n greenhouse svc/greenhouse-controller-manager-metrics-service 8080:8080

# Query the webhook metrics (in another terminal)
curl -k http://localhost:8080/metrics | grep "controller_runtime_webhook_requests_total{webhook=\"<webhook-path>\"}"
```

Look at both successful (code="200") and failed (code!="200") requests.

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
