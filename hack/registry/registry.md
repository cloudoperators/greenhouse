## Registry Mirror Mappings

The local zot registry runs at `registry:5000` (from inside the cluster) or `localhost:5000` (from the host machine).

| Source Registry   | Mirror Path                                        |
|-------------------|----------------------------------------------------|
| `docker.io`       | `registry:5000/greenhouse-dockerhub-mirror/...`    |
| `gcr.io`          | `registry:5000/greenhouse-gcr-mirror/...`          |
| `ghcr.io`         | `registry:5000/greenhouse-ghcr-io-mirror/...`      |
| `quay.io`         | `registry:5000/greenhouse-quay-mirror/...`         |
| `registry.k8s.io` | `registry:5000/greenhouse-registry-k8s-io-mirror/...` |

Mirrors are on-demand: zot pulls from the upstream registry on first request and caches locally.

## Push a Local Docker Image

Tag and push your image to the local registry:

```bash
docker tag myimage:latest localhost:5000/myimage:latest
docker push localhost:5000/myimage:latest
```

From inside the cluster, reference it as:

```yaml
image: registry:5000/myimage:latest
```

## Push a Local Helm Chart

Package and push a chart using `helm push` (OCI):

```bash
helm package ./charts/mychart
helm push mychart-0.1.0.tgz oci://localhost:5000/charts
```

Install from the local registry:

```bash
helm install myrelease oci://localhost:5000/charts/mychart --version 0.1.0
```

## Pull an Image via Mirrors

Zot pulls upstream images on demand. To warm the cache or verify a mirror path:

```bash
# docker.io/library/nginx:latest → pulled via dockerhub mirror
docker pull localhost:5000/greenhouse-dockerhub-mirror/library/nginx:latest

# ghcr.io/cloudoperators/greenhouse:latest → pulled via ghcr mirror
docker pull localhost:5000/greenhouse-ghcr-io-mirror/cloudoperators/greenhouse:latest

# registry.k8s.io/pause:3.9 → pulled via k8s registry mirror
docker pull localhost:5000/greenhouse-registry-k8s-io-mirror/pause:3.9
```

Inside a pod, use `registry:5000` instead of `localhost:5000`.

## Trigger Mirroring via Crane Manifest

Use `crane manifest` to trigger on-demand sync without pulling the full image layers:

```bash
# Pre-warm a docker.io image
crane manifest localhost:5000/greenhouse-dockerhub-mirror/library/nginx:latest

# Pre-warm a ghcr.io image
crane manifest localhost:5000/greenhouse-ghcr-io-mirror/cloudoperators/greenhouse:latest

# Pre-warm a gcr.io image
crane manifest localhost:5000/greenhouse-gcr-mirror/google-containers/pause:3.9

# Pre-warm a quay.io image
crane manifest localhost:5000/greenhouse-quay-mirror/prometheus/prometheus:latest
```

### Fetch manifest for specific platform

```bash
# ghcr.io
crane manifest --platform linux/amd64 localhost:5000/greenhouse-ghcr-io-mirror/fluxcd/helm-controller:v1.5.3

# gcr.io
crane manifest --platform linux/amd64 localhost:5000/greenhouse-gcr-mirror/distroless/static:nonroot
```

Install crane via `brew install crane` or `go install github.com/google/go-containerregistry/cmd/crane@latest`.
