# ADR-6 Plugin Option overrides

## Decision Contributors

- ...

## Status

- Proposed

## Context and Problem Statement

In Greenhouse Plugins are the primary way to extend the functionality of the Operations Platform. Since there are some Plugins that are required in most clusters, such as `CertManager`, there are PluginPresets. These PluginPresets are a way to define a default configuration for a Plugin, which is deployed to all clusters matching the PluginPreset's selector.

The issue now is that there are some cases where the default configuration between two clusters is different, or requires a cluster-specific secret. This is currently not possible with PluginPresets, as they are applied with the same configuration to all clusters matching the selector.

Another issue is setting default values that are valid for all plugins inside of an Organization, or for all plugins for a specific cluster. Currently, this requires setting these values in every Plugin's spec.

Greenhouse should offer a way to override PluginOptionValues for a specific cluster, for all Plugins of a certain PluginDefinition, or for all plugins in an Organization.

## Decision Drivers

- Stability:
  - Overrides should be consistent
  - Overrides should be applied in a deterministic way (most specific last)
  - There should be no conflicts between overrides or constant reconciliation loops
  - Changes to the overrides should be applied to all relevant plugins

- Transparency:
  - End-users should be able to see/understand which overrides are applied to a Plugin

- Compatibility:
  - Overrides should be compatible with existing Plugins
  - Overrides should be compatible with existing PluginPresets

## Decision

We will introduce a new CRD called `PluginOverride`. This CRD will allow users to override PluginOptionValues.
It will be possible to:

- define a ClusterSelector to specify the relevant clusters
- specify PluginDefinitionNames to only apply values to Plugins instantiated from any listed PluginDefinition
- apply the overrides to all Plugins in an Organization

The Clusters relevant for the override should be determined by the ClusterSelector. The ClusterNames are the names of the clusters that should be affected by the override. The IgnoreClusters are the names of the clusters that should not be affected by the override. The LabelSelector is a metav1.LabelSelector that should be used to select the clusters.

```golang
type ClusterSelector struct{
  LabelSelector * metav1.LabelSelector `json:"labelSelector,omitempty"`
  ClusterNames []string `json:"clusterNames,omitempty"`
  IgnoreClusters []string `json:"ignoreClusters,omitempty"`
}
```

This could look like:

```yaml
kind: PluginOverride
name: my-overrides
spec:
  pluginDefinitionNames:
    - my-plugindefinition # if empty applies to all plugins
  clusterSelector: # if empty applies to all clusters
    - matchLabels:
        my-cluster-label: my-cluster-value
  overrides:
    - path: my-option
      value: value-override
```

There is a central override component, which is able to retrieve the list of relevant overrides for a Plugin. This component will be called from the PluginPresetController during reconciliation of the individual Plugins.
Overrides for Plugins not managed by a PluginPreset will be applied by a separate controller.

The PluginPresetController and the PluginOverrideController should watch for changes to relevant PluginOverrides and update the respective PluginSpec

The following events should trigger the reconciliation:

- Plugin was updated
- PluginPreset was updated
- PluginOverride was updated

The Plugin's status should contain the list of PluginOverrides that were applied. This ensures that the user can easily see how the Plugin was configured.

The PluginOverrides should be applied together, this means if one changes the whole list must be reapplied to ensure consistency.
The order of application of the PluginOverrides must be from most generic first, to most specific last. This means that a PluginOverride not specifying a Cluster or PluginDefinition will be applied first, and a PluginOverride specifying a Cluster and a PluginDefinition will be applied last.

In the case that a Plugin/PluginPreset already specifies a value that is covered by the override, than the override is ignored. This means that the Plugin/PluginPreset has precedence over the PluginOverride.

## Consequences

- Changes to a PluginOptionValue in a Plugin will be overridden by the PluginOverride Operator. This means overriden values can only be changed by updating the PluginOverride.
- Order of PluginOverrides is fixed from the most general to the most specific last. This means a PluginOverride not specifying Cluster or PluginDefinition will be applied first, and a PluginOverride specifying a Cluster and a PluginDefinition will be applied last.
