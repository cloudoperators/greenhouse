# Greenhouse via OCM + Flux — Step-by-Step Deployment Guide

---

## Why OCM + Flux?

You could deploy Greenhouse with raw `helm upgrade --install`. That works, but it requires
you to be present on every cluster, carry the right kubeconfig and Helm repos, and run
commands in the right order.

OCM + Flux solve this differently:

| Problem | OCM answer | Flux answer |
|---|---|---|
| Where do I store the charts? | Package into a versioned, signed bundle | — |
| How do they get onto the cluster? | Push to an OCI registry | — |
| Who installs the Helm releases? | — | Controllers running inside the cluster |
| How do I track desired vs actual state? | — | Flux reconciles continuously |

One sentence each:

- **OCM (Open Component Model)** — a standard for bundling software artifacts
  (Helm charts, container images, config) into a signed, versioned *component*
  stored in an OCI registry.
- **Flux** — Kubernetes controllers that watch OCI registries and apply Helm releases
  or raw manifests automatically.

---

## Greenhouse vs Keystone: different delivery model

Keystone uses a **hybrid** approach:
- OCM for versioning/signing the bundle
- Flux `OCIRepository` + `HelmRelease` CRs that point *directly* to ghcr.io chart paths

Greenhouse uses a **fully OCM-native** approach via `FluxDeployer`:
- OCM controller pulls each chart from the bundle into a local Snapshot (OCI artifact)
- `FluxDeployer` CRs auto-create `OCIRepository` + `HelmRelease` from those Snapshots
- Flux reads from the *local OCM registry* (not directly from ghcr.io)

This is more air-gap-friendly: once the OCM bundle is pulled to the cluster, no further
external registry access is needed for chart delivery.

---

## Architecture

```
OCI Registry (ghcr.io/sofiya-mahamadjavid-desai_sap/greenhouse-ocm)
  └── Component: github.com/cloudoperators/greenhouse v0.16.1
        ├── componentRef: core
        │     ├── greenhouse-chart   (helmChart)
        │     └── greenhouse-image   (ociImage)
        ├── componentRef: cert-manager
        │     └── cert-manager-chart (helmChart)
        ├── componentRef: flux
        │     └── flux2-chart        (helmChart)
        ├── componentRef: kro
        │     └── kro-chart          (helmChart)
        └── componentRef: ocm-controller
              └── ocm-controller-chart (helmChart)
                       │
               ComponentVersion CR (greenhouse ns)
                       │
               Resource CRs → OCM Snapshots (local OCI artifacts, ocm-system)
                       │
               FluxDeployer CRs → OCIRepository + HelmRelease (auto-created in greenhouse ns)
                       │
               Flux helm-controller installs:
                 cert-manager (namespace: cert-manager)
                 ocm-controller (namespace: ocm-system)
                 kro (namespace: kro-system)
                 greenhouse (namespace: greenhouse)
```

**Installation order** (enforced by `dependsOn`):

```
flux [bootstrapped manually]
     │
cert-manager ──────────────────────┐
     │                             │
ocm-controller   kro               │
                                   ▼
                              greenhouse
```

---

## File inventory

| File | Purpose |
|---|---|
| `ocm/component-constructor.yaml` | OCM component definition — what charts go in the bundle |
| `ocm/Makefile.ocm` | All commands wrapped as make targets |
| `ocm/greenhouse-bundle.ctf/` | Local OCM CTF archive (built by `make build`) |
| `ocm/charts/` | Downloaded Helm charts used during bundle build |
| `ocm/deploy/namespace.yaml` | greenhouse namespace + registry creds comment |
| `ocm/deploy/componentversion.yaml` | ComponentVersion CR — watches OCM registry |
| `ocm/deploy/resources.yaml` | Resource CRs — one per chart resource in the bundle |
| `ocm/deploy/flux-deployers.yaml` | FluxDeployer CRs — auto-creates OCIRepository + HelmRelease per chart |
| `ocm/deploy/flux-deployers-bootstrap-flux2.yaml` | flux2 FluxDeployer (for fresh air-gapped clusters ONLY) |
| `ocm/deploy/kustomization.yaml` | Groups all deploy manifests |
| `ocm/deploy/greenhouse-values-example.yaml` | Minimal working values for greenhouse chart |

