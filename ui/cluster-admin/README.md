# greenhouse-cluster-admin

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
][![Built with Juno](https://cloudoperators.github.io/juno/built-with-juno.svg)](https://github.com/cloudoperators/juno)

This is the ui for the greenhouse cluster admin screen.

Spin up a local instance via:

```
npm install
npm start
```

This sets `NODE_ENV=development` which expects an local k8s api server running on `http://127.0.0.1:3005`.

We propose to use the [greenhouse dev-env](https://github.com/cloudoperators/greenhouse-extensions/tree/main/dev-env) to provide such a setup.
Spin up the `dev-env` and the local ui will automatically connect to it's api.
Some k8s resources are bootstrapped into the dev-env to illustrate the working UI.
You can easily bootstrap your own resource to the `dev-env`.

Use the local [docker-compose.yaml](./docker-compose.yaml) to spin up the `dev-env` with some tailored resources for this ui:

```
docker compose up -d
npm start
```

Frontend is served on [localhost:3000](http://localhost:3000)
