---
title: "Plugin Lifecycle Management"
linkTitle: "Plugin LCM"
weight: 5
description: >
  Software lifecycle management done the Greenhouse way
---

## What is Plugin Lifecycle Management (LCM)?

When we are talking about Plugin LCM we refer to the active maintenance phase of software provided as Greenhouse Plugins. This includes tasks such as:

- Bug fixing
- Feature shipping
- Dependency updates
- Version upgrades
- and more...

The following features are offered via Greenhouse:

| Feature                   | Status |
|---------------------------|:-----------:|
| Automatic Updates         | 🟩          |
| Automatic Rollback on Failing Updates         | 🟩          |
| Version Constraining/Pinning | 🟩        |
| Staged Rollout            | 🟨          |

## Involved actors

### Plugin Developers

These are the people providing the Plugins that can be used with Greenhouse. These Plugins are offered via (Cluster-)Plugindefinitions.
Plugin developers ensure that PluginDefinition versions follow strict [SemVer](https://semver.org/).

They control

- The code of the Plugin including the Helm chart and possible frontend code.
- The PluginDefinition manifests
- The repository where the PluginDefinitions live in

### Plugin Consumers

People configuring Plugins in their Organizations.

They control the actual resources deployed to their Greenhouse Organization:

- Catalogs
- PluginDefinitions
- PluginPresets (& Plugins)

### Involved Resources

#### [Catalogs](./../../reference/api/index.html#greenhouse.sap/v1alpha1.Catalog)

Catalogs enable Greenhouse Organizations to import the PluginDefinitions they want to use. A Catalog resource points to a Git repository that contains PluginDefinition (or ClusterPluginDefinition) manifests and continuously syncs them into the Organization namespace.

For full configuration details, see the [Catalog reference documentation](https://cloudoperators.github.io/greenhouse/docs/reference/components/catalog/).

#### [PluginDefinitions](./../../reference/api/index.html#greenhouse.sap/v1alpha1.PluginDefinition)

PluginDefinitions bundle backend and frontend packages with configuration. Backends are shipped as [Helm charts](https://helm.sh/) and frontends as [Juno Applications](https://github.com/cloudoperators/juno).

All PluginDefinitions are versioned (`.Spec.Version`) with [SemVer](https://semver.org/).

#### [PluginPresets](./../../reference/api/index.html#greenhouse.sap/v1alpha1.PluginPreset)

PluginPresets allow you to configure a set of Plugins to be deployed to a set of Clusters referencing a PluginDefinition.

### Features

#### Automatic Updates

**All** updates and upgrades (`major`, `minor` and `patch`) made to a PluginDefinition are shipped to all referencing Plugin(Presets) by default via the Greenhouse controller.

Greenhouse per default follows a fix forward update strategy.

We strongly encourage Plugin Consumers to always keep their Plugin versions up to date. All Greenhouse provided processes aim at easing upgrades or provide auto upgrade strategies.

#### Automatic Rollback on Failure

The underlying flux machinery's [`.spec.upgrade.remediation` is set to `rollback`](https://fluxcd.io/flux/components/helm/helmreleases/#upgrade-remediation). This will keep the Plugins running even if updates or upgrades fail.

#### Version Pinning and Constraints

Versioning of a PluginDefinition is achieved with the Catalog resource. Since a Catalog references a Git repository via `spec.source.git`, you control which PluginDefinition versions are available in your Organization by controlling the Git reference.

For details on how to pin PluginDefinition versions using `spec.source.git.ref`, see the [Catalog Ref configuration](https://cloudoperators.github.io/greenhouse/docs/reference/components/catalog/#ref).

For details on how to override PluginDefinition names using `spec.overrides`, see [Configuring overrides for PluginDefinitions](https://cloudoperators.github.io/greenhouse/docs/reference/components/catalog/#configuring-overrides-for-plugindefinitions).

#### Staged rollouts

Until Greenhouse has an inhouse solution for staging rollouts, we suggest you use [renovate](https://github.com/renovatebot/renovate) configuration in combination with `git tags` to stage rollouts of PluginDefinition versions.

The following steps are needed:

1. Plugin developers need to `git tag` their PluginDefinitions and/or Catalogs. The tagging convention is `<plugin-name>/<version>` (e.g. `kube-monitoring/1.2.3`).

    > **Tip:** Automate this with a CI workflow that reads `.spec.version` from `plugindefinition.yaml` on push to `main` and creates the corresponding git tag.

2. Have renovate open PRs to update the resources in your Catalogs. The following example shows a renovate configuration for a Catalog maintained with a Helm Chart that expects values to be nested in `common.catalogs`:

    E.g.

    ```json
    "customManagers": [
    {
      "customType": "jsonata",
      "fileFormat": "yaml",
      "description": "Update catalog tags in values.yaml (github.com)",
      "managerFilePatterns": [
        "/values\\.yaml$/"
      ],
      "matchStrings": [
        "common.catalogs.*.sources[ref.tag and $match(ref.tag, /^[^\\/]+\\/\\d+\\.\\d+\\.\\d+$/) and $contains(repository, 'github.com')].({\"depName\": $split(ref.tag, '/')[0], \"packageName\": $substringAfter(repository, 'https://github.com/'), \"currentValue\": ref.tag, \"datasource\": 'github-tags', \"registryUrl\": 'https://github.com', \"versioning\": 'regex:^' & $split(ref.tag, '/')[0] & '/(?<major>\\\\d+)\\\\.(?<minor>\\\\d+)\\\\.(?<patch>\\\\d+)$'})"
      ]
    }

  ],
    ```

1. Maintain different Catalogs for your stages.

2. Set renovate PRs to `automerge` and configure a `schedule` for the different stages. With no manual interaction (e.g. blocking PRs) you will roll through your stages based on your schedule.
