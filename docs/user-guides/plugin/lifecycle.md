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

Catalogs enable Greenhouse Organizations to import the PluginDefinitions they want to use.

A Catalog resource points to a Git repository that contains PluginDefinition (or ClusterPluginDefinition) manifests. It is defined by:

- **`spec.source.git.url`** — the URL of the Git repository containing the PluginDefinition manifests.
- **`spec.source.git.ref`** — an optional Git reference to pin the catalog to a specific `branch`, `tag`, or `sha` (commit). If omitted, defaults to the repository's default branch.
- **`spec.source.path`** — an optional path within the repository where the manifests are located (defaults to the repository root).
- **`spec.overrides`** — an optional list of overrides to rename/alias PluginDefinitions via Kustomize patches. Each override specifies a `name` (original PluginDefinition name) and an `alias` (new name to apply).

Under the hood, the Catalog controller creates a Flux [GitRepository](https://fluxcd.io/flux/components/source/gitrepositories/) and a [Kustomization](https://fluxcd.io/flux/components/kustomize/kustomizations/) to continuously sync PluginDefinitions from the referenced Git repository into the Organization namespace.

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

Versioning of a PluginDefinition is achieved with the Catalog resource. Since a Catalog references a Git repository via `spec.source.git`, you control which PluginDefinition versions are available in your Organization by controlling the Git reference:

- **Branch tracking** (e.g. `branch: main`): You always get the latest PluginDefinitions as they are committed to the branch. This is the default behavior and enables automatic updates.
- **Tag pinning** (e.g. `tag: "kube-monitoring/1.2.3"`): You only get PluginDefinitions as of that tagged commit. Updates only happen when you change the tag reference.
- **SHA pinning** (e.g. `sha: "a1b2c3d4e5f6..."`): You freeze to an exact commit. This provides the strongest version guarantee.

##### Pinning PluginDefinition versions

Reference by branch (always track latest):

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: Catalog
metadata:
  name: my-catalog
  namespace: my-org
spec:
  source:
    git:
      url: https://github.com/cloudoperators/greenhouse-extensions
      ref:
        branch: main
    path: plugindefinitions
```

Reference by tag (pin to a specific release):

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: Catalog
metadata:
  name: my-catalog-pinned
  namespace: my-org
spec:
  source:
    git:
      url: https://github.com/cloudoperators/greenhouse-extensions
      ref:
        tag: "kube-monitoring/1.2.3"
    path: plugindefinitions
```

Reference by commit SHA (freeze to exact state):

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: Catalog
metadata:
  name: my-catalog-frozen
  namespace: my-org
spec:
  source:
    git:
      url: https://github.com/cloudoperators/greenhouse-extensions
      ref:
        sha: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
    path: plugindefinitions
```

##### Overriding PluginDefinition names

You can use `spec.overrides` to alias PluginDefinitions, for example to run multiple configurations of the same PluginDefinition side by side:

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: Catalog
metadata:
  name: my-custom-catalog
  namespace: my-org
spec:
  source:
    git:
      url: https://github.com/my-org/my-plugin-catalog
      ref:
        branch: main
    path: plugindefinitions
  overrides:
    - name: cert-manager
      alias: cert-manager-custom
    - name: ingress-nginx
      alias: ingress-nginx-custom
    - name: kube-monitoring
      alias: kube-monitoring-custom
```

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