---

## Bootstrapping problem and solution

There is a chicken-and-egg problem with this stack:

- To use `FluxDeployer`, you need the OCM controller
- OCM controller needs cert-manager (webhook TLS)
- cert-manager is installed *by* the cert-manager FluxDeployer

**Solution**: manually bootstrap Flux, cert-manager, and OCM controller *once* via
`helm install`. This is done by the `install-ocm-controller` make target. After that,
the FluxDeployer CRs take over and manage subsequent installs/upgrades — including
cert-manager and ocm-controller (they will adopt the manually installed releases).

Because cert-manager and ocm-controller are already installed when the FluxDeployers
are applied, those FluxDeployers perform an upgrade (not install) — which is safe.

---

## Concepts

### ComponentVersion

A `ComponentVersion` CR watches the OCM registry for a specific component name + semver.
When the OCM controller resolves the version, it downloads the component descriptor and
makes all bundled resources accessible to `Resource` CRs.

```yaml
apiVersion: delivery.ocm.software/v1alpha1
kind: ComponentVersion
metadata:
  name: greenhouse
  namespace: greenhouse
spec:
  component: github.com/cloudoperators/greenhouse
  version:
    semver: ">=0.16.1"
  repository:
    url: ghcr.io/sofiya-mahamadjavid-desai_sap/greenhouse-ocm
    secretRef:
      name: greenhouse-ocm-registry-creds
```

### Resource

A `Resource` CR extracts a single chart from the ComponentVersion (navigating the
component reference tree via `referencePath`). The OCM controller materialises it as
a local OCI artifact called a **Snapshot** — stored in the OCM controller's internal
OCI registry in `ocm-system`.

```yaml
apiVersion: delivery.ocm.software/v1alpha1
kind: Resource
metadata:
  name: greenhouse-chart
  namespace: greenhouse
spec:
  sourceRef:
    kind: ComponentVersion
    name: greenhouse
    resourceRef:
      name: greenhouse-chart
      referencePath:
        - name: core    # navigate into the "core" componentReference
```

### FluxDeployer

A `FluxDeployer` watches a `Resource`'s Snapshot and auto-creates:
1. An `OCIRepository` CR pointing to that Snapshot in the OCM internal registry
2. A `HelmRelease` CR with the fields you specify in `helmReleaseTemplate`

Both the `OCIRepository` and `HelmRelease` are created in the FluxDeployer's own
namespace (always `greenhouse` here). `targetNamespace` controls where the chart
*installs its resources*.

```yaml
helmReleaseTemplate:          # fields are helmv2.HelmReleaseSpec — NOT wrapped in spec:
  interval: 10m
  targetNamespace: greenhouse
  dependsOn:
    - name: cert-manager
      namespace: greenhouse   # dependsOn must also reference greenhouse namespace
```

### TLS secret cross-namespace

The OCM controller creates `ocm-registry-tls-certs` in `ocm-system`. The
`OCIRepository` CRs that FluxDeployer creates in `greenhouse` namespace also need
those TLS certs to connect to the OCM internal registry. Copy this secret to
`greenhouse` after the OCM controller starts.

---

## Prerequisites (tools)

```bash
# Check you have all required tools
ocm version     # >= 0.11
helm version    # >= 3.12
kubectl version
flux version    # >= 2.0
```

Install if missing:

```bash
# OCM CLI
curl -sfL https://ocm.software/install-cli.sh | bash

# Flux CLI
brew install fluxcd/tap/flux

# Helm
brew install helm
```

---

## Step 1 — Build and push the OCM bundle

