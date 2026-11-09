---
title: "PluginPresetNotReconciled"
linkTitle: "PluginPresetNotReconciled"
landingSectionIndex: false
weight: 3
description: >
  Playbook for the PluginPresetNotReconciled Alert
---

## Alert Description

This alert fires when a PluginPreset has not been able to create or update all of its Plugins for more than 15 minutes.

## What does this alert mean?

A PluginPreset creates one Plugin per Cluster that matches its `.spec.clusterSelector`. The alert fires while the `PluginFailed` condition of the PluginPreset is not `False`, which means the Plugin for at least one Cluster could not be created or updated.

For the affected Clusters an existing Plugin keeps its previous spec and a new Plugin is not created.

The alert does not cover Plugins that were created but are not ready. Those are reported by the [PluginNotReady](../plugin-not-ready) alert.

## Diagnosis

### Get the PluginPreset Resource

```bash
kubectl describe pluginpreset <pluginpreset-name> -n <organization-namespace>
```

### Check the PluginFailed Condition

Look at the `PluginFailed` condition in `.status.statusConditions`. Its message lists every failing Plugin as `<pluginpreset-name>-<cluster-name>: <error>`.

When the `PluginDefinitionNotFound` condition is `True`, the PluginDefinition or ClusterPluginDefinition in `.spec.plugin.pluginDefinitionRef` does not exist and the Plugin of every Cluster fails with a `not found` error.

Otherwise only the listed Clusters are affected. Common causes are:

- an option value could not be resolved, for example a `valueFrom.ref` to a Plugin or PluginPreset that does not exist, or a CEL expression that fails for this Cluster
- `expression` or `valueFrom.ref` is used while the matching feature flag is disabled. The flags are read when the Greenhouse controller-manager starts, so restart it after enabling one
- the Plugin was rejected by the admission webhook, for example an option value that does not match the PluginDefinition

See [Troubleshooting Plugin integrations](../../../../user-guides/plugin/plugin-integration#troubleshooting) for the errors of `valueFrom.ref` and expressions.

### Check the Plugin

When the error points to a single Cluster, compare the Plugin of that Cluster with the PluginPreset:

```bash
kubectl get plugin <pluginpreset-name>-<cluster-name> -n <organization-namespace> -o yaml
```

## Additional Resources

- [PluginPreset Documentation](../../../../reference/components/pluginpreset)
- [Feature Flags for PluginPresets](../../../../reference/components/pluginpreset#feature-flags)
