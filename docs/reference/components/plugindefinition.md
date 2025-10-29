---
title: "PluginDefinitions"
weight: 2
---

A PluginDefinition brings either a UI application, a Helm chart deployment, or both, to the Greenhouse platform. The Helm chart for a PluginDefinition can be used to deploy infrastructure components to a Kubernetes cluster managed with Greenhouse. The PluginDefinition provides an opinionated way to configure, integrated and deploy these components with Greenhouse.

## Example PluginDefinition Spec

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: PluginDefinition
metadata:
  name: alerts
  namespace: example-organization
spec:
  description: The Alerts Plugin consists of both Prometheus Alertmanager and Supernova,
    the holistic alert management UI
  displayName: Alerts
  docMarkDownUrl: https://raw.githubusercontent.com/cloudoperators/greenhouse-extensions/main/alerts/README.md
  helmChart:
    name: alerts
    repository: oci://ghcr.io/cloudoperators/greenhouse-extensions/charts
    version: 4.0.3
  icon: https://raw.githubusercontent.com/cloudoperators/greenhouse-extensions/main/alerts/logo.png
  uiApplication:
    name: supernova
    version: latest
  version: 5.0.3
  weight: 0
```

## Writing a PluginDefinition Spec

`.spec.displayName` is a human-readable name for the PluginDefinition. This field is optional; if not provided, it defaults to the value of `metadata.name`. This name is used in the Greenhouse UI to display the PluginDefinition in the Catalog of available PluginDefinitions.

`.spec.version` specifies the semantic version of the PluginDefinition. This versions the combination of Helm chart, UI application and any options provided in the PluginDefinition. The version should be incremented whenever any of these fields are updated.

`.spec.uiApplication` is an optional field that specifies the UI application associated with the PluginDefinition. The UI application will be made available in the Greenhouse UI when a Plugin is created from this PluginDefinition.

```yaml
spec:
  uiApplication:
    name: supernova
    version: latest
  weight: 0
  icon: https://raw.githubusercontent.com/cloudoperators/greenhouse-extensions/main/alerts/logo.png
  docMarkDownUrl: https://raw.githubusercontent.com/cloudoperators/greenhouse-extensions/main/alerts/README.md
```

The fields `weight` and `icon` are optional and are used to customize the appearance of the Plugin in the Greenhouse UI sidebar. The optional field `docMarkDownUrl` can be used to provide a link to documentation for the PluginDefinition, which will be displayed in entry of available PluginDefinitions in the Greenhouse UI.

`.spec.helmChart` is an optional field that specifies the Helm chart that is deployed when creating a Plugin from this PluginDefinition.

```yaml
spec:
 helmChart:
    name: alerts
    repository: oci://ghcr.io/cloudoperators/greenhouse-extensions/charts
    version: 4.0.3
```

`.spec.options` is an optional field that specifies default configuration options for the PluginDefinition. These options will be pre-filled when creating a Plugin from this PluginDefinition, but can be overridden by the user.

```yaml
spec:
  options:
  - description: Alertmanager API Endpoint URL
    name: endpoint
    required: true
    type: string
  - description: FilterLabels are the labels shown in the filter dropdown, enabling
      users to filter alerts based on specific criteria. The format is a list of strings.
    name: filterLabels
    required: false
    type: list
  - default: false
    description: Install Prometheus Operator CRDs if kube-monitoring has not already
      installed them.
    name: alerts.crds.enabled
    required: false
    type: bool
```

`.required` indicates whether the option is mandatory when creating a Plugin from this PluginDefinition. `.default` contains the default value for the option if the Plugin does not provide a value for it. `.type` is used to enforce validation of the value. The following types are supported: `string`, `bool`, `int`, `list`, `map` and `secret`.

| :information_source: The type secret requires a secret reference. Disallowing clear-text credentials. |

## Next Steps

- [Creating a PluginDefinition](./../../../contribute/plugins)
- [PluginPreset reference](./../pluginpreset)
- [Plugin reference](./../plugin)
