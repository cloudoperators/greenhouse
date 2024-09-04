# Greenhouse UI Docker Image

This Dockerfile builds a standalone, runnable Greenhouse image that includes all dependent apps, fully integrated within the container. Application properties can be configured either by setting environment variables or by providing a custom appProps.json file through a Docker volume.

## Build the Image

To build the Docker image, navigate to the ui directory and execute the following command:

```bash
cd ui
docker build -t greenhouse -f docker/Dockerfile .
```

This process copies the entire ui folder into the image, clones the greenhouse-extensions repository, and copies the supernova, doop, and heureka apps into the core apps folder. It then builds each app sequentially, creates a manifest file with key-path pairs, and finally starts an Nginx server to serve the application.

## Running the Container

There are two primary methods to run the Greenhouse container:

### 1. Using a Custom `appProps.json` File

To provide a custom configuration file, mount your `appProps.json` via a Docker volume:

```bash
docker run -v /path/to/your/appProps.json:/appProps.json -p 3010:80 greenhouse
```

#### Default appProps.json values:

```json
{
  "currentHost": "origin",
  "apiEndpoint": "https://api.endpoint.com",
  "environment": "prod",
  "authIssuerUrl": "https://auth.endpoint.com",
  "authClientId": "clientID"
}
```

### 2. Using Environment Variables

Alternatively, you can configure the application directly using environment variables:

```bash
docker run -e OIDC_ISSUER_URL="https://oidc.com" -p 3010:80 greenhouse
```

### Combining Both Methods

You can also combine both methods by providing a custom appProps.json via a volume and then overriding specific values using environment variables:

```bash
docker run -v /path/to/your/appProps.json:/appProps.json -p 3010:80 -e THEME="theme-light" greenhouse
```

## Environment Variables

You can customize the Greenhouse application using the following environment variables:

- **`CURRENT_HOST`**: The host for dependent apps, defaults to "origin", indicating that the apps are hosted on the same server.
- **`OIDC_ISSUER_URL`**: The URL of the OIDC issuer.
- **`OIDC_CLIENT_ID`**: The client ID for OIDC authentication.
- **`K8S_API_ENDPOINT`**: The URL for the Kubernetes API endpoint.
- **`ENVIRONMENT`**: The environment setting, with possible values of qa, dev or production (default).
- **`THEME`**: Determines the visual theme of the application. Available options:
  - `theme-light`: Light theme.
  - `theme-dark`: Dark theme.

This setup provides flexibility for running the Greenhouse UI in different environments while ensuring that all necessary applications are seamlessly integrated into a single Docker container.
