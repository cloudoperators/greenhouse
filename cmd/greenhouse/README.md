## Running Greenhouse Operator Locally

An extensive guide is available in the [localenv](../../dev-env/README.md) documentation.

This is a quick note on running the operator with Dex storage backend configuration

### Using `kubernetes` as the Dex storage backend

When running the operator locally the kubernetes mode is automatically detected:

1. `KUBECONFIG` environment variable - example - `export KUBECONFIG=/path/to/config` (Priority 1)
2. kube config from the recommended dir and file - example - `$HOME/.kube/config` (Priority 2)
3. Running inside a kubernetes cluster, `in-cluster` mode is used. (Priority 3)

### Using `postgres` as the Dex storage backend

If you are using `postgres` as the dex storage backend, you need to set the following environment variables:

- `PG_DATABASE=<postgres-database>` ex: `postgres` (defaults to `postgres` if not set)
- `PG_PORT=<postgres-port>` ex: `5432` (defaults to `5432` if not set)
- `PG_USER=<postgres-user>` ex: `postgres` (defaults to `postgres` if not set)
- `PG_HOST=<postgres-host>` ex: `localhost` (required)
- `PG_PASSWORD=<postgres-password>` ex: `password` (required)

### Running Catalog Controller Locally

To run the Catalog Controller locally, you need to port-forward flux `source-watcher` SVC to your localhost.

Example command:

```shell
kubectl -n flux-system port-forward svc/source-watcher 5050:80
```

Then set the following environment variable when running the operator (IDE Debugger or Shell):

```shell
ARTIFACT_DOMAIN=localhost:5050
```
