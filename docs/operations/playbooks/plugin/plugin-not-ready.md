---
title: "PluginNotReady"
linkTitle: "PluginNotReady"
landingSectionIndex: false
weight: 1
description: >
  Playbook for the PluginNotReady Alert
---

## Alert Description

This alert fires when a Plugin has not been ready for more than 15 minutes.

## What does this alert mean?

A Plugin in Greenhouse represents an application or service deployed to a target cluster via Helm. When a Plugin is not ready, it indicates that the deployment or the workload resources are not functioning correctly.

This could be due to:

- Helm chart deployment failures
- Missing or invalid PluginDefinition
- Cluster access issues
- Invalid plugin option values
- Workload resources not becoming ready (pods failing, etc.)
- Dependencies not being met (via waitFor)

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

### Check the Status Conditions

Look at the `status.statusConditions` section in the plugin resource. Pay special attention to:

- **Ready**: The main indicator of plugin health. Set to `false` if cluster access fails, the PluginDefinition is unavailable, or the Helm release is not deployed successfully.
- **HelmReleaseCreated**: Indicates whether the Flux HelmRelease object has been successfully created. Common reasons for `false`:
  - `PluginDefinitionNotAvailable` — the referenced PluginDefinition or ClusterPluginDefinition does not exist
  - `OptionValueResolutionFailed` — option values could not be resolved (e.g. a referenced secret is missing)
  - `PluginOptionValueInvalid` — option values could not be converted to Helm values
  - `FluxHelmReleaseConfigInvalid` — the generated Flux HelmRelease manifest is invalid
  - `ClusterAccessFailed` — the controller cannot access the target cluster; check target [Cluster status](../cluster/cluster-not-ready.md)
- **HelmReleaseDeployed**: Mirrors the Flux HelmRelease `Ready` condition and reflects whether the Helm release has been successfully deployed on the target cluster. Common reasons for `false`:
  - `FluxHelmReleaseStalled` — install/upgrade retries have been exhausted
  - `HelmUninstallFailed` — uninstalling a previous Helm release revision failed
- **ExposedServicesSynced**: Indicates whether the list of exposed services is up to date with the services defined in the deployed Helm chart.

### Check Underlying Flux Resources

Since Greenhouse uses Flux as the default deployment mechanism, you can inspect the Flux HelmRelease resource belonging to a Plugin:

```bash
# Get the HelmRelease in the organization namespace
kubectl get helmrelease <plugin-name> -n <namespace> -o yaml

# Describe the HelmRelease for detailed status
kubectl describe helmrelease <plugin-name> -n <namespace>
```

## Additional Resources

- [Greenhouse Plugin Documentation](../../../reference/components/plugin.md)
- [Plugin Configuration Guide](../../../user-guides/plugin/configure.md)
- [Flux HelmRelease Documentation](https://fluxcd.io/flux/components/helm/helmreleases/)
