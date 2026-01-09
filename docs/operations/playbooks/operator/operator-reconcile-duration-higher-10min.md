---
title: "OperatorReconcileDurationHigher10Min"
linkTitle: "OperatorReconcileDurationHigher10Min"
landingSectionIndex: false
weight: 2
description: >
  Playbook for the OperatorReconcileDurationHigher10Min Alert
---

## Alert Description

**Severity:** Warning  
**Alert Name:** GreenhouseOperatorReconcileDurationHigher10Min

This alert fires when the average reconciliation duration exceeds 10 minutes for a controller for 15 minutes.

**Alert Message:**

```
Greenhouse Operator reconciliation takes longer than ({{ $value | humanizeDuration }}) while reconciling {{ $labels.controller }}
```

## What does this alert mean?

Controllers should reconcile resources quickly. When reconciliation takes longer than 10 minutes on average, it indicates performance issues that can lead to delays in applying configuration changes and resource state updates.

This could be due to:

- High number of resources being managed
- Slow external API calls (e.g., to remote clusters, SCIM APIs)
- Resource contention or controller pod being throttled
- Inefficient reconciliation logic
- Large resource objects or complex computations

## Diagnosis

### Identify the Affected Controller

The alert label `controller` identifies which controller has slow reconciliations.

### Check Controller Metrics

View the current error rate. Either on the Prometheus instance monitoring your Greenhouse controller or directly in cluster:

```bash
# Port-forward to the metrics service
kubectl port-forward -n greenhouse svc/greenhouse-controller-manager-metrics-service 8080:8443

# Query the metrics (in another terminal)
curl -k https://localhost:8080/metrics | grep "controller_runtime_reconcile_time_seconds.*controller=\"<controller-name>\""
```

### Check Controller Logs for Slow Operations

Review the controller logs for slow operations:

```bash
kubectl logs -n greenhouse -l app=greenhouse --tail=1000 | grep "controller=\"<controller-name>\""
```

Look for:

- Long-running operations
- Timeouts or retries
- External API call latencies
- Large number of resources being processed

### Check Number of Managed Resources

Count how many resources the controller is managing:

```bash
kubectl get <resource-type> --all-namespaces --no-headers | wc -l
```

Replace `<resource-type>` with the appropriate resource the controller is managing.

### Check Controller Resource Usage

Verify the controller pod has sufficient resources:

```bash
kubectl top pod -n greenhouse -l app=greenhouse

kubectl describe pod -n greenhouse -l app=greenhouse | grep -A 5 "Limits:\|Requests:"
```

### Check for Resource Throttling

Check if the controller pod is being CPU throttled:

```bash
kubectl describe pod -n greenhouse -l app=greenhouse | grep -i throttl
```

### Check External System Latency

If the controller interacts with external systems (remote clusters, SCIM, etc.), verify their responsiveness:

```bash
# For cluster controller - check if remote clusters are accessible
kubectl get clusters --all-namespaces -o json | jq -r '.items[] | select(.status.statusConditions.conditions[]? | select(.type=="Ready" and .status!="True")) | "\(.metadata.namespace)/\(.metadata.name)"'

# For organization controller - check SCIM connectivity
kubectl get organizations -o json | jq -r '.items[] | select(.status.statusConditions.conditions[]? | select(.type=="SCIMAPIAvailable" and .status!="True")) | .metadata.name'
```

## Additional Resources

- [Greenhouse Architecture](../../../architecture/components.md)
- [Controller Runtime Metrics](https://book.kubebuilder.io/reference/metrics-reference.html)
