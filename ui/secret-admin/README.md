# greenhouse-secrets

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

This is the ui for the greenhouse secrets admin screen.

Copy the local [secretProps.template.json](./secretProps.template.json) to `./secretProps.json` to locally inject a `props.endpoint` expecting the k8s api server running on `http://127.0.0.1:8090`.

```
cp secretProps.template.json secretProps.json
```

We propose to use the [greenhouse dev-env](https://github.com/cloudoperators/greenhouse-extensions/tree/main/dev-env) to provide such a setup.

This is easily done with the [docker-compose.yaml](./docker-compose.yaml) provided in `./`:

```
docker compose up -d
```

This spins up the `dev-env`, bootstrapping [some mock resources](./bootstrap/) to visualize the different aspects of this ui.

Spin up a local instance via:

```
npm install
npm start
```

Frontend is served on [localhost:3000](http://localhost:3000)
