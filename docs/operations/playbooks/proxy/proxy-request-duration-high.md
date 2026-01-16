---
title: "ProxyRequestDurationHigh"
linkTitle: "ProxyRequestDurationHigh"
landingSectionIndex: false
weight: 2
description: >
  Playbook for the ProxyRequestDurationHigh Alert
---

## Alert Description

**Severity:** Warning  
**Alert Name:** GreenhouseProxyRequestDurationHigh

This alert fires when the 90th percentile latency of a proxy service exceeds 500ms for 15 minutes.

**Alert Message:**

```
The 90th percentile latency of Greenhouse proxy - {{ $labels.service }} is {{ $value | humanizeDuration }}
```

## What does this alert mean?

High latency in proxy services degrades user experience and can cause timeouts. When response times consistently exceed 500ms, it indicates performance issues that need investigation.

This could be due to:

- Slow backend services
- Network latency to remote clusters or services
- Resource constraints on the proxy pod
- High traffic volume overwhelming the proxy
- Inefficient routing or processing logic
- DNS resolution delays

## Diagnosis

### Identify the Affected Proxy Service

The alert label `proxy` identifies which proxy service has high latency:

- `greenhouse-service-proxy` - Proxies requests to services in remote clusters. Is deployed to the `<org-name>` namespace, not `greenhouse`!
- `greenhouse-cors-proxy` - Handles CORS for frontend applications
- `greenhouse-idproxy` - Handles authentication and identity proxying

The placeholder `<proxy-name>` from here on is the above without the `greenhouse-` prefix. E.g. `idproxy`.

### Check Proxy Metrics

View the current request duration metrics. Either on the Prometheus instance monitoring your Greenhouse or directly in cluster:

```bash
# Port-forward to the metrics service of the affected proxy
kubectl port-forward -n greenhouse svc/greenhouse-controller-manager-metrics-service 8080:8080

# Query the metrics (in another terminal)
curl -k http://localhost:8080/metrics | grep "request_duration_seconds.*service=\"<proxy>\""
```

### Check Proxy Logs

> Important! the `service-proxy` is deployed to the `<org-name>` namespace, not `greenhouse`!

Review proxy logs for slow requests:

```bash
kubectl logs -n greenhouse -l app.kubernetes.io/name=<proxy-name> --tail=500
```

Look for patterns indicating slow responses or timeout warnings.

### Check Backend Service Response Times

For service-proxy, verify that backend services in remote clusters are responding quickly:

```bash
# List plugins with exposed services
kubectl get plugins --all-namespaces -l greenhouse.sap/plugin-exposed-services=true

# Check if any plugins are not ready
kubectl get plugins --all-namespaces -l greenhouse.sap/plugin-exposed-services=true -o json | jq -r '.items[] | select(.status.statusConditions.conditions[]? | select(.type=="Ready" and .status!="True")) | "\(.metadata.namespace)/\(.metadata.name)"'
```

### Check Network Latency

Test network latency to remote clusters:

```bash
# For each cluster, check connectivity
kubectl get clusters --all-namespaces -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}'
```

### Check Proxy Pod Resource Usage

Verify the proxy pod has sufficient resources and is not throttled:

```bash
kubectl top pod -n greenhouse -l app.kubernetes.io/name=<proxy-name>

kubectl describe pod -n greenhouse -l app.kubernetes.io/name=<proxy-name>
```

## Additional Resources

- [Greenhouse Architecture](../../../architecture/components.md)
- [Service Proxy Documentation](../../../user-guides/plugin/plugin-deployment.md#urls-for-exposed-services-and-ingresses)
