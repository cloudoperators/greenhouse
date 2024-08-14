## localenv setup webhook

Setup webhooks for Greenhouse (Validating and Mutating webhooks)

### Synopsis

Setup Validating and Mutating webhooks for Greenhouse controller development convenience

```
localenv setup webhook [flags]
```

### Examples

```
localenv setup webhook -c my-kind-cluster-name -n my-namespace -p path/to/chart -f path/to/Dockerfile
```

### Options

```
  -p, --chartPath string    local chart path where manifests are located - e.g. <path>/<to>/charts/manager
  -x, --current-context     Use your current kubectl context
  -f, --dockerfile string   local path to the Dockerfile of greenhouse manager
  -h, --help                help for webhook
  -k, --kubeconfig string   Path to the kubeconfig file
  -c, --name string         Name of the kind cluster - e.g. my-cluster (without the kind prefix)
  -n, --namespace string    namespace to install the resources
```

### SEE ALSO

* [localenv setup](localenv_setup.md)	 - setup Greenhouse

