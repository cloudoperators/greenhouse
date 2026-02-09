---
title: "OperatorReconcileErrorsHigh"
linkTitle: "OperatorReconcileErrorsHigh"
landingSectionIndex: false
weight: 1
description: >
  Playbook for the OperatorReconcileErrorsHigh Alert
---

## Alert Description

This alert fires when more than 10% of reconciling operations fail for a controller for 15 minutes.

## What does this alert mean?

The Greenhouse operator uses controllers to manage various resources. When a controller's reconciliation error rate exceeds 10%, it indicates systemic issues preventing the controller from properly managing its resources.

This could be due to:

- API server connectivity issues
- Resource conflicts or invalid resource states
- Missing dependencies or referenced resources
- Permission issues preventing controller operations
- Resource exhaustion (memory, CPU) affecting controller performance
- Bugs in the controller logic

## Diagnosis

### Identify the Affected Controller

The alert label `controller` identifies which controller is failing.

### Check Controller Metrics

View the current error rate. Either on the Prometheus instance monitoring your Greenhouse controller or directly in cluster:

```bash
# Port-forward to the metrics service
kubectl port-forward -n greenhouse svc/greenhouse-controller-manager-metrics-service 8080:8080

# Query the metrics (in another terminal)
curl -k http://localhost:8080/metrics | grep "controller_runtime_reconcile_errors_total{controller=\"<controller-name>\"}"
curl -k http://localhost:8080/metrics | grep "controller_runtime_reconcile_total{controller=\"<controller-name>\"}"
```

### Check Controller Logs

Review the controller logs for specific error messages:

```bash
kubectl logs -n greenhouse -l app=greenhouse --tail=500 | grep "controller=\"<controller-name>\"" | grep "error"
```

Look for patterns in the errors to identify the root cause.

### Check Affected Resources

List resources managed by the failing controller that are not ready:

```bash
kubectl get <resource-type> --all-namespaces -o json | jq -r '.items[] | select(.status.statusConditions.conditions[]? | select(.type=="Ready" and .status!="True")) | "\(.metadata.namespace)/\(.metadata.name)"'
```

Replace `<resource-type>` with the appropriate resource the controller is managing (e.g., `clusters`, `plugins`, `organizations`, `teams`, `teamrolebindings`).

### Check Controller Resource Usage

Verify the controller pod is not resource-constrained:

```bash
kubectl top pod -n greenhouse -l app=greenhouse

kubectl describe pod -n greenhouse -l app=greenhouse
```

### Check API Server Connectivity

Test if the controller can reach the Kubernetes API server:

```bash
kubectl get --raw /healthz
kubectl get --raw /readyz
```

## Additional Resources

- [Greenhouse Architecture](../../../architecture/components.md)
- [Controller Runtime Metrics](https://book.kubebuilder.io/reference/metrics-reference.html)
