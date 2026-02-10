---
title: "ProxyRequestErrorsHigh"
linkTitle: "ProxyRequestErrorsHigh"
landingSectionIndex: false
weight: 1
description: >
  Playbook for the ProxyRequestErrorsHigh Alert
---

## Alert Description

This alert fires when more than 10% of HTTP requests result in 4xx (excluding 401/403) or 5xx errors for a proxy service for 15 minutes.

## What does this alert mean?

Greenhouse proxy services (like service-proxy, cors-proxy, idproxy) handle HTTP traffic for various purposes. High error rates indicate that requests are failing, which affects user experience and functionality.

This could be due to:

- Backend services being unavailable or unhealthy
- Misconfigured routing or proxy rules
- Authentication/authorization issues (if 401/403 are included)
- Network connectivity problems to backend services
- Resource exhaustion in the proxy pod
- Invalid requests from clients

## Diagnosis

### Identify the Affected Proxy Service

The alert label `proxy` identifies which proxy service has high error rates:

- `greenhouse-service-proxy` - Proxies requests to services in remote clusters. Is deployed to the `<org-name>` namespace, not `greenhouse`!
- `greenhouse-cors-proxy` - Handles CORS for frontend applications
- `greenhouse-idproxy` - Handles authentication and identity proxying

The placeholder `<proxy-name>` from here on is the above without the `greenhouse-` prefix. E.g. `idproxy`.

### Check Proxy Metrics

Access the Prometheus instance monitoring your Greenhouse cluster and query the proxy request metrics using the following PromQL queries:

```promql
# Total HTTP requests by status code
http_requests_total{service="<proxy-name>"}

# Successful requests (2xx)
http_requests_total{service="<proxy-name>",status=~"2.."}

# Client errors (4xx, excluding 401/403)
http_requests_total{service="<proxy-name>",status=~"4..",status!~"40[13]"}

# Server errors (5xx)
http_requests_total{service="<proxy-name>",status=~"5.."}

# Error rate
(rate(http_requests_total{service="<proxy-name>",status=~"4..",status!~"40[13]"}[5m]) + rate(http_requests_total{service="<proxy-name>",status=~"5.."}[5m])) / rate(http_requests_total{service="<proxy-name>"}[5m])
```

Replace `<proxy-name>` with the actual proxy service name from the alert (e.g., `greenhouse-service-proxy`, `greenhouse-cors-proxy`, `greenhouse-idproxy`).

### Check Proxy Logs

> Important! the `service-proxy` is deployed to the `<org-name>` namespace, not `greenhouse`!

Review proxy logs for detailed error messages:

```bash
kubectl logs -n greenhouse -l app.kubernetes.io/name=<proxy-name> --tail=500 | grep -i error
```

For service-proxy specifically:

```bash
kubectl logs -n greenhouse -l app.kubernetes.io/name=idproxy --tail=500 | grep -E "error|status.*[45][0-9]{2}"
```

Look for:

- Backend connection failures
- Timeout errors
- Authentication/authorization failures
- Invalid routing or target service issues

### Check Backend Service Health

If the proxy is routing to backend services, verify they are healthy. For service-proxy, check plugins with exposed services:

```bash
kubectl get plugins --all-namespaces -l greenhouse.sap/plugin-exposed-services=true

# Check if any plugins are not ready
kubectl get plugins --all-namespaces -l greenhouse.sap/plugin-exposed-services=true -o json | jq -r '.items[] | select(.status.statusConditions.conditions[]? | select(.type=="Ready" and .status!="True")) | "\(.metadata.namespace)/\(.metadata.name)"'
```

### Check Proxy Pod Resource Usage

Verify the proxy pod has sufficient resources:

```bash
kubectl top pod -n greenhouse -l app=<service-name>

kubectl describe pod -n greenhouse -l app=<service-name>
```

## Additional Resources

- [Greenhouse Architecture](../../../architecture/components.md)
- [Service Proxy Documentation](../../../user-guides/plugin/plugin-deployment.md#urls-for-exposed-services-and-ingresses)
