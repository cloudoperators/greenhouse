# greenhouse-plugin-admin

This is the ui for the greenhouse plugin overview screen.

Spin up a local instance via:

```
npm install
npm start
```

This sets `NODE_ENV=development` which expects an local k8s api server running on `http://127.0.0.1:8090`.

We propose to use the [greenhouse dev-env](https://github.com/cloudoperators/greenhouse-extensions/tree/main/dev-env) to provide such a setup.
Spin up the `dev-env` and the local ui will automatically connect to it's api.
Some k8s resources are bootstrapped into the dev-env to illustrate the working UI.
You can easily bootstrap your own resource to the `dev-env`.

Have a look at the [docker-compose.yaml](./docker-compose.yaml) and use it to bootstrap some resources necessary for mocking this UI.

Frontend is served on [localhost:3001](http://localhost:3001)
