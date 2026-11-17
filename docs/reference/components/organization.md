---
title: "Organizations"
weight: 1
---

An Organization is the top-level entity within Greenhouse. Each Organization represents a different tenant within Greenhouse and is provided with a dedicated Namespace in the Greenhouse cluster.

## Example Organization Spec

```yaml
apiVersion: greenhouse.sap/v1alpha1
kind: Organization
metadata:
  name: example-organization
spec:
  authentication:
    oidc:
      issuer: https://accounts.example.com
      clientId: example-client-id
```

## Writing an Organization Spec

### DisplayName

`.spec.displayName` is a human-friendly name for the Organization. This field is optional.

### Authentication

#### OIDC

`.spec.authentication.oidc` is used to configure how members of the Organization will authenticate to Greenhouse. Greenhouse IDProxy is using [Dex](https://dexidp.io/) to provide the OIDC authentication for multiple Organizations. Each Organization receives their own Dex connector.

The config requires `issuer`, the URL of the identity provider, and `clientIDReference` and `clientSecretReference`, which reference Kubernetes Secrets containing the OIDC client ID and client secret, respectively.

```yaml
spec:
  authentication:
    oidc:
      issuer: https://accounts.example.com
      clientIdReference:
        name: oidc-client-id-secret
        key: client-id
      clientSecretReference:
        name: oidc-client-secret
        key: client-secret
```

`.authentication.oidc.redirectURI` is an optional field to specify a custom redirect URI for OIDC authentication. If not provided, it defaults to the Greenhouse ID Proxy (`auth.<greenhouse domain name>`).

`.authentication.oidc.oauth2ClientRedirectURIs` is an optional list of URIs that are added to the Dex connector as allowed redirect URIs for OAuth2 clients.

#### SCIM

`.spec.authentication.scim` is used by Greenhouse to retrieve the members of a Team from the Organization's identity provider. This field is optional; if not provided, Greenhouse will not attempt to sync users via SCIM.

The configuration requires `baseURL`, the URL of the SCIM endpoint, and `authType`, which specifies the authentication method to use when connecting to the SCIM endpoint. Supported methods are `basic` and `token`.

```yaml
spec:
  authentication:
    scim:
      baseURL: https://scim.example.com
      authType: token
      bearerToken:
        secret:
          name: scim-bearer-token-secret
          key: bearer-token
      bearerPrefix: Bearer
      bearerHeader: Authorization
```

`.authentication.scim.bearerPrefix` is an optional field to specify a custom prefix for the bearer token in the authorization header. If not provided, it defaults to `Bearer`.

`.authentication.scim.bearerHeader` is an optional field to specify a custom header name for the bearer token. If not provided, it defaults to `Authorization`.

### MappedOrgAdminIdPGroup

`.spec.mappedOrgAdminIdPGroup` is an optional field that specifies the name of an identity provider group whose members will be granted Organization Admin privileges within Greenhouse. If this field is not provided, no users will be granted Organization Admin privileges.

## Working with Organizations

### Role-Based Access Control within the Organization namespace

Greenhouse provisions a set of default Roles and RoleBindings within each Organization's Namespace to facilitate Role-Based Access Control (RBAC). These roles can be used by the Organization Admins as a starting point to manage access to resources within their Organization.

The following roles are seeded for each Organization:

| Name                            | Description                                                | ApiGroups                 | Resources                                                                                            | Verbs                       | Cluster scoped |
| ------------------------------- | ---------------------------------------------------------- | ------------------------- | ---------------------------------------------------------------------------------------------------- | --------------------------- | ---- |
| `role:<org-name>:admin`         | An admin of a Greenhouse `Organization`. This entails the permissions of `role:<org-name>:cluster-admin` and `role:<org-name>:plugin-admin`                    | `greenhouse.sap/v1alpha1`, `greenhouse.sap/v1alpha2` | \*                                                                                                   | \*                          | - |
|                                 |                                                            | `v1`                      | `secrets`                                                                                            | \*                          | - |
|                                 |                                                            | `""`                      | `pods`, `replicasets`, `deployments`, `statefulsets`, `daemonsets`, `cronjobs`, `jobs`, `configmaps` | `get`, `list`, `watch`      | - |
|                                 |                                                            | `monitoring.coreos.com`   | `alertmanagers`, `alertmanagerconfigs`                                                               | `get`, `list`, `watch`      | - |
| `role:<org-name>:cluster-admin` | An admin of Greenhouse `Clusters` within an `Organization` | `greenhouse.sap/v1alpha1`, `greenhouse.sap/v1alpha2` | `clusters`, `teamrolebindings`                                                                       | \*                          | - |
|                                 |                                                            | `v1`                      | `secrets`                                                                                            | `create`, `update`, `patch` | - |
| `role:<org-name>:plugin-admin`  | An admin of Greenhouse `Plugins` within an `Organization`  | `greenhouse.sap/v1alpha1` | `plugins`, `pluginpresets`, `catalogs`, `plugindefinitions`                                                                           | \*                          | - |
|                                 |                                                            | `v1`                      | `secrets`                                                                                            | `create`, `update`, `patch` | - |
| `organization:<org-name>`        | A member of a Greenhouse `Organization`                    | `greenhouse.sap/v1alpha1` | \*                                                                                                   | `get`, `list`, `watch`      | - |
| `organization:<org-name>`       | A member of a Greenhouse `Organization`                    | `greenhouse.sap/v1alpha1` | `organizations`, `clusterplugindefinitions`                                                                 | `get`, `list`, `watch`      | x |

## Release Events

An Organization can report every new Helm release of its Plugins to an HTTP endpoint. It is configured in the Organization ConfigMap set in `spec.configMapRef` and a Secret in the Organization namespace. See [Reporting Plugin Releases](./../../../user-guides/organization/release-events) for a walkthrough.

### ConfigMap keys

| Key | Description |
|-----|-------------|
| `releaseEventsConfig` | The payload template. Release events are on for the Organization when this key is set. |
| `releaseEventsSecretName` | The name of the Secret with the endpoint and the credentials. Required when `releaseEventsConfig` is set. |

### Secret keys

| Key | Required | Description |
|-----|----------|-------------|
| `authType` | yes | `BASIC` or `OAUTH2`. |
| `endpoint` | yes | The URL the events are sent to. |
| `username` | yes | The basic auth user, or the client ID with `OAUTH2`. |
| `password` | yes | The basic auth password, or the client secret with `OAUTH2`. |
| `tokenURL` | with `OAUTH2` | The token endpoint for the OAuth2 client credentials flow. |
| `tokenHeader` | no | The header the OAuth2 token is sent in, as the plain token. Without it the token is sent as `Authorization: Bearer <token>`. |

### What the template gets

The payload template is a Go template with the [Sprig](https://masterminds.github.io/sprig/) functions except `env` and `expandenv`. It has to render valid JSON or nothing. When it renders nothing, the release is skipped.

| Field | Description |
|-------|-------------|
| `.Organization` | The name of the Organization. |
| `.Plugin` | The Plugin, with `metadata`, `spec` and `status`, for example `.Plugin.Name`, `.Plugin.Labels` or `.Plugin.Spec.PluginDefinitionRef.Name`. |
| `.Cluster` | The Cluster the Plugin is deployed to, for example `.Cluster.Name` or `.Cluster.Labels`. Empty for a Plugin without `spec.clusterName`. |
| `.Release` | The Helm release as recorded by Flux in the HelmRelease history. |

The fields of `.Release`:

| Field | Description |
|-------|-------------|
| `.Release.Name` | The name of the Helm release. |
| `.Release.Namespace` | The namespace the release is deployed to. |
| `.Release.Version` | The revision of the release, starting at 1. |
| `.Release.Status` | The Helm status of the release, for example `deployed` or `failed`. |
| `.Release.Action` | The Helm action that created the release, for example `install` or `upgrade`. |
| `.Release.ChartName` | The name of the chart. |
| `.Release.ChartVersion` | The version of the chart. |
| `.Release.AppVersion` | The `appVersion` of the chart. |
| `.Release.FirstDeployed` | When the release was first deployed. |
| `.Release.LastDeployed` | When this revision was deployed. |
| `.Release.Digest` | The checksum of the release, unique for every revision. |
| `.Release.ConfigDigest` | The checksum of the values of the release. |

### Feature Flags

Release events are behind the feature flag `releaseEventsEnabled` in the `plugin` section of the `greenhouse-feature-flags` ConfigMap. It is `false` when unset or invalid. The flags are read when the Greenhouse controller starts.

```yaml
# greenhouse-feature-flags ConfigMap
apiVersion: v1
kind: ConfigMap
metadata:
  name: greenhouse-feature-flags
  namespace: greenhouse
data:
  plugin: |
    releaseEventsEnabled: true
```

While the flag is off, no events are sent and the `ReleaseEventSent` condition is removed from all Plugins.

## Next Steps

- [Creating an Organization](./../../../user-guides/organization/creation)
- [Reporting Plugin Releases](./../../../user-guides/organization/release-events)