> **Skip this step if the bundle is already in the registry.**
> The bundle for v0.16.1 is already at `ghcr.io/sofiya-mahamadjavid-desai_sap/greenhouse-ocm`.
> Run `make verify` to confirm.

```bash
cd greenhouse   # the cloned greenhouse repo
export GITHUB_TOKEN=<your-github-pat>

# Package the greenhouse chart (resolves file:// deps + OCI)
make -f ocm/Makefile.ocm package

# Fetch the ocm-controller chart (needs v-prefix OCI tag)
make -f ocm/Makefile.ocm fetch-prereqs

# Build the OCM CTF archive
make -f ocm/Makefile.ocm build

# Verify what's inside
make -f ocm/Makefile.ocm verify

# Push to ghcr.io
make -f ocm/Makefile.ocm push GITHUB_TOKEN=$GITHUB_TOKEN
```

**What the build creates:**

```
github.com/cloudoperators/greenhouse v0.16.1        (top-level product)
  ├── github.com/cloudoperators/greenhouse/core                v0.16.1
  │     ├── greenhouse-chart   helmChart
  │     └── greenhouse-image   ociImage → ghcr.io/cloudoperators/greenhouse:v0.16.1
  ├── github.com/cloudoperators/greenhouse/prerequisites/cert-manager   v1.16.1
  ├── github.com/cloudoperators/greenhouse/prerequisites/flux           v2.15.0
  ├── github.com/cloudoperators/greenhouse/prerequisites/kro            v0.9.4
  └── github.com/cloudoperators/greenhouse/prerequisites/ocm-controller v0.33.0
```

> **Note on OCM label names:** OCM component labels must use valid Kubernetes label key
> format (`prefix/name`, max 1 slash). Multi-slash names like
> `github.com/cloudoperators/greenhouse/version` will cause an
> `Invalid value` error in the OCM controller. Use `cloudoperators.github.com/role` style.

> **Note on chart version:** `helm package` must use `--version $(GREENHOUSE_VERSION)` (not
> `$(GREENHOUSE_CHART_VERSION)`) because the OCM controller tags Snapshots with the
> *component version* (`0.16.1`), and Flux enforces a strict version match on the chartRef.

---

## Step 2 — Get the cluster kubeconfig

Place the kubeconfig for the target cluster at `greenhouse-kubeconfig.yaml`
(in the `sci-k8s-cluster` root). This is already present as `greenhouse-kubeconfig.yaml`.

```bash
# Verify cluster access
KUBECONFIG=greenhouse-kubeconfig.yaml kubectl get nodes
```

All subsequent steps run from `greenhouse/ocm/` with
`SHOOT_KUBECONFIG=../../greenhouse-kubeconfig.yaml` (the Makefile.ocm default).

---

## Step 3 — Bootstrap Flux on the cluster

```bash
cd greenhouse/ocm
make bootstrap-flux
```

This runs `flux install` and waits for source-controller and helm-controller to be Ready.

```bash
# Verify
KUBECONFIG=../../greenhouse-kubeconfig.yaml kubectl get pods -n flux-system
```

Expected: source-controller, helm-controller, kustomize-controller, notification-controller
all in Running state.

---

## Step 4 — Grant helm-controller cluster-admin RBAC

The default Flux `helm-controller` ServiceAccount only has namespace-scoped permissions.
Installing CRDs and ClusterRoles (cert-manager, kro, ocm-controller) requires cluster-admin.

```bash
make grant-helm-rbac
```

> **Why this is required:** cert-manager installs `ValidatingWebhookConfiguration`,
> `ClusterRole`, `ClusterRoleBinding`, and a number of CRDs. Without cluster-admin,
> the HelmRelease will fail with `admission webhook denied` or RBAC errors.

---

## Step 5 — Install prometheus-operator CRDs

The greenhouse chart renders `PrometheusRule` objects regardless of whether monitoring
is enabled in values. If the `PrometheusRule` CRD doesn't exist, the helm install fails.

```bash
make install-prometheus-crds
```

