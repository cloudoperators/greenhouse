## localenv manifests

install manifests for Greenhouse

### Synopsis

install CRDs, Webhook definitions, RBACs, Certs, etc... for Greenhouse into the target cluster

```
localenv manifests [flags]
```

### Examples

```
localenv manifests -x -n greenhouse -r greenhouse -p path/to/greenhouse/charts
```

### Options

```
  -p, --chartPath string           local absolute chart path where manifests are located - e.g. <path>/<to>/charts/manager
  -d, --crd-only                   Install only CRDs
  -x, --current-context            Use your current kubectl context
  -e, --excludeKinds stringArray   Exclude kinds from the generated manifests (default [Deployment])
  -h, --help                       help for manifests
  -k, --kubeconfig string          Path to the kubeconfig file
  -c, --name string                Name of the kind cluster - e.g. greenhouse-123 (without the kind prefix)
  -n, --namespace string           namespace to install the resources
  -r, --releaseName string         Helm release name, Default value: greenhouse - e.g. your-release-name (default "greenhouse")
  -v, --valuesPath string          local absolute values file path - e.g. <path>/<to>/my-values.yaml
```

