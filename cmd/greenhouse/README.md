## Running Greenhouse Operator Locally

An extensive guide is available in the [localenv](../../dev-env/localenv/README.md) documentation.

This is a quick note on running the operator with Dex storage backend configuration

### Using `kubernetes` as the Dex storage backend

If you are using `kubernetes` as the dex storage backend, you need to set the following environment variables:

- `KUBECONFIG=<path-to-kubeconfig>`

> [!NOTE]
> If your kube configs (for KinD) are merged in the default ~/.kube/config path
> then you can set the `KUBECONFIG` environment variable to `~/.kube/config` and set the current context to
> kind-greenhouse-admin
> `KUBECONFIG` is needed because dex will revert to InCluster mode if KUBECONFIG is not set

### Using `postgres` as the Dex storage backend

If you are using `postgres` as the dex storage backend, you need to set the following environment variables:

- `PG_DATABASE=<postgres-database>` ex: `postgres` (defaults to `postgres` if not set)
- `PG_PORT=<postgres-port>` ex: `5432` (defaults to `5432` if not set)
- `PG_USER=<postgres-user>` ex: `postgres` (defaults to `postgres` if not set)
- `PG_HOST=<postgres-host>` ex: `localhost` (required)
- `PG_PASSWORD=<postgres-password>` ex: `password` (required)

> [!NOTE]
> Explicitly setting `KUBECONFIG` is not required when using `postgres` as the dex storage backend