This installs `prometheus-operator-crds` into the `monitoring` namespace.

---

## Step 6 — Bootstrap cert-manager and OCM controller

Install both via helm directly (one-time bootstrap, before FluxDeployers are applied):

```bash
make install-ocm-controller
```

This target:
1. Installs cert-manager v1.16.1 in `cert-manager` namespace with `installCRDs=true`
2. Installs OCM controller v0.33.0 in `ocm-system` namespace
3. Waits for the OCM controller to create the TLS secret `ocm-registry-tls-certs`

Wait for both to be healthy before continuing:

```bash
KUBECONFIG=../../greenhouse-kubeconfig.yaml kubectl get pods -n cert-manager
KUBECONFIG=../../greenhouse-kubeconfig.yaml kubectl get pods -n ocm-system
```

---

## Step 7 — Create namespace and secrets

### 7a. Create the greenhouse namespace

```bash
make create-greenhouse-ns
```

### 7b. Copy the OCM TLS secret to greenhouse namespace

The OCM controller's internal OCI registry uses TLS. FluxDeployer-created
`OCIRepository` objects in the `greenhouse` namespace need this cert to connect.

```bash
make copy-tls-secret
```

Verify:

```bash
KUBECONFIG=../../greenhouse-kubeconfig.yaml \
  kubectl get secret ocm-registry-tls-certs -n greenhouse
```

### 7c. Create registry credentials secret

This lets the `ComponentVersion` controller pull the greenhouse bundle from ghcr.io.

```bash
export GITHUB_TOKEN=<your-github-pat>
make create-registry-secret
```

> **Secret type must be `docker-registry`**, not `generic`. This stores credentials
> in `.dockerconfigjson`, the format OCI clients expect.

### 7d. Create the greenhouse values secret

```bash
# Copy the example and edit for your environment
cp deploy/greenhouse-values-example.yaml greenhouse-values.yaml
# Edit as needed, then:
make create-values-secret
```

**Minimum required values** (see `deploy/greenhouse-values-example.yaml` for the full
working minimal config):

```yaml
global:
  dnsDomain: greenhouse.example.com   # your DNS domain
  oidc:
    enabled: false                    # disable for test/dev
  dex:
    backend: kubernetes               # MUST be "kubernetes" or "postgres" — NOT "memory"
```

> **Important — `dex.backend`:** Using `memory` is invalid. Using `postgres` without
> setting `postgresqlUsername` creates a secret name with a trailing hyphen
> (`pguser-`) which is invalid. Use `kubernetes` for dev/test.

> **Important — `PrometheusRule`:** The greenhouse chart always renders
> `PrometheusRule` objects regardless of `monitoring.enabled`. Install the
> CRDs (step 5) before applying.

---

## Step 8 — Apply OCM + Flux manifests

```bash
make deploy-apply
```

This runs `kubectl apply -k deploy/` which applies:
- `namespace.yaml` — greenhouse namespace
- `componentversion.yaml` — ComponentVersion CR (watches ghcr.io bundle)
- `resources.yaml` — Resource CRs (one per chart: cert-manager, flux2, kro, ocm-controller, greenhouse)
- `flux-deployers.yaml` — FluxDeployer CRs (cert-manager, ocm-controller, kro, greenhouse)

> **flux2 FluxDeployer is excluded** from the main kustomization. Flux is already
> installed (step 3). The flux2 FluxDeployer is in a separate file
> `deploy/flux-deployers-bootstrap-flux2.yaml` for air-gapped fresh installs only.

Verify the ComponentVersion becomes Ready:

```bash
KUBECONFIG=../../greenhouse-kubeconfig.yaml \
  kubectl wait componentversion/greenhouse -n greenhouse \
  --for=condition=Ready --timeout=120s
```

---

## Step 9 — Watch Flux reconcile

```bash
make deploy-watch
```

Expected progression:

