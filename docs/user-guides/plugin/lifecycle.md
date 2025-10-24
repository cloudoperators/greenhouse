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
| Version Constraining/Pinning | 🟨       |
| Staged Rollout            | 🟨          |

## Involved actors

### Plugin Developers

These are the people providing the Plugins that can be used with Greenhouse. These Plugins are offered via (Cluster-)Plugindefinitions.
Plugin developers ensure that PluginDefinition versions follow strict [SemVer](https://semver.org/).

They control

- The code of the Plugin including the Helm chart and possible frontend code.
- Default PluginDefinition
- Default Catalog(s)
- The repository where the PluginDefinitions and Catalogs live in

### Plugin Consumers

People configuring Plugins in their Organizations.

They control the actual resources deployed to their Greenhouse Organization:

- Custom Catalogs
- PluginDefinitions
- PluginPresets (& Plugins)

### Involved Resources

#### [Catalogs](./../../reference/api/index.html#greenhouse.sap/v1alpha1.Catalog)

Catalogs enable Greenhouse Organizations to import the PluginDefinitions they want to use. Leveraging the underlying [flux](https://fluxcd.io/) machinery, a Catalog item targets a `kustomization.yaml` living in a git repository.

This might be either

- the default kustomization provided by Plugin developers
- or a custom kustomization suited to the needs of your Organization

The Catalog resource allows you to target the `kustomization.yaml` via `branch`, `tag` or `commit` in the `.Spec.Source.Git.Ref` field.

#### [PluginDefinitions](./../../reference/api/index.htmlgreenhouse.sap/v1alpha1.PluginDefinition)

PluginDefinitions bundle backend and frontend packages with configuration. Backends are shipped as [Helm charts](https://helm.sh/) and frontends as [React components](https://react.dev/reference/react/Component).

All PluginDefinitions are versioned (`.Spec.Version`) with [SemVer](https://semver.org/).

#### [PluginPresets](./../../reference/api/index.htmlgreenhouse.sap/v1alpha1.PluginPresets)

PluginPresets allow you to configure a set of Plugins to be deployed to a set of Clusters referencing a PluginDefinition.

🟨 In development:
PluginPresets allow to pin or constrain deployed versions via the `.Spec.PluginDefinitionReference.Version` field with [semantic version constraints](https://jubianchi.github.io/semver-check/).

### Features

#### Automatic Updates

**All** updates and upgrades (`major`, `minor` and `patch`) made to a PluginDefinition are shipped to all referencing Plugin(Presets) by default via the Greenhouse controller if no version constraint is set.

Greenhouse per default follows a fix forward update strategy.

We strongly encourage Plugin Consumers to always keep their Plugin versions up to date. All Greenhouse provided processes aim at easing upgrades or provide auto upgrade strategies.

#### Automatic Rollback on Failure

The underlying flux machinery's [`.spec.upgrade.remediation` is set to `rollback`](https://fluxcd.io/flux/components/helm/helmreleases/#upgrade-remediation). This will keep the Plugins running even if updates or upgrades fail.

#### Version Pinning and Constraints

Until the `Plugin.Spec.PluginDefinitionReference.Version` field and its automation is fully implemented, we suggest to use custom Catalogs to maintain versioned PluginDefinitions in your Organization. You can do so by

- pinning versions of PluginDefinitions
- pinning version of a Catalog

versioned in a git repository.

##### Pinning PluginDefinition versions

Maintain a `kustomization.yaml` file targeting specific PluginDefinitions as resources, via git `commits`, `tags` or `branches`.
E.g.:

```yaml
resources:
# reference by branch
- https://raw.githubusercontent.com/cloudoperators/greenhouse-extensions/refs/heads/main/cert-manager/plugindefinition.yaml
# reference by commit
- https://raw.githubusercontent.com/cloudoperators/greenhouse-extensions/6c41128df8d9ff72c63aee7e3d3122468490ea21/ingress-nginx/plugindefinition.yaml
# reference by tag
- https://raw.githubusercontent.com/cloudoperators/greenhouse-extensions/refs/tags/v0.0.2/kube-monitoring/plugindefinition.yaml
```

Point your Catalog to this `kustomization.yaml` and **alias** the PluginDefinition `name`:

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: Catalog
metadata:
  name: my-custom-catalog
  namespace: {{ .Release.Namespace }}
spec:
  source:
    git:
      url: https://<url-to-repo-with-kustomization>
      ref:
        branch: main
    path: path/to/kustomization
  overrides:
    - name: cert-manager
      alias: cert-manager-custom
    - name: ingress-nginx
      alias: ingress-nginx-custom
    - name: kube-monitoring
      alias: kube-monitoring-custom

```

!Note:
> There is currently some restrictions in targeting raw data on Github Enterprise installations with flux. Maintain the `kustomization.yaml` together with the targeted PluginDefinitions in one repository targeted by the Catalog instead.

##### Pinning Catalogs

Point your Catalog to an existing `kustomization.yaml` and optionally alias the PluginDefintion `name`:

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: Catalog
metadata:
  name: greenhouse-extensions-pinned
  namespace: {{ .Release.Namespace }}
spec:
  source:
    git:
      url: https://github.com/cloudoperators/greenhouse-extensions
      ref:
        sha: <COMMIT_SHA>
  overrides:
    - name: alerts
      alias: alerts-pinned
    - name: audit-logs
      alias: audit-logs-pinned
    ...

```

#### Staged rollouts

Until Greenhouse has an inhouse solution for staging rollouts, we suggest you use [renovate](https://github.com/renovatebot/renovate) configuration in combination to `git tags` to stage rollouts of PluginDefinition versions.

The following steps are needed:

1. Plugin developers need to `git tag` their PluginDefinitions and/or Catalogs.
2. Have renovate open PRs to update the resources in your `kustomization.yaml` or in your `catalog.yaml`.

    E.g.

    <!-- TODO: Actually test this config! -->
    ```json
    "customManagers": [
      {
         "customType": "regex",
         "description": "Bump kube-monitoring version in kustomize",
         "managerFilePatterns": [
            "/(^|/)kustomization\\.ya?ml$/"
         ],
         "matchStrings": [
            "https://raw.githubusercontent.com/cloudoperators/greenhouse-extensions/refs/tags/(?<currentValue>.*?)/kube-monitoring/plugindefinition.yaml"
         ],
         "datasource": "git-tags",
         "depNameTemplate": "cloudoperators/kube-monitoring"
      }
    ]
    ```

3. Maintain different `kustomization.yaml` or `catalog.yaml` for your stages.

    ```md
    catalogs/
    ├── bronze/
    │   └── kustomization.yaml
    ├── silver/
    │   └── kustomization.yaml
    └── gold/
        └── kustomization.yaml

    ```

<!-- Need to actually test a valid automerge with  -->
4. Set renovate PRs to `automerge` and configure a `schedule` for the different stages. With no manual interaction (e.g. blocking PRs) you will roll through your stages based on your schedule.
