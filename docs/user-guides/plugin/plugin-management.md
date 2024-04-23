---
title: "Managing Plugins for multiple clusters"
linkTitle: "plugin-management"
description: >
  Deploy a Greenhouse Plugin with the same configuration into multiple clusters.
---

## Managing Plugins for multiple clusters

This guide describes how to configure and deploy a Greenhouse Plugin with the same configuration into multiple clusters.

The _PluginPreset_ resource is used to create and deploy Plugins with a the identical configuration into multiple clusters. The list of clusters the Plugins will be deployed to is determind by a [LabelSelector](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors).

As a result, whenever a cluster, that matches the _ClusterSelector_ is onboarded or offboarded, the Controller for the PluginPresets will take care of the Plugin Lifecycle. This means creating or deleting the Plugin for the respective cluster.

The same validation applies to the _PluginPreset_ as to the _Plugin_. This includes immutable _PluginDefinition_ and _ReleaseNamespace_ fields, as well as the validation of the _OptionValues_ against the _PluginDefinition_.

In case the _PluginPreset_ is updated all of the Plugin instances that are managed by the _PluginPreset_ will be updated as well. Each Plugin instance that is created from a _PluginPreset_ has a label `greenhouse.sap/pluginpreset: <PluginPreset name>`.

Changes that are done directly on a _Plugin_ which was created from a _PluginPreset_ will be overwritten immediately by the _PluginPreset_ Controller. All changes must be performed on the _PluginPreset_ itself.

## Example _PluginPreset_

```yaml
    apiVersion: greenhouse.sap/v1alpha1
    kind: PluginPreset
    metadata:
      name: kube-monitoring-preset
      namespace: <organization namespace>
    spec:
      plugin: # this embeds the PluginSpec
        displayName: <any human readable name>
        pluginDefinition: <PluginDefinition name> # k get plugindefinition
        releaseNamespace: <namespace> # namespace where the plugin is deployed to on the remote cluster. Will be created if not exists
        optionValues:
          - name: <from the PluginDefinition options>
            value: <from the PluginDefinition options> 
          - ..
      clusterSelector: # LabelSelector for the clusters the Plugin should be deployed to
        matchLabels:
          <label-key>: <label-value>
```
