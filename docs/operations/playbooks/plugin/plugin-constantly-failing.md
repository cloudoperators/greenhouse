---
title: "PluginConstantlyFailing"
linkTitle: "PluginConstantlyFailing"
landingSectionIndex: false
weight: 2
description: >
  Playbook for the PluginConstantlyFailing Alert
---


<!-- TODO: understand difference to PluginNotReady -->
## Alert Description

**Severity:** Warning  
**Alert Name:** GreenhousePluginConstantlyFailing

This alert fires when a Plugin reconciliation is constantly failing for 15 minutes.

**Alert Message:**

```
Plugin {{ $labels.plugin }} in organization {{ $labels.namespace }} keeps failing with reason: {{ $labels.reason }}
```

## What does this alert mean?

This alert indicates that the Greenhouse controller is repeatedly failing to reconcile the Plugin resource. Unlike a one-time failure, this suggests a persistent issue that prevents the Plugin from being properly managed.

Common causes include:

- Invalid plugin option values that cannot be resolved
- Missing PluginDefinition reference
- Persistent Helm chart rendering or installation errors
- Invalid or missing secrets referenced in option values
- Cluster access issues that don't resolve
- Configuration conflicts

## Diagnosis

### Get the Plugin Resource

Retrieve the plugin resource to view its current status:

```bash
kubectl get plugin <plugin-name> -n <namespace> -o yaml
```

Or use kubectl describe for a more readable output:

```bash
kubectl describe plugin <plugin-name> -n <namespace>
```

### Check the Status Conditions and Reasons

Look at the `status.statusConditions` section in the plugin resource. Pay special attention to:

- **Ready**: The main indicator of plugin health
- **ClusterAccessReady**: Indicates if Greenhouse can access the target cluster. If `false` check target [Cluster status](../cluster/cluster-not-ready.md).
- **HelmReconcileFailed**: Shows if Helm reconciliation failed
- **HelmDriftDetected**: Indicates drift between desired and actual state
- **HelmChartTestSucceeded**: Shows if Helm chart tests passed
- **WaitingForDependencies**: Indicates if waiting for other plugins
- **RetriesExhausted**: Shows if all retry attempts have been exhausted

Common failure reasons to look for:

- **PluginDefinitionNotFound**: The referenced PluginDefinition does not exist
- **OptionValueResolutionFailed**: Option values could not be resolved
- **PluginOptionValueInvalid**: Option values could not be converted to Helm values
- **HelmUninstallFailed**: The Helm release could not be uninstalled

### Check for Specific Issues

#### PluginDefinitionNotFound

```bash
# Check if the PluginDefinition exists
kubectl get plugindefinition <plugin-definition-name> -n <namespace>

# Or check ClusterPluginDefinition
kubectl get clusterpluginefinition <plugin-definition-name> -n greenhouse
```

#### OptionValueResolutionFailed

```bash
# Check if referenced secrets exist (ValueFrom.Secret)
kubectl get secrets -n <namespace>

# Verify option values in the plugin spec
kubectl get plugin <plugin-name> -n <namespace> -o jsonpath='{.spec.optionValues}'
```

### Check Controller Logs

Review the Greenhouse controller logs for detailed reconciliation errors:

```bash
kubectl logs -n greenhouse -l app=greenhouse --tail=200 | grep "<plugin-name>" | grep "error"
```

### Check Underlying Flux Resources

Check the Flux HelmRelease for additional error details:

```bash
kubectl get helmrelease <plugin-name> -n <namespace> -o yaml

kubectl describe helmrelease <plugin-name> -n <namespace>
```

## Additional Resources

- [Greenhouse Plugin Documentation](../../../reference/components/plugin.md)
- [Plugin Configuration Guide](../../../user-guides/plugin/configure.md)