```
NAME            READY     STATUS
cert-manager    Unknown   Running 'install' action...
kro             Unknown   Running 'install' action...
ocm-controller  False     dependency 'greenhouse/cert-manager' is not ready

cert-manager    True      Helm install succeeded, revision 1
kro             True      Helm install succeeded, revision 1
ocm-controller  True      Helm install succeeded, revision 1

greenhouse      Unknown   Running 'install' action...
greenhouse      True      Helm install succeeded, revision 1
```

While greenhouse is installing, watch the init behaviour:

```bash
KUBECONFIG=../../greenhouse-kubeconfig.yaml kubectl get pods -n greenhouse -w
```

You should see the `greenhouse-controller-manager` pod become Running after the
cert-manager webhook is ready.

---

## Step 10 — Verify greenhouse is running

```bash
make deploy-status
```

Check HelmReleases and pods across all namespaces:

```bash
KUBECONFIG=../../greenhouse-kubeconfig.yaml kubectl get helmrelease -n greenhouse
# Expected: cert-manager, kro, ocm-controller, greenhouse — all READY True

KUBECONFIG=../../greenhouse-kubeconfig.yaml kubectl get pods -n greenhouse
# Expected: greenhouse-controller-manager Running (plus any enabled subcomponents)

KUBECONFIG=../../greenhouse-kubeconfig.yaml kubectl get pods -n cert-manager
# Expected: cert-manager-*, cert-manager-cainjector-*, cert-manager-webhook-* Running

KUBECONFIG=../../greenhouse-kubeconfig.yaml kubectl get pods -n ocm-system
# Expected: ocm-controller-* Running

KUBECONFIG=../../greenhouse-kubeconfig.yaml kubectl get pods -n kro-system
# Expected: kro-* Running
```

Verify greenhouse CRDs are installed:

```bash
KUBECONFIG=../../greenhouse-kubeconfig.yaml \
  kubectl get crd | grep greenhouse
```

You should see CRDs like `clusters.greenhouse.sap`, `organizations.greenhouse.sap`, etc.

---

## Step 11 — All-in-one deploy (fresh cluster)

For a fully automated fresh cluster deploy:

```bash
export GITHUB_TOKEN=<your-github-pat>
cp deploy/greenhouse-values-example.yaml greenhouse-values.yaml
# edit greenhouse-values.yaml as needed, then:
make deploy-full
```

`deploy-full` runs all steps in order:
1. `bootstrap-flux`
2. `grant-helm-rbac`
3. `install-prometheus-crds`
4. `install-ocm-controller` (cert-manager + OCM controller bootstrap)
5. `create-greenhouse-ns`
6. `create-registry-secret`
7. `copy-tls-secret`
8. `create-values-secret`
9. `deploy-apply`

---

## Upgrade flow

To upgrade greenhouse to a new version:

1. **Bump versions in Makefile.ocm:**

   ```makefile
   GREENHOUSE_VERSION        ?= 0.17.0
   GREENHOUSE_CHART_VERSION  ?= 0.24.0
   ```

2. **Rebuild and push the bundle:**

   ```bash
   make package build push GITHUB_TOKEN=$GITHUB_TOKEN
   ```

3. **Update the ComponentVersion semver in deploy/componentversion.yaml:**

   ```yaml
   version:
     semver: ">=0.17.0"
   ```

4. **Apply:**

   ```bash
   make deploy-apply
   ```

   Flux detects the new ComponentVersion → OCM controller downloads the new charts →
   FluxDeployer updates the OCIRepositories → Flux runs `helm upgrade`. No manual
   intervention on the cluster.

---

## Troubleshooting

