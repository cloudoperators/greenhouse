# Setting up development environment

This handy CLI tool will help you to setup your development environment in no time.
## Prerequisites
- [docker](https://docs.docker.com/get-docker/)
- [KinD](https://kind.sigs.k8s.io/docs/user/quick-start/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
## Usage

You can use `greenhousectl` either by downloading the latest binary from [here](https://github.com/cloudoperators/greenhouse/releases)

Or you can build it from source by running the following command: `build-greenhousectl`

> [!NOTE]  
> The CLI binary will be available in the `bin` folder

## Additional information

Charts needed for dev env setup for `KinD`

- `charts/manager`
- `charts/idproxy`

When setting up your development environment, certain resources are modified for development convenience -

 - The manager `Deployment` has environment variables `WEBHOOK_ONLY` and `CONTROLLERS_ONLY`
 - `WEBHOOK_ONLY=true` will only run the webhook server
 - `CONTROLLERS_ONLY=true` will only run the controllers
 - Only one of the above can be set to `true` at a time otherwise the manager will error out

if `DevMode` is enabled for webhooks then depending on the OS the webhook manifests are altered by removing `clientConfig.service` and 
replacing it with `clientConfig.url`, allowing you to debug the code locally.

> [!NOTE]  
> The `DevMode` can be enabled by setting the `--dev-mode` flag while individually setting up the webhook or by setting the `devMode` key to `true` in the `dev-env/localenv/sample.config.json` file.

- `linux` - the ipv4 addr from `docker0` interface is used - ex: `https://172.17.0.2:9443/<path>`
- `macOS` - host.docker.internal is used - ex: `https://host.docker.internal:9443/<path>`
- `windows` - ideally `host.docker.internal` should work, otherwise please reach out with a contribution :heart
- webhook certs are generated by `charts/manager/templates/kube-webhook-certgen.yaml` Job in-cluster and they are extracted and saved to `/tmp/k8s-webhook-server/serving-certs`
- `kubeconfig` of the created cluster(s) are saved to `/tmp/greenhouse/<clusterName>.kubeconfig`

Below you will find a list of commands available for dev env setup
  
---