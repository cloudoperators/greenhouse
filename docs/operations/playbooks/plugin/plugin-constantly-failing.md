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

This alert fires when a Plugin reconciliation is constantly failing for 15 minutes.

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

- **Ready**: The main indicator of plugin health. Set to `false` if cluster access fails, the PluginDefinition is unavailable, or the Helm release is not deployed successfully.
- **HelmReleaseCreated**: Indicates whether the Flux HelmRelease object has been successfully created. If `false`, check for PluginDefinition or option value issues.
- **HelmReleaseDeployed**: Mirrors the Flux HelmRelease `Ready` condition and reflects whether the Helm release has been successfully deployed on the target cluster.
- **ExposedServicesSynced**: Indicates whether the list of exposed services is up to date with the services defined in the deployed Helm chart.

Common failure reasons to look for:

- **PluginDefinitionNotAvailable**: The referenced PluginDefinition or ClusterPluginDefinition does not exist
- **PluginDefinitionNotBackedByHelmChart**: The PluginDefinition does not define a Helm chart
- **OptionValueResolutionFailed**: Option values could not be resolved (e.g. a referenced secret is missing)
- **PluginOptionValueInvalid**: Option values could not be converted to Helm values
- **FluxHelmReleaseConfigInvalid**: The generated Flux HelmRelease manifest is invalid and could not be applied
- **FluxHelmReleaseStalled**: The Flux HelmRelease is stalled, typically because install/upgrade retries have been exhausted
- **ClusterAccessFailed**: The controller cannot access the target cluster — check target [Cluster status](../cluster/cluster-not-ready.md)
- **HelmUninstallFailed**: The Helm release could not be uninstalled (relevant during Plugin deletion)

### Check for Specific Issues

#### PluginDefinitionNotAvailable

```bash
# Check if the PluginDefinition exists
kubectl get plugindefinition <plugin-definition-name> -n <namespace>

# Or check ClusterPluginDefinition
kubectl get clusterpluginefinition <plugin-definition-name> -n greenhouse # requires permissions on the greenhouse namespace
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
