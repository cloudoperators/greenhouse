---
title: "OperatorWorkqueueNotDrained"
linkTitle: "OperatorWorkqueueNotDrained"
landingSectionIndex: false
weight: 3
description: >
  Playbook for the OperatorWorkqueueNotDrained Alert
---

## Alert Description

**Severity:** Warning  
**Alert Name:** GreenhouseOperatorWorkqueueNotDrained

This alert fires when a controller's workqueue backlog is not getting drained for 15 minutes.

**Alert Message:**

```
The workqueue backlog of Greenhouse Operator controller - {{ $labels.name }} is not getting drained.
```

## What does this alert mean?

Each controller uses a workqueue to process reconciliation requests. When the workqueue depth continues to grow rather than being drained, it indicates that the controller cannot keep up with the incoming reconciliation requests.

This could be due to:

- High rate of resource changes overwhelming the controller
- Slow reconciliation operations (see also [OperatorReconcileDurationHigher10Min](operator-reconcile-duration-higher-10min.md))
- Controller pod being resource-constrained
- Deadlocks or stuck reconciliation loops
- External systems being slow or unavailable

## Diagnosis

### Identify the Affected Controller

The alert label `name` identifies the controller workqueue that is not draining.

### Check Workqueue Metrics

View the current workqueue depth and rates. Either on the Prometheus instance monitoring your Greenhouse controller or directly in cluster:

```bash
# Port-forward to the metrics service
kubectl port-forward -n greenhouse svc/greenhouse-controller-manager-metrics-service 8080:8080

# Query the workqueue metrics (in another terminal)
curl -k http://localhost:8080/metrics | grep "workqueue_depth{name=\"<controller-name>\"}"
curl -k http://localhost:8080/metrics | grep "workqueue_adds_total{name=\"<controller-name>\"}"
curl -k http://localhost:8080/metrics | grep "workqueue_work_duration_seconds.*name=\"<controller-name>\""
```

### Check Controller Logs

Review controller logs to see if reconciliations are processing:

```bash
kubectl logs -n greenhouse -l app=greenhouse --tail=500 | grep "<controller-name>"
```

Look for:

- Repeated reconciliation of the same resources
- Error messages indicating stuck operations
- Long pauses between log entries

### Check Reconciliation Duration

If reconciliations are slow, this may prevent the queue from draining:

```bash
curl -k http://localhost:8080/metrics | grep "controller_runtime_reconcile_time_seconds.*controller=\"<controller-name>\""
```

### Check Controller Resource Usage

Verify the controller has sufficient resources:

```bash
kubectl top pod -n greenhouse -l app=greenhouse

kubectl describe pod -n greenhouse -l app=greenhouse
```

### Check Number of Resources

A high number of resources may be causing excessive reconciliation load:

```bash
kubectl get <resource-type> --all-namespaces --no-headers | wc -l
```

Replace `<resource-type>` with the appropriate resource the controller is managing.

### Check for External System Issues

If the controller depends on external systems, verify they are responsive:

```bash
# Check cluster connectivity
kubectl get clusters --all-namespaces -o json | jq -r '.items[] | select(.status.statusConditions.conditions[]? | select(.type=="Ready" and .status!="True")) | "\(.metadata.namespace)/\(.metadata.name)"'

# Check organization SCIM connectivity
kubectl get organizations -o json | jq -r '.items[] | select(.status.statusConditions.conditions[]? | select(.type=="SCIMAPIAvailable" and .status!="True")) | .metadata.name'
```

## Additional Resources

- [Greenhouse Architecture](../../../architecture/components.md)
- [Controller Runtime Metrics](https://book.kubebuilder.io/reference/metrics-reference.html)
- [Workqueue Metrics](https://book.kubebuilder.io/reference/metrics-reference.html#workqueue-metrics)
