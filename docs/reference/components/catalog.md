---
title: "Catalogs"
weight: 5
---

A Catalog is a collection of PluginDefinitions that can be made available to Organizations within Greenhouse. 
Catalogs allow organization admins to manage the lifecycle of PluginDefinitions by controlling which version of a PluginDefinition is deployed to their cluster fleet.

> The Catalog API is currently in **alpha** and is still under active development and is subjected to change.

## Example

The following is an example of a Greenhouse Catalog that reconciles the PluginDefinition manifests stored in a Git Repository. 

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: Catalog
metadata:
  name: greenhouse-extensions
  namespace: <organization-namespace>
spec:
  sources:
    - repository: https://github.com/cloudoperators/greenhouse-extensions
      resources:
        - alerts/plugindefinition.yaml
        - audit-logs/plugindefinition.yaml
        - cert-manager/plugindefinition.yaml
        - external-dns/plugindefinition.yaml
        - repo-guard/plugindefinition.yaml
        - ingress-nginx/plugindefinition.yaml
        - kube-monitoring/plugindefinition.yaml
        - logs/plugindefinition.yaml
        - perses/plugindefinition.yaml
        - thanos/plugindefinition.yaml
      ref:
        branch: main
```

In the above example:

- The Greenhouse Catalog references a Git Repository targeting the `main` branch.
- The Catalog is configured to target specific `PluginDefinition` manifests stored in a path within the repository specified under `.spec.sources[].resources[]`.
- The Catalog watches for changes in the specified Git Repository branch and reconciles the `PluginDefinitions` in the Organization namespace accordingly.


## Writing a Catalog Spec

### Sources

`.spec.sources` is a list of sources from which the Catalog will fetch PluginDefinition manifests. Currently, only Git repositories are supported as sources.
Each source requires a `repository` URL and a list of `resources` that specify the paths to the PluginDefinition manifests within the repository. 
The `ref` field is used to specify the branch, tag, or commit SHA to fetch from the repository.

⚠️  Each Organization has a dedicated ServiceAccount used to apply the Catalog resources. This ServiceAccount only has permissions to apply PluginDefinitions into the Organization's Namespace. It will not be possible to bring other Kinds using the Catalog.

### Repository

`.spec.sources[].repository` is used to specify the URL of the Git repository containing the PluginDefinition manifests.

### Resources

`.spec.sources[].resources` is a list of file paths within the repository that point to the PluginDefinition manifests to be included in the Catalog.

> Empty resources list will result in an error.

### Ref

`.spec.sources[].ref` is used to specify the branch, tag, or commit SHA to fetch from the repository.

Available fields are:

- **sha** - The commit SHA to fetch.
- **tag** - The tag to fetch.
- **branch** - The branch that is targeted.

The priority order is: _sha_ > _tag_ > _branch_. If multiple fields are specified, the field with the highest priority will be used.

> When multiple sources are defined with the same repository and ref, a duplicate error will be raised.

### Interval (Optional)

`.spec.sources[].interval` is an optional field that specifies how often the Catalog should check for updates in the source repository.
The default value is `1h` (1 hour) if not specified.

The value must be in a [Go recognized duration format](https://pkg.go.dev/time#ParseDuration), e.g. `5m0s` to reconcile the source every 5 minutes.

### Secret Reference (Optional)

`.spec.sources[].secretName` is an optional field that specifies a reference to a Kubernetes Secret name containing credentials for accessing private Git repositories.

The following are the types of authentication supported:

- [Basic Authentication](#configuring-secret-for-basic-authentication)
- [GitHub App Credentials](#configure-secret-for-github-app-authentication)

The Secret must be in the same namespace as the Catalog resource.

### Overrides (Optional)

`.spec.sources[].overrides` is an optional field that allows specifying overrides for specific PluginDefinitions in the Catalog. 
This can be used to customize certain fields of PluginDefinitions.

Currently, you can override the following fields:

- `alias` field to override `metadata.name` of the PluginDefinition
- `repository` field to override the helm chart repository in the PluginDefinition spec

## Working with Catalog

### Configuring Secret for Basic Authentication:

To authenticate towards a Git repository over HTTPS using basic access authentication (in other words: using a username and password),
the referenced Secret is expected to contain `.data.username` and `.data.password` values.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: git-credentials
  namespace: <catalog-namespace>
type: Opaque
data:
  username: <BASE64>
  password: <BASE64>
```

