---
title: "Testing a Plugin"
linkTitle: "Testing a Plugin"
landingSectionIndex: false
weight: 2
description: >
  Guidelines for testing plugins contributed to the Greenhouse project.
---
## Overview
![Plugin Test architecture](../plugin-chart-test-drawing.png)
## Plugin Testing Requirements

All plugins contributed to [plugin-extensions](https://github.com/cloudoperators/greenhouse-extensions) repository should include comprehensive [Helm Chart Tests](https://helm.sh/docs/topics/chart_tests/) using the `bats/bats-detik` testing framework. This ensures our plugins are robust, deployable, and catch potential issues early in the development cycle.

**What is bats/bats-detik?**

The [bats/bats-detik](https://github.com/bats-core/bats-detik) framework simplifies end-to-end (e2e) Testing in Kubernetes. It combines the Bash Automated Testing System (`bats`) with Kubernetes-specific assertions (`detik`). This allows you to write test cases using natural language-like syntax, making your tests easier to read and maintain.

**Implementing Tests**

1. Create a `/tests` folder inside your Plugin's Helm Chart `templates` folder to store your test resources.

2. **ConfigMap definition**:

   - Create a `test-<plugin-name>-config.yaml` file in the `templates/tests` directory to define a `ConfigMap` that will hold your test script.
   - This `ConfigMap` contains the test script `run.sh` that will be executed by the test `Pod` to run your tests.

```yaml
{{- if .Values.testFramework.enabled -}}
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Release.Name }}-test
  namespace: {{ .Release.Namespace }}
  labels:
    type: integration-test
data:
  run.sh: |-

    #!/usr/bin/env bats

    load "/usr/lib/bats/bats-detik/utils"
    load "/usr/lib/bats/bats-detik/detik"

    DETIK_CLIENT_NAME="kubectl"

    @test "Verify successful deployment and running status of the {{ .Release.Name }}-operator pod" {
        verify "there is 1 deployment named '{{ .Release.Name }}-operator'"
        verify "there is 1 service named '{{ .Release.Name }}-operator'"
        try "at most 2 times every 5s to get pods named '{{ .Release.Name }}-operator' and verify that '.status.phase' is 'running'"
    }

    @test "Verify successful creation and bound status of {{ .Release.Name }} persistent volume claims" {
        try "at most 3 times every 5s to get persistentvolumeclaims named '{{ .Release.Name }}.*' and verify that '.status.phase' is 'Bound'"
    }

    @test "Verify successful creation and available replicas of {{ .Release.Name }} Prometheus resource" {
        try "at most 3 times every 5s to get prometheuses named '{{ .Release.Name }}' and verify that '.status.availableReplicas' is more than '0'"
    }

    @test "Verify creation of required custom resource definitions (CRDs) for {{ .Release.Name }}" {
        verify "there is 1 customresourcedefinition named 'prometheuses'"
        verify "there is 1 customresourcedefinition named 'podmonitors'"
    }
{{- end -}}
```

> **Note:** You can use [this guide](https://github.com/bats-core/bats-detik/blob/master/examples/bats/test_kubectl_and_oc.sh) for reference when writing your test assertions.

3. **Test Pod Definition**:

   - Create a `test-<plugin-name>.yaml` file in the `templates/tests` directory to define a `Pod` that will run your tests.
   - This test `Pod` will mount the `ConfigMap` created in the previous step and will execute the test script `run.sh`.

```yaml
{{- if .Values.testFramework.enabled -}}
apiVersion: v1
kind: Pod
metadata:
  name: {{ .Release.Name }}-test
  namespace: {{ .Release.Namespace }}
  labels:
    type: integration-test
  annotations:
    "helm.sh/hook": test
    "helm.sh/hook-delete-policy": "before-hook-creation,hook-succeeded"
spec:
  serviceAccountName: {{ .Release.Name }}-test
  containers:
    - name: bats-test
      image: "{{ .Values.testFramework.image.registry}}/{{ .Values.testFramework.image.repository}}:{{ .Values.testFramework.image.tag }}"
      imagePullPolicy: {{ .Values.testFramework.image.pullPolicy }}
      command: ["bats", "-t", "/tests/run.sh"]
      volumeMounts:
        - name: tests
          mountPath: /tests
          readOnly: true   volumes:
    - name: tests
      configMap:
        name: {{ .Release.Name }}-test
  restartPolicy: Never
{{- end -}}
```

4. **RBAC Permissions**:

- Create the necessary RBAC resources in the `templates/tests` folder with a dedicated `ServiceAccount` and role authorisations so that the test `Pod` can cover test the cases.
- You can use [test-permissions.yaml](https://github.com/cloudoperators/greenhouse-extensions/blob/main/kube-monitoring/charts/templates/tests/test-permissions.yaml) from the `kube-monitoring` as a reference to configure RBAC permissions for your test Pod.

5. **Configure the Test Framework in Plugin's `values.yaml`**:
   - Add the following configuration to your Plugin's `values.yaml` file:

```yaml
testFramework:
  enabled: true
  image:
    registry: ghcr.io
    repository: cloudoperators/greenhouse-extensions-integration-test
    tag: main
  imagePullPolicy: IfNotPresent
```

6. **Running the Tests**:

> **Important:** Once you have completed all the steps above, you are ready to run the tests. However, before running the tests, ensure that you perform a fresh Helm installation or upgrade of your Plugin's Helm release against your test Kubernetes cluster (for example, Minikube or Kind) by executing the following command:

```yaml
# For a new installation
helm install <Release name> <chart-path>

# For an upgrade
helm upgrade <Release name> <chart-path>
```

- After the Helm installation or upgrade is successful, run the tests against the same test Kubernetes cluster by executing the following command.

```yaml
helm test <Release name>
```

## Plugin Testing with dependencies during Pull Requests
### Overview

Some plugins require other plugins to be installed in the cluster for their tests to run successfully. To support this, each plugin can declare required dependencies using a `test-dependencies.yaml` file.

> [!NOTE]  
>The `test-dependencies.yaml` file is required if other plugins need to be installed in the Kind cluster created by the GitHub Actions workflow before running tests during a Pull Request for the plugin.


### How It Works
- Each plugin can optionally include a `test-dependencies.yaml` file in the plugin’s root directory (e.g., `thanos/test-dependencies.yaml`).
- This file defines both the dependencies (other plugins) that should be installed before testing begins and custom values for these dependencies.

### Example `test-dependencies.yaml`
```yaml
dependencies:
  - kube-monitoring
values:
  kubeMonitoring:
    prometheus:
      enabled: true
      serviceMonitor:
        enabled: false
      prometheusSpec:
        thanos:
          objectStorageConfig:
            secret:
              type: FILESYSTEM
              config:
                directory: "/test"
              prefix: ""
```

In this example, the plugin:
- Declares `kube-monitoring` as a dependency that must be installed first
- Provides custom values for this dependent plugin, specifically configuring Prometheus settings

### Dependecy Structure
The `test-dependencies.yaml` file supports:

- **dependencies**: A list of plugin names that should be installed before testing the current plugin.
- **values**: Custom configuration values to be applied when installing dependencies
  
### Automation during Pull Requests
The GitHub Actions workflow automatically:

1. Detects plugins that are changed in the pull request.
2. Parses the `test-dependencies.yaml` for each changed plugin if present.
3. Installs listed dependencies in order
4. Proceeds with helm chart linting and testing

## Testing Values Configuration

### Parent Plugin Configuration
- A plugin may optionally provide a `<PLUGIN-NAME>/ci/test-values.yaml` file
- The GitHub Actions workflow will use this values file for testing the plugin if it exists
- This allows you to customize values specifically for CI testing, without modifying the default `values.yaml`

### Dependent Plugin Configuration
- Values for dependent plugins should be specified in the values section of your plugin's `test-dependencies.yaml` file.
- This allows you to customize the configuration of dependent plugins when they are installed for testing.
- The values specified in the `test-dependencies.yaml` file will override the default values of the dependent plugins.

### Example File Structure:
```
alert/
├── charts/
├── ci/
│   └── test-values.yaml
└── test-dependencies.yaml
```

**Contribution Checklist**

Before submitting a pull request:

- Ensure your Plugin's Helm Chart includes a `/tests` directory.
- Verify the presence of `test-<plugin-name>.yaml`, `test-<plugin-name>-config.yaml`, and `test-permissions.yaml` files.
- Test your Plugin thoroughly using `helm test <release-name>` and confirm that all tests pass against a test Kubernetes cluster.
- Include a brief description of the tests in your pull request.
- Make sure that your Plugin's Chart Directory and the Plugin's Upstream Chart Repository are added to this [greenhouse-extensions helm test config file](https://github.com/cloudoperators/greenhouse-extensions/blob/main/.github/configs/helm-test.yaml). This will ensure that your Plugin's tests are automatically run in the GitHub Actions workflow when you submit a pull request for this Plugin.
- Note that the [dependencies](https://helm.sh/docs/helm/helm_dependency/) of your Plugin's helm chart might also have their own tests. If so, ensure that the tests of the dependencies are also passing.
- If your plugin relies on other plugins for testing, please follow the [Plugin Testing with dependencies](#plugin-testing-with-dependencies) section for declaring those dependencies.

**Important Notes**

- **Test Coverage:** Aim for comprehensive test coverage to ensure your Plugin's reliability.
