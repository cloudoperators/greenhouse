

# Setting up development environment

This handy CLI tool will help you to setup your development environment in no time.

## Prerequisites

- [docker](https://docs.docker.com/get-docker/)
- [KinD](https://kind.sigs.k8s.io/docs/user/quick-start/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

## Usage

You can use `greenhousectl` either by downloading the latest binary
from [here](https://github.com/cloudoperators/greenhouse/releases)

Or you can build it from source by running the following command: `make cli`

> [!NOTE]  
> The CLI binary will be available in the `bin` folder

## Setting up the development environment

There are multiple local development environment setup available for the Greenhouse project. You can choose the one that
fits your needs.

`All commands will spin up KinD clusters and setup the necessary components`

### Develop controllers locally and run the webhook server in-cluster

```shell
make setup-controller-dev
```

> [!NOTE]
> set the environment variable `CONTROLLERS_ONLY=true` in your debugger configuration
> If no environment variable is set, the webhook server will error out due to the missing certs

### Develop Admission Webhook server locally

```shell
make setup-webhook-dev
```

> [!NOTE]
> set the environment variable `WEBHOOK_ONLY=true` in your debugger configuration
> If both the controllers and the webhook server needs to be run locally, do not set any environment variable

### Running Greenhouse Dashboard in-cluster

```shell
make setup-dashboard
```

> [!NOTE]
> You will need to port-forward the cors-proxy service and the dashboard service to access the dashboard
> Information on how to access the dashboard is displayed after the command is executed

### Develop Greenhouse Dashboard locally

```shell
make setup
```

- This will install the operator, cors-proxy, sample organization with an onboarded remote cluster
- Additionally, it also creates a `appProps.json` `ConfigMap` in the `greenhouse` namespace
- You can now retrieve the generated `appProps.json` in-cluster by executing
  `kc get cm greenhouse-dashboard-app-props -n greenhouse -o=json | jq -r '.data.["appProps.json"]'`
- Optionally you can also redirect this output to `appProps.json`
  in [Juno Repository](https://github.com/cloudoperators/juno/blob/main/apps/greenhouse/README.md)
- Follow the instructions in the terminal to `port-forward` the cors-proxy service (ignore the `port-forward` of
  dashboard service)
- Start the dashboard locally
- `PluginDefinition(s)` can be applied
  from [Greenhouse Extensions](https://github.com/cloudoperators/greenhouse-extensions) repository


### Test Plugin / Greenhouse Extension charts locally

```shell
PLUGIN_DIR=<absolute-path-to-charts-dir> make setup
```

- This will install a full running setup of operator, dashboard, sample organization with an onboarded remote cluster
- Additionally, it will mount the plugin charts directory on to the `node` of the `KinD` cluster
- The operator deployment has a hostPath volume mount to the plugin charts directory from the `node` of the `KinD`
  cluster

You would need to apply the `PluginDefinition(s)` of the chart that needs to be tested.

However, before applying the `PluginDefinition(s)`, you need to modify the `PluginDefinition(s)` to point to a local
file path.

Modify `spec.helmChart.name` to point to the local file path of the chart that needs to be tested

Example Scenario:

You have cloned the [Greenhouse Extensions](https://github.com/cloudoperators/greenhouse-extensions) repository,
and you want to test `cert-manager` plugin chart locally.

```yaml

apiVersion: greenhouse.sap/v1alpha1
kind: PluginDefinition
metadata:
  name: cert-manager
spec:
  description: Automated TLS certificate management
  displayName: Certificate manager
  docMarkDownUrl: >-
    https://raw.githubusercontent.com/cloudoperators/greenhouse-extensions/main/cert-manager/README.md
  helmChart:
    name: cert-manager # <- replace it with 'local/plugins/cert-manager/charts/v1.11.0/cert-manager'
    repository: oci://ghcr.io/cloudoperators/greenhouse-extensions/charts # <- replace it with empty ''
    version: 1.11.0 # <- replace it with empty ''
...

```

## Additional information

When setting up your development environment, certain resources are modified for development convenience -

- The manager `Deployment` has environment variables `WEBHOOK_ONLY` and `CONTROLLERS_ONLY`
- `WEBHOOK_ONLY=true` will only run the webhook server
- `CONTROLLERS_ONLY=true` will only run the controllers
- Only one of the above can be set to `true` at a time otherwise the manager will error out

if `DevMode` is enabled for webhooks then depending on the OS the webhook manifests are altered by removing
`clientConfig.service` and replacing it with `clientConfig.url`, allowing you to debug the code locally.

- `linux` - the ipv4 addr from `docker0` interface is used - ex: `https://172.17.0.2:9443/<path>`
- `macOS` - host.docker.internal is used - ex: `https://host.docker.internal:9443/<path>`
- `windows` - ideally `host.docker.internal` should work, otherwise please reach out with a contribution <3
- webhook certs are generated by `charts/manager/templates/kube-webhook-certgen.yaml` Job in-cluster, and they are
  extracted and saved to `/tmp/k8s-webhook-server/serving-certs`
- `kubeconfig` of the created cluster(s) are saved to `/tmp/greenhouse/<clusterName>.kubeconfig`

---
## greenhousectl dev setup

setup dev environment with a configuration file

```
greenhousectl dev setup [flags]
```

### Examples

```

# Setup Greenhouse dev environment with a configuration file
greenhousectl dev setup -f dev-env/localenv/dev.config.yaml

- This will create an admin and a remote cluster
- Install CRDs, Webhook definitions, RBACs, Certs, etc... for Greenhouse into the admin cluster
- Depending on the devMode, it will install the webhook in-cluster or enable it for local development

Overriding certain values in dev.config.yaml:

- Override devMode for webhook development with d=true or devMode=true
- Override helm chart installation with c=true or crdOnly=true
- Override environment variables for manager deployment with e="ENV_NAME=VALUE" or env="ENV_NAME=VALUE" (can be repeated)

e.g. greenhousectl dev setup -f dev-env/localenv/dev.config.yaml d=true e="WEBHOOK_ONLY=false" e="CONTROLLERS_ONLY=true"

```

### Options

```
  -f, --config string   configuration file path - e.g. -f dev-env/localenv/dev.config.yaml
  -h, --help            help for setup
```

## greenhousectl dev setup dashboard

setup dashboard for local development with a configuration file

```
greenhousectl dev setup dashboard [flags]
```

### Examples

```

# Setup Greenhouse dev environment with a configuration file
greenhousectl dev setup dashboard -f dev-env/localenv/ui.config.yaml

- Installs the Greenhouse dashboard and CORS proxy into the admin cluster

```

### Options

```
  -f, --config string   configuration file path - e.g. -f dev-env/localenv/ui.config.yaml
  -h, --help            help for dashboard
```


## Generating Docs
To generate the markdown documentation, run the following command:
```shell
make dev-docs
```
