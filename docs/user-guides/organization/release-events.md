---
title: "Reporting Plugin Releases"
linkTitle: "Release Events"
weight: 4
description: >
  Send an event to an HTTP endpoint for every new Helm release of a Plugin.
---

## Overview

This guide describes how to report every new Helm release of the Plugins in an Organization to an HTTP endpoint, for example a change management system.

Every time a Plugin gets a new Helm release, Greenhouse renders a payload template with the Plugin, its Cluster and the release, and sends the result as a JSON POST to the endpoint. The Organization sets the endpoint and the credentials in a Secret, and the payload template in its ConfigMap.

See [Release Events](./../../../reference/components/organization#release-events) for more details on all fields.

## Prerequisites

The feature flag `plugin.releaseEventsEnabled` must be enabled in your Greenhouse installation. It is off by default, see [Feature Flags](./../../../reference/components/organization#feature-flags).

The Organization needs a ConfigMap set in `spec.configMapRef`. If it already has one, for example for [Registry Overrides](./../registry-overrides), the release events configuration goes into the same ConfigMap.

## Store the endpoint and the credentials

The Secret lives in the Organization namespace and holds the endpoint and how to authenticate there. With basic auth, Greenhouse sends the `username` and `password` with every request:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: release-events
  namespace: example-organization
type: Opaque
stringData:
  authType: BASIC
  endpoint: https://changes.example.com/api/events
  username: greenhouse
  password: <password>
```

With OAuth2, Greenhouse uses the `username` and `password` as client ID and client secret, gets a token from `tokenURL` with the client credentials flow and sends it with the request. The token is sent as `Authorization: Bearer <token>`. If the endpoint expects the token in a different header, set it in `tokenHeader`, for example `X-Auth-Token` for an OpenStack Keystone token:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: release-events
  namespace: example-organization
type: Opaque
stringData:
  authType: OAUTH2
  endpoint: https://changes.example.com/api/events
  username: <client ID>
  password: <client secret>
  tokenURL: https://identity.example.com/v3/OS-OAUTH2/token
  tokenHeader: X-Auth-Token
```

Greenhouse gets a new token for every event.

## Configure the payload

The Organization ConfigMap gets two keys. `releaseEventsSecretName` is the name of the Secret, `releaseEventsConfig` is the payload template. Release events are switched on for the Organization as soon as `releaseEventsConfig` is set.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: example-organization-config
  namespace: example-organization
data:
  releaseEventsSecretName: release-events
  releaseEventsConfig: |
    {{- if eq .Release.Status "deployed" }}
    {
      "organization": {{ .Organization | toJson }},
      "cluster": {{ .Cluster.Name | default "central" | toJson }},
      "plugin": {{ .Plugin.Name | toJson }},
      "owner": {{ index .Plugin.Labels "greenhouse.sap/owned-by" | default "unknown" | toJson }},
      "action": {{ .Release.Action | toJson }},
      "revision": {{ .Release.Version }},
      "chart": {{ printf "%s-%s" .Release.ChartName .Release.ChartVersion | toJson }},
      "appVersion": {{ .Release.AppVersion | toJson }},
      "deployedAt": {{ .Release.LastDeployed | toJson }}
    }
    {{- end }}
```

The template is a Go template with the [Sprig](https://masterminds.github.io/sprig/) functions, like a Helm chart. It gets the Organization name, the Plugin, the Cluster of the Plugin and the Helm release, see [What the template gets](./../../../reference/components/organization#what-the-template-gets). Use `toJson` for the values, it quotes and escapes them.

When the Plugin `cert-manager-example-cluster` is upgraded to the second revision, Greenhouse sends:

```json
{
  "organization": "example-organization",
  "cluster": "example-cluster",
  "plugin": "cert-manager-example-cluster",
  "owner": "example-team",
  "action": "upgrade",
  "revision": 2,
  "chart": "cert-manager-v1.17.2",
  "appVersion": "v1.17.2",
  "deployedAt": "2026-11-17T09:30:12Z"
}
```

### Skipping releases

Every new release is passed to the template, also a failed install or upgrade. When the template renders nothing, nothing is sent and the release counts as reported. The example above uses this to send only releases with the status `deployed`.

The template can also skip Plugins. For example, if the endpoint needs a region for every event, a template that starts with this line skips all Plugins on clusters without a `region` label:

```yaml
{{- if and (index .Cluster.Labels "region") (eq .Release.Status "deployed") }}
```

### Plugins without a cluster

A Plugin without `spec.clusterName` runs in the Greenhouse cluster itself. For these Plugins `.Cluster` is empty, so `.Cluster.Name` is an empty string. The example sends `central` instead with `default "central"`.

## What gets sent

Greenhouse sends an event for every Helm release of a Plugin that it has not sent yet:

- When release events are switched on, the releases that are already deployed are not sent. The next release of each Plugin is.
- A Plugin that is created after that sends its first install.
- If a Plugin gets several releases while the endpoint can't be reached, only the latest of them is sent once the endpoint is back.
- In rare cases a release can be sent twice, for example when the Plugin status can't be written after the event was sent. If duplicates matter, the endpoint can drop them by `.Release.Digest`, which is unique for every release.

The request is a `POST` with `Content-Type: application/json`. Any `2xx` response counts as success. Any other response, or no response within 10 seconds, counts as a failure.

Changes to the Secret or the ConfigMap are picked up the next time a Plugin is reconciled.

## Checking delivery

The condition `ReleaseEventSent` of a Plugin shows whether its latest release was sent:

```bash
kubectl describe plugin cert-manager-example-cluster --namespace=example-organization
```

When the event can't be sent, the condition is `False` with the error, and Greenhouse tries again every minute. The Plugin itself stays Ready.

## Troubleshooting

| Error | Cause |
|-------|-------|
| `releaseEventsSecretName is missing in ConfigMap <name>` | `releaseEventsConfig` is set, but `releaseEventsSecretName` is not. |
| `secrets "<name>" not found` | The Secret doesn't exist in the Organization namespace. |
| `secret <name> is missing endpoint, username or password` | One of the three keys is missing or empty. |
| `secret <name> has unsupported authType "<type>", expected BASIC or OAUTH2` | `authType` is missing or has a typo. It is case sensitive. |
| `secret <name> is missing tokenURL` | `authType` is `OAUTH2`, but `tokenURL` is not set. |
| `template: payloadTemplate:<line>: ...` | The template can't be parsed, for example because of a function that doesn't exist, or fails on the data of the Plugin. |
| `payloadTemplate did not render valid JSON` | The rendered payload is not valid JSON, for example because of a missing comma or a value without `toJson`. |
| `release events endpoint responded with <status>` | The endpoint rejected the event, for example `401 Unauthorized` for wrong credentials or `400 Bad Request` for a payload it doesn't accept. |
| `failed to send the release event, see the controller logs` | The endpoint or the token URL can't be reached, or the token request failed. The full error is in the logs of the Greenhouse controller. |

## Next Steps

- [Organization reference](./../../../reference/components/organization)
- [Registry Overrides](./../registry-overrides)
- [Plugin reference](./../../../reference/components/plugin)
