# Greenhouse authz

Authorization server for Greenhouse, enabling support-group-scoped access control via Kubernetes authorizer webhook.

## Prerequisites

- Kubernetes 1.30+
- [cert-manager](https://cert-manager.io/) for TLS certificate management

## Installing the Chart

```shell
helm upgrade --install greenhouse-authz oci://ghcr.io/cloudoperators/greenhouse/charts/authz \
  --namespace greenhouse \
  --create-namespace
```

## Gardener Usage

Use [shoot-dns-service](https://gardener.cloud/docs/extensions/shoot-dns-service/) and [HA VPN](https://gardener.cloud/docs/gardener/high-availability/) to make the webhook reachable from the seed.

### How it works

1. **HA VPN** — makes ClusterIPs reachable from the seed. See [Gardener docs](https://gardener.cloud/docs/gardener/reversed-vpn-tunnel/#high-availability-for-reversed-vpn-tunnel).
2. **DNSEntry** — shoot-dns-service registers a DNS record pointing to the static ClusterIP. See [Gardener docs](https://gardener.cloud/docs/guides/networking/DNS-extension/#creating-a-dnsentry-resource-explicitly).
3. **TLS SAN** — the serving certificate includes the DNS hostname as a SAN.

### Requirements

- [shoot-dns-service](https://gardener.cloud/docs/extensions/shoot-dns-service/) extension enabled on the shoot
- [HA VPN](https://gardener.cloud/docs/gardener/high-availability/) enabled on the shoot

### Configuration

Pick a static `ClusterIP` from the shoot's service CIDR (check `spec.networking.services` in the shoot resource)

Example configuration:

```yaml
service:
  clusterIP: "100.104.1.10"

tls:
  extraDNSNames:
    - "greenhouse-authz.your-shoot.example.tld"

dnsEntry:
  annotations:
    dns.gardener.cloud/class: garden
```

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| args | list | `[]` | Additional args to pass to the authz binary. |
| commonAnnotations | object | `{}` | Annotations to add to all resources. |
| commonLabels | object | `{}` | Labels to add to all resources. |
| dnsEntry.annotations | object | `{}` | Annotations to add to the DNSEntry. Set `dns.gardener.cloud/class: garden` to enable shoot-dns-service processing. |
| dnsEntry.dnsName | string | `""` | DNS name to register. Required when `dnsEntry.enabled` is true. |
| dnsEntry.enabled | bool | `false` | Enable DNSEntry resource. Set to true when using Gardener shoot-dns-service to register a DNS record for the webhook. |
| env | list | `[]` | Additional environment variables. |
| fullnameOverride | string | `""` | Overrides the full name of the chart. |
| ha.enabled | bool | `false` | When enabled, configures podAntiAffinity and topologySpreadConstraints for high availability. |
| image.digest | string | `""` | Image digest. Takes precedence over tag if set. |
| image.pullPolicy | string | `"IfNotPresent"` | Image pull policy. |
| image.repository | string | `"ghcr.io/cloudoperators/greenhouse"` | Image repository. |
| image.tag | string | `""` | Overrides the image tag. Defaults to the chart appVersion. |
| imagePullSecrets | list | `[]` | Image pull secrets. |
| nameOverride | string | `""` | Overrides the chart name. |
| podSecurityContext | object | `{}` | Pod security context. |
| replicaCount | int | `1` | Number of replicas. |
| resources | object | `{}` | Resource requests and limits. |
| securityContext | object | `{}` | Container security context. |
| service.annotations | object | `{}` | Annotations to add to the Service. |
| service.clusterIP | string | `""` | Static ClusterIP. Required as the DNSEntry target when `dnsEntry.enabled` is true. |
| service.port | int | `9443` | Port the service listens on. |
| service.type | string | `"ClusterIP"` | Kubernetes service type. |
| serviceAccount.annotations | object | `{}` | Annotations to add to the ServiceAccount. |
| serviceAccount.create | bool | `true` | Whether to create a ServiceAccount. |
| serviceAccount.name | string | `""` | The name of the ServiceAccount to use. If not set, a name is generated using the fullname template. |
| tls.enabled | bool | `true` | Enable TLS serving certs. Set to false for local debugging outside the cluster. |
| tls.extraDNSNames | list | `[]` | Additional DNS SANs to add to the serving certificate. Example: extraDNSNames:   - "greenhouse-authz.example.tld" |
| tls.secretName | string | `""` | Name of the cert-manager generated Secret. Defaults to `<fullname>-serving-certs`. |
| tls.useSecret | bool | `true` | When true, mounts the cert-manager Secret as the serving-certs volume. When false, supply the volume yourself via `volumes`/`volumeMounts`. |
| volumeMounts | list | `[]` | Additional volumeMounts to add to the container. Example: volumeMounts:   - name: serving-certs     mountPath: /tmp/k8s-webhook-server/serving-certs     readOnly: true |
| volumes | list | `[]` | Additional volumes to add to the Deployment. Example: volumes:   - name: serving-certs     hostPath:       path: /etc/kubernetes/webhook/certs |
