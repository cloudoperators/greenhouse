## Running `idproxy` service locally

You can run the `idproxy` service locally by setting the following arguments and environment variables -

## Prerequisites

- Greenhouse Organization with `oidc` authentication in `spec.authentication`

> [!NOTE]
> If you do not have a proper IDP to authenticate with, you can run `keycloak` locally in a docker container
> Or you can use a mock oauth2 server like [ghcr.io/navikt/mock-oauth2-server](https://github.com/navikt/mock-oauth2-server)
> Disclaimer: `keycloak` and `mock-oauth2-server` are not tested with `idproxy` service

Arguments:

- `--listen-addr=:<PORT_NUM>` ex: `--listen-addr=:8085` (defaults to `:8080` if not set)
- `--issuer=<SERVER_URL>` ex: `--issuer=http://localhost:<listen-addr-port>` (required)

Environment Variables:

- If you want to use `kubernetes` as the dex storage backend, the mode is determined by the following settings: 
    1. `KUBECONFIG` environment variable - example - `export KUBECONFIG=/path/to/config` (Priority 1)
    2. kube config from the recommended dir and file - example - `$HOME/.kube/config` (Priority 2)
    3. Running inside a kubernetes cluster, `in-cluster` mode is used. (Priority 3)

- If want to use `postgres` as the dex storage backend, set the following environment variables:
    - `PG_DATABASE=<postgres-database>` ex: `postgres` (defaults to `postgres` if not set)
    - `PG_PORT=<postgres-port>` ex: `5432` (defaults to `5432` if not set)
    - `PG_USER=<postgres-user>` ex: `postgres` (defaults to `postgres` if not set)
    - `PG_HOST=<postgres-host>` ex: `localhost` (required)
    - `PG_PASSWORD=<postgres-password>` ex: `password` (required)

> [!NOTE]
> There should be a configured `Connector` and `OAuth2Client` for the `idproxy` service to work properly. 
> The `Connector` and `OAuth2Client` is created when you create an `Organization`