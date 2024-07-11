# greenhouse-plugin-admin

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Built with Juno](https://cloudoperators.github.io/juno/built-with-juno.svg)](https://github.com/cloudoperators/juno)

This is the ui for the greenhouse plugin admin screen.

Copy the local [secretProps.template.json](./secretProps.template.json) to `./secretProps.json` to locally inject a `props.endpoint` expecting the k8s api server running on `http://127.0.0.1:8090`.

```
cp secretProps.template.json secretProps.json
```

We propose to use the [greenhouse dev-env](https://github.com/cloudoperators/greenhouse-extensions/tree/main/dev-env) to provide such a setup.

This is easily done with the [docker-compose.yaml](./docker-compose.yaml) provided in `./`:

```
docker-compose up -d
```

This spins up the `dev-env`, bootstrapping [some mock resources](./bootstrap/) to visualize the different aspects of this ui.

Spin up a local instance via:

```
npm install
npm start
```

Frontend is served on [localhost:3000](http://localhost:3000)

Use the following template to point this MFE to a running greenhouse installation:

```json
{
  "endpoint": "https://your-greenhouse-k8s-api-endpoint",
  "environment": "development",
  "appDependencies": {
    "auth": {
      "authIssuerUrl": "https://your-greenhouse-idproxy-url",
      "authClientId": "your-oidc-client-id"
    }
  }
}
```

Start the app on a `port` different to `3000`:

```bash
APP_PORT=<your-port> npm start
```
