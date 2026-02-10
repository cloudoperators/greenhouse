---
title: "IDProxyErrorsHigh"
linkTitle: "IDProxyErrorsHigh"
landingSectionIndex: false
weight: 3
description: >
  Playbook for the IDProxyErrorsHigh Alert
---

## Alert Description

This alert fires when more than 10% of IDProxy operations result in errors for 15 minutes.

## What does this alert mean?

The IDProxy handles authentication and identity proxying for Greenhouse. High error rates indicate authentication or identity management issues that prevent users from accessing resources.

This could be due to:

- Issues with the identity provider (IdP) integration
- OIDC/OAuth configuration problems
- Network connectivity to the IdP
- Invalid or expired tokens
- Misconfigured callback URLs or client credentials
- Resource constraints on the IDProxy pod

## Diagnosis

### Check IDProxy Metrics

Access the Prometheus instance monitoring your Greenhouse cluster and query the IDProxy request metrics using the following PromQL queries:

```promql
# Total HTTP requests by status code
http_requests_total{service="greenhouse-idproxy"}

# Successful requests (2xx)
http_requests_total{service="greenhouse-idproxy",status=~"2.."}

# Error requests (4xx and 5xx)
http_requests_total{service="greenhouse-idproxy",status=~"[45].."}

# Error rate
rate(http_requests_total{service="greenhouse-idproxy",status=~"[45].."}[5m]) / rate(http_requests_total{service="greenhouse-idproxy"}[5m])
```

Analyze the distribution of HTTP status codes to understand what types of errors are occurring.

### Check IDProxy Logs

Review IDProxy logs for detailed error messages:

```bash
kubectl logs -n greenhouse -l app.kubernetes.io/name=idproxy --tail=500 | grep -i error
```

Look for:

- Authentication failures
- Token validation errors
- IdP connection issues
- OIDC/OAuth errors
- Callback URL mismatches

### Check Identity Provider Status

Verify the identity provider is accessible and responding:

```bash
# Check Organization configuration
kubectl get organization <org-name> -o jsonpath='{.spec.authentication}'
```

Test connectivity to the IdP endpoints if accessible.

### Check IDProxy Configuration

Verify the IDProxy configuration in the Organization resource:

```bash
kubectl get organization <org-name> -o yaml
```

Check:

- OIDC issuer URL is correct
- Client ID and client secret are configured
- Redirect URIs are properly set

### Check IDProxy Pod Resource Usage

Verify the IDProxy pod has sufficient resources:

```bash
kubectl top pod -n greenhouse -l app.kubernetes.io/name=idproxy

kubectl describe pod -n greenhouse -l app.kubernetes.io/name=idproxy
```

### Check for Certificate Issues

If using HTTPS for IdP communication, verify certificates are valid:

```bash
kubectl logs -n greenhouse -l app.kubernetes.io/name=idproxy --tail=500 | grep -i "certificate\|tls\|x509"
```

## Additional Resources

- [Greenhouse Architecture](../../../architecture/components.md)
- [Authentication Configuration](../../../user-guides/organization/authentication.md)
