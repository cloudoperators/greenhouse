
# Setting up a local development environment

Greenhouse provides a couple of cli commands based on `make` to run a local Greenhouse instance.

- [Setting up the development environment](#setting-up-the-development-environment)
- [Run local Greenhouse](#run-local-greenhouse)
- Developing Greenhouse core functionality:
  - [Develop Controllers locally and run the webhook server in-cluster](#develop-controllers-locally-and-run-the-webhook-server-in-cluster)
  - [Develop Admission Webhook server locally](#develop-admission-webhook-server-locally)
- Greenhouse Dashboard
  - [Running Greenhouse Dashboard in-cluster](#running-greenhouse-dashboard-in-cluster)
  - [Run Greenhouse Core for UI development](#run-greenhouse-core-for-ui-development)
- Greenhouse Extensions
  - [Test Plugin / Greenhouse Extension charts locally](#test-plugin--greenhouse-extension-charts-locally)
- [Additional information](#additional-information)

This handy CLI tool will help you to setup your development environment in no time.

## Prerequisites

- [docker](https://docs.docker.com/get-docker/)
- [KinD](https://kind.sigs.k8s.io/docs/user/quick-start/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

## Usage

Build `greenhousectl` from source by running the following command: `make cli`

> [!NOTE]  
> The CLI binary will be available in the `bin` folder

## Setting up the development environment

There are multiple local development environment setup available for the Greenhouse project. You can choose the one that
fits your needs.

`All commands will spin up KinD clusters and setup the necessary components`

if you have a `~/.kube/config` file then `KinD` will automatically merge the `kubeconfig` of the created cluster(s)

use `kubectl config use-context kind-greenhouse-admin` to switch to `greenhouse admin` cluster context
use `kubectl config use-context kind-greenhouse-remote` to switch to `greenhouse remote` cluster context

if you do not have the contexts of the created cluster(s) in `~/.kube/config` file then you can extract it from the
operating system's `tmp` folder, where the CLI will write `kubeconfig` of the created `KinD` clusters

> [!NOTE]
> `linux / macOS`: in `unix` like systems you can find the `kubeconfig` at `$TMPDIR/greenhouse/<clusterName>.kubeconfig`
>
> `windows`: in `windows` many tmp folders exist so the CLI can write the `kubeconfig` to the first non-empty value from
`%TMP%`, `%TEMP%`, `%USERPROFILE%`
>
> The path where the `kubeconfig` is written will be displayed in the terminal after the command is executed by the CLI

use `kubectl --kubeconfig=<path to admin / remote kubeconfig>` to interact with the local `greenhouse` clusters

### Run Greenhouse Locally

```shell
make setup
```

- This will install the operator, the dashboard, cors-proxy and a sample organization with an onboarded remote cluster
- port-forward the `cors-proxy` by `kubectl port-forward svc/greenhouse-cors-proxy 9090:80 -n greenhouse &`
- port-forward the `dashboard` by `kubectl port-forward svc/greenhouse-dashboard 5001:80 -n greenhouse &`
- Access the local `demo` organization on the Greenhouse dashboard on [localhost:5001](http://localhost:5001/?org=demo)

### Develop Controllers locally and run the webhook server in-cluster

```shell
make setup-controller-dev
```

> [!NOTE]
> set the environment variable `CONTROLLERS_ONLY=true` in your debugger configuration
>
> If no environment variable is set, the webhook server will error out due to the missing certs

### Develop Admission Webhook server locally

```shell
make setup-webhook-dev
```

> [!NOTE]
> set the environment variable `WEBHOOK_ONLY=true` in your debugger configuration if you only want to run the webhook
> server

### Develop Controllers and Admission Webhook server locally

```shell
DEV_MODE=true WEBHOOK_ONLY=true make setup-manager
```

This will modify the `ValidatingWebhookConfiguration` and `MutatingWebhookConfiguration` to use the
`host.docker.internal` (macOS / windows) or `ipv4` (linux) address for the webhook server
Write the webhook certs to `/tmp/k8s-webhook-server/serving-certs`

> [!NOTE]
> The deployed controller-manager is running in webhook only mode, but it does not receive any requests

Now you can run the webhook server and the controllers locally

Since both need to be run locally no `CONTROLLERS_ONLY` or `WEBHOOK_ONLY` environment variables are needed in your
debugger configuration


> [!NOTE]
> The dev setup will modify the webhook configurations to have 30s timeout for the webhook requests, but
> when break points are used to debug webhook requests, it can result into timeouts.
> In such cases, modify the CR with a dummy annotation to re-trigger the webhook request and reconciliation

### Running Greenhouse Dashboard in-cluster

```shell
make setup-dashboard
```

> [!NOTE]
> You will need to port-forward the cors-proxy service and the dashboard service to access the dashboard
>
> Information on how to access the dashboard is displayed after the command is executed


### Run Greenhouse Core for UI development
- Startup the environment as in [Run local Greenhouse](#run-local-greenhouse)
- An `appProps.json` `ConfigMap` is created  in the `greenhouse` namespace to configure the dashboard.
- You can now retrieve the generated `appProps.json` in-cluster by executing
  `kubectl get cm greenhouse-dashboard-app-props -n greenhouse -o=json | jq -r '.data.["appProps.json"]'`
- Optionally you can also redirect this output to `appProps.json`
  in [Juno Repository](https://github.com/cloudoperators/juno/blob/main/apps/greenhouse/README.md)
- After port-forwarding `cors-proxy` service, it should be used as `apiEndpoint` in `appProps.json`
- Start the dashboard locally (more information on how to run the dashboard locally can be found in
  the [Juno Repository](https://github.com/cloudoperators/juno/blob/main/apps/greenhouse/README.md)
### Test Plugin / Greenhouse Extension charts locally

```shell
PLUGIN_DIR=<absolute-path-to-charts-dir> make setup
```

- This will install a full running setup of operator, dashboard, sample organization with an onboarded remote cluster
- Additionally, it will mount the plugin charts directory on to the `node` of the `KinD` cluster
- The operator deployment has a hostPath volume mount to the plugin charts directory from the `node` of the `KinD`
  cluster

To test your local Chart (now mounted to the KinD cluster) with a `plugindefinition.yaml` you would need to adjust `.spec.helmChart.name` to use the local chart.
With the provided mounting mechanism it will always live in `local/plugins/` within the KinD cluster.

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
    name: 'local/plugins/<path-to-cert-manager-chart-folder>'
    repository: '' # <- has to be empty
    version: '' # <- has to be empty
...

```

Apply the `plugindefinition.yaml` to the `admin` cluster

```shell
kubectl --kubeconfig=<your-kind-config> apply -f plugindefinition.yaml
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