```bash
# Overall status
make deploy-status

# ComponentVersion events
KUBECONFIG=../../greenhouse-kubeconfig.yaml \
  kubectl describe componentversion greenhouse -n greenhouse

# Resource events
KUBECONFIG=../../greenhouse-kubeconfig.yaml \
  kubectl describe resource greenhouse-chart -n greenhouse

# FluxDeployer events (shows OCIRepository + HelmRelease it created)
KUBECONFIG=../../greenhouse-kubeconfig.yaml \
  kubectl describe fluxdeployer greenhouse -n greenhouse

# HelmRelease events
KUBECONFIG=../../greenhouse-kubeconfig.yaml \
  kubectl describe helmrelease greenhouse -n greenhouse

# Force immediate reconciliation
make deploy-reconcile

# Flux controller logs
KUBECONFIG=../../greenhouse-kubeconfig.yaml \
  kubectl logs -n flux-system deploy/helm-controller --tail=50
KUBECONFIG=../../greenhouse-kubeconfig.yaml \
  kubectl logs -n flux-system deploy/source-controller --tail=50

# OCM controller logs
KUBECONFIG=../../greenhouse-kubeconfig.yaml \
  kubectl logs -n ocm-system deploy/ocm-controller --tail=50
```

### Common failure modes

| Symptom | Likely cause | Fix |
|---|---|---|
| `ComponentVersion` stays `NotReady` | Wrong registry URL or bad credentials | Check `greenhouse-ocm-registry-creds` secret exists in `greenhouse` ns |
| `ComponentVersion` Not Ready: `semver no match` | Bundle not pushed or version mismatch | Run `make push` or check semver constraint |
| `Resource` stays `NotReady` | `referencePath` wrong (wrong component ref name) | Check `referencePath` in resources.yaml matches component-constructor componentReferences |
| `FluxDeployer` creates OCIRepository but it fails TLS | TLS cert not copied to greenhouse ns | Run `make copy-tls-secret` |
| `HelmRelease` cert-manager fails: `cannot patch ClusterRole` | helm-controller lacks cluster-admin | Run `make grant-helm-rbac` |
| `HelmRelease` greenhouse fails: `no matches for kind PrometheusRule` | prometheus-operator CRDs missing | Run `make install-prometheus-crds` |
| `HelmRelease` greenhouse fails: `dex backend "memory"` | invalid dex backend value | Set `global.dex.backend: kubernetes` in values |
| `HelmRelease` greenhouse fails: invalid secret name | `postgres` backend without `postgresqlUsername` | Use `dex.backend: kubernetes` or set username |
| `HelmRelease` stuck in terminal error | Previous failed install left stale Helm state | Run `make flux-fix APP=greenhouse` |
| OCM component labels error: `Invalid value` | Label name contains more than one slash | Use `prefix/name` format (e.g. `cloudoperators.github.com/role`) |
| Chart version mismatch: Snapshot/HelmRelease not Ready | Chart packaged with wrong version | Package with `--version $(GREENHOUSE_VERSION)` not chart version |
| `HelmRelease` dependsOn not satisfied | Dependency HelmRelease not yet Ready | Wait for cert-manager to become Ready first |

### Recovery: HelmRelease stuck in terminal failed state

```bash
# For any stuck HelmRelease:
make flux-fix APP=greenhouse
# or:
make flux-fix APP=cert-manager
```

This removes the stuck finalizer, clears the stale Helm release secret, restarts
helm-controller, and re-applies the deploy manifests.

---

## Comparison with Keystone deployment

| Dimension | Keystone (OCM + Flux) | Greenhouse (OCM + FluxDeployer) |
|---|---|---|
| Chart delivery mechanism | Flux OCIRepository → ghcr.io directly | FluxDeployer → OCM Snapshot → local registry |
| Who creates OCIRepository + HelmRelease? | You (manually apply flux-helmreleases.yaml) | OCM controller via FluxDeployer (auto-created) |
| Air-gap friendly? | Needs ghcr.io access per chart | Yes — after initial bundle pull, no external registry needed |
| Number of OCM components | 1 flat component | 1 top-level + 5 sub-components (nested) |
| Bootstrap needed? | Flux only | Flux + cert-manager + OCM controller (manual helm install) |
| TLS secret copy needed? | No | Yes — OCM internal registry TLS cert must be in greenhouse ns |