> password is the Personal Access Token (PAT) for accessing the Git repository.

### Configure Secret for GitHub App authentication:

Pre-requisites:

- [Register](https://docs.github.com/en/apps/creating-github-apps/registering-a-github-app/registering-a-github-app) the GitHub App with the necessary permissions and generate a private key for the app.

- [Install](https://docs.github.com/en/apps/using-github-apps/installing-your-own-github-app) the app in the organization/account configuring access to the necessary repositories.

To authenticate towards a GitHub repository using a GitHub App, the referenced secret is expected to contain the following values:

- Get the App ID from the app settings page at `https://github.com/settings/apps/<app-name>`.
- Get the App Installation ID from the app installations page at `https://github.com/settings/installations`. Click the installed app, the URL will contain the installation ID `https://github.com/settings/installations/<installation-id>`. For organizations, the first part of the URL may be different, but it follows the same pattern.
- The private key that was generated in the pre-requisites.
- (Optional) GitHub Enterprise Server users can set the base URL to http(s)://HOSTNAME/api/v3.
- (Optional) If GitHub Enterprise Server uses a private CA, include its bundle (root and any intermediates) in `ca.crt`. If the `ca.crt` is specified, then it will be used for TLS verification for all API / Git over HTTPS requests to the GitHub Enterprise Server.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: github-app-credentials
  namespace: <catalog-namespace>
type: Opaque
stringData:
  githubAppID: "5001"
  githubAppInstallationID: "1005"
  githubAppBaseURL: "github.enterprise.example.com/api/v3" #optional, required only for GitHub Enterprise Server users
  githubAppPrivateKey: |
    -----BEGIN RSA PRIVATE KEY-----
    .....
    -----END RSA PRIVATE KEY-----
  ca.crt: | #optional, for GitHub Enterprise Server users
    -----BEGIN CERTIFICATE-----
    ...
    -----END CERTIFICATE-----
```

> Minimum required permissions for the GitHub App or Personal Access Token (PAT) is content read access for the target repository.

### Configuring Overrides for PluginDefinitions

If you want to bring in multiple versions of the same PluginDefinition, you can use the `alias` option to reference the `PluginDefinition` under a different name.

Example:

```yaml
spec:
  sources:
    - repository: https://github.com/cloudoperators/greenhouse-extensions
      resources:
        - perses/plugindefinition.yaml
      ref:
        branch: main
      overrides:
        - name: perses
          alias: perses-latest
    - repository: https://github.com/cloudoperators/greenhouse-extensions
      resources:
        - perses/plugindefinition.yaml
      ref:
        sha: <commit-sha>
      overrides:
        - name: perses
          alias: perses-stable
```

> `overrides[].name` must match the `metadata.name` of the PluginDefinition being overridden.

Each PluginDefinition has a helm chart reference in its spec. If you want to override the helm chart repository,
you can do that in `overrides`

Example:

```yaml
spec:
  sources:
    - repository: https://github.com/cloudoperators/greenhouse-extensions
      resources:
        - perses/plugindefinition.yaml
      ref:
        branch: main
      overrides:
        - name: perses
          repository: oci://your-registry.io/some-repo/perses-chart
```

PluginDefinitions can define configuration options with default values in their `.spec.options[]` array. You can override these default values using the `optionsOverride` field in the Catalog overrides.

This is useful when you want to:
- Provide different default values for different environments (dev, staging, production)
- Customize plugin behavior without modifying the original PluginDefinition
- Set organization-specific defaults for plugins

Example using the Perses PluginDefinition:

```yaml
spec:
  sources:
    - repository: https://github.com/cloudoperators/greenhouse-extensions
      resources:
        - perses/plugindefinition.yaml
      ref:
        branch: main
      overrides:
        - name: perses
          optionsOverride:
            - name: perses.image.version
              value: "0.47.0"
            - name: perses.serviceMonitor.selector.matchLabels
              value:
                app.kubernetes.io/name: perses
```

**NOTE:**

- The `optionsOverride[].name` must match an existing option name in the PluginDefinition's `.spec.options[]`
- The `value` type must match the option's type.
- This only overrides the **default value** of the option.
- If the option doesn't exist in the PluginDefinition, the override will be ignored

### Suspending the Catalog's reconciliation

The annotation `greenhouse.sap/suspend` can be added to a Catalog resource to temporarily suspend reconciliation.
When this annotation is present, the Catalog controller will set it's underlying Flux resources to suspended state.
Any changes made to the Catalog `.spec` will be ignored while the annotation is present.
To resume reconciliation, simply remove the annotation from the Catalog resource.
The annotation also blocks the deletion of the Catalog resource until the annotation is removed.

### Triggering reconciliation of the Catalog’s managed resources

The Catalog resource orchestrates a combination of Flux resources to fetch and apply PluginDefinitions.
The Flux resources managed by Catalog have their own reconciliation intervals.
To trigger an immediate reconciliation of the Catalog and its managed resources, the annotation `greenhouse.sap/reconcile` can be set.
This can be useful to trigger an immediate reconciliation when the source repository has changed, and you want to apply 
the changes without waiting for the next scheduled reconciliation.

### Troubleshooting

Greenhouse uses [FluxCD](https://fluxcd.io/) under the hood to reconcile Catalog sources and 
for each source a map of grouped status inventory is shown.

Run `kubectl get cat -n <organization-namespace>` to see the reconciliation status.

```shell
NAMESPACE    NAME                     READY
greenhouse   greenhouse-extensions    True
```

Run `kubectl describe cat greenhouse-extenions -n greenhouse` to see the reconciliation status conditions

```shell
Status:
  Inventory:
    github-com-cloudoperators-greenhouse-extensions-main-9689366613293914683:
      Kind:     GitRepository
      Message:  stored artifact for revision 'main@sha1:50cbc65c8e8ea390cb947f2a53e8f3dd33265417'
      Name:     repository-9689366613293914683
      Ready:    True
      Kind:     ArtifactGenerator
      Message:  reconciliation succeeded, generated 1 artifact(s)
      Name:     generator-9689366613293914683
      Ready:    True
      Kind:     ExternalArtifact
      Message:  Artifact is ready
      Name:     artifact-9689366613293914683
      Ready:    True
      Kind:     Kustomization
      Message:  Applied revision: latest@sha256:a6114ad3b1c591f1585d78818320d603e78d29b04f527c88321027c59372d506
      Name:     kustomize-9689366613293914683
      Ready:    True
  Status Conditions:
    Conditions:
      Last Transition Time:  2025-10-31T00:14:59Z
      Message:               all catalog objects are ready
      Reason:                CatalogReady
      Status:                True
      Type:                  Ready
```

Run `kubectl get gitrepository repository-9689366613293914683 -n greenhouse` to see the GitRepository status

```shell
NAME                             URL                                                       AGE     READY   STATUS
repository-9689366613293914683   https://github.com/cloudoperators/greenhouse-extensions   7d10h   True    stored artifact for revision 'main@sha1:50cbc65c8e8ea390cb947f2a53e8f3dd33265417'
```

Run `kubectl describe gitrepository repository-9689366613293914683 -n greenhouse` to see the reconciliation status conditions of the GitRepository

```shell
...
Spec:
  Interval:  60m0s
  Provider:  generic
  Ref:
    Branch:  main
  Timeout:   60s
  URL:       https://github.com/cloudoperators/greenhouse-extensions
Status:
  Artifact:
    Digest:            sha256:b2662d5c547a7b499c2030e9f646d292413e9745f1a8be9759a212375bc93b42
    Last Update Time:  2025-10-30T12:12:00Z
    Path:              gitrepository/greenhouse/repository-9689366613293914683/50cbc65c8e8ea390cb947f2a53e8f3dd33265417.tar.gz
    Revision:          main@sha1:50cbc65c8e8ea390cb947f2a53e8f3dd33265417
    Size:              7668967
    URL:               http://source-controller.flux-system.svc.cluster.local./gitrepository/greenhouse/repository-9689366613293914683/50cbc65c8e8ea390cb947f2a53e8f3dd33265417.tar.gz
  Conditions:
    Last Transition Time:  2025-10-30T12:12:00Z
    Message:               stored artifact for revision 'main@sha1:50cbc65c8e8ea390cb947f2a53e8f3dd33265417'
    Observed Generation:   2
    Reason:                Succeeded
    Status:                True
    Type:                  Ready
    Last Transition Time:  2025-10-30T12:12:00Z
    Message:               stored artifact for revision 'main@sha1:50cbc65c8e8ea390cb947f2a53e8f3dd33265417'
    Observed Generation:   2
    Reason:                Succeeded
    Status:                True
    Type:                  ArtifactInStorage
  Observed Generation:     2
Events:
  Type    Reason                 Age                   From               Message
  ----    ------                 ----                  ----               -------
  Normal  GitOperationSucceeded  4m40s (x51 over 12h)  source-controller  no changes since last reconcilation: observed revision 'main@sha1:50cbc65c8e8ea390cb947f2a53e8f3dd33265417'
```

In case of authentication failures due to invalid credentials, you will see errors in the GitRepository status conditions. (The same error message and ready status will also be reflected in the Catalog `.status.inventory` for the respective source.)

```shell
  - message: >-
      failed to checkout and determine revision: unable to list remote for
      'https://github.com/cloudoperators/greenhouse-extensions': authentication
      required: Invalid username or token.
    observedGeneration: 3
    reason: GitOperationFailed
    status: "False"
    type: Ready
```

PluginDefinitions referenced in `.spec.sources[].resources` are accumulated using Flux ArtifactGenerator. Run `kubectl get artifactgenerator generator-9689366613293914683 -n greenhouse` to see the status. 

```shell
NAME                            AGE     READY   STATUS
generator-9689366613293914683   7d11h   True    reconciliation succeeded, generated 1 artifact(s)
```

Run `kubectl describe artifactgenerator generator-9689366613293914683 -n greenhouse` to see the reconciliation status conditions of the ArtifactGenerator

```shell
Status:
  Conditions:
    Last Transition Time:  2025-10-31T00:14:59Z
    Message:               reconciliation succeeded, generated 1 artifact(s)
    Observed Generation:   2
    Reason:                Succeeded
    Status:                True
    Type:                  Ready
  Inventory:
    Digest:                 sha256:a6114ad3b1c591f1585d78818320d603e78d29b04f527c88321027c59372d506
    Filename:               2528970247.tar.gz
    Name:                   artifact-9689366613293914683
    Namespace:              greenhouse
  Observed Sources Digest:  sha256:bc9221b47aecc3f4c75f41b8657a3a7c985823487da94b8521803251a3628030
```

If there was an error accumulating the manifests, you will see errors in the ArtifactGenerator status conditions. (The same error message and ready status will also be reflected in the Catalog `.status.inventory` for the respective source.)

```shell
  - message: >-
      artifact-9689366613293914683 build failed: failed to apply copy
      operations: failed to apply copy operation from
      '@catalog/thanos/plugindefinition.yamls' to
      '@artifact/catalogs/thanos/plugindefinition.yamls': source
      'thanos/plugindefinition.yamls' does not exist
    observedGeneration: 3
    reason: BuildFailed
    status: "False"
    type: Ready
```

Finally, the accumulated manifests are applied using Flux Kustomization. Run `kubectl get kustomization kustomize-9689366613293914683 -n greenhouse` to see the status.

```shell
NAME                            AGE     READY   STATUS
kustomize-9689366613293914683   7d11h   True    Applied revision: latest@sha256:a6114ad3b1c591f1585d78818320d603e78d29b04f527c88321027c59372d506
```

Run `kubectl describe kustomization kustomize-9689366613293914683 -n greenhouse` to see the reconciliation status conditions of the Kustomization

In case of permissions issues, you will see errors in the Kustomization status conditions. (The same error message and ready status will also be reflected in the Catalog `.status.inventory` for the respective source.)

```shell
  - lastTransitionTime: "2025-10-31T00:45:08Z"
    message: >
      PluginDefinition/greenhouse/cert-manager dry-run failed
      (Forbidden): plugindefinitions.greenhouse.sap "cert-manager" is
      forbidden: User "system:serviceaccount:greenhouse:greenhouse-catalog-sa"
      cannot patch resource "plugindefinitions" in API group
      "greenhouse.sap" at the cluster scope
    observedGeneration: 4
    reason: ReconciliationFailed
    status: "False"
    type: Ready
```

## Next Steps

- [Catalog API Reference](./../../api/)
