## Plugin Testing Requirements

All plugins contributed to this repository MUST include comprehensive [Helm chart tests](https://helm.sh/docs/topics/chart_tests/) using the `bats/bats-detik` testing framework in conjunction with standard Helm chart tests. This ensures our plugins are robust, deployable, and catch potential issues early in the development cycle.

**What is bats/bats-detik?**

The [bats/bats-detik](https://github.com/bats-core/bats-detik) framework simplifies end-to-end (e2e) Testing in Kubernetes. It combines the Bash Automated Testing System (`bats`) with Kubernetes-specific assertions (`detik`). This allows you to write test cases using natural language-like syntax, making your tests easier to read and maintain.

**Implementing Tests**

1. **Create a `tests` Directory:** In your Helm chart directory, create a `tests` subdirectory. This directory will contain all the necessary files for your plugin tests.

2. **ConfigMap defnition**:

   - Create a `test-<plugin-name>-config.yaml` file in the `templates/tests` directory to define a ConfigMap that will hold your test script.
   - This ConfigMap contains the test script `run.sh` that will be executed by the test Pod to run your tests.

Example:

```yaml
{{- if .Values.testFramework.enabled -}}
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Release.Name }}-test
  namespace: {{ .Release.Namespace }}
  labels:
    type: integration-test
  annotations:
    "helm.sh/hook": test
    "helm.sh/hook-weight": "-5" # Run before the test pod
    "helm.sh/hook-delete-policy": "before-hook-creation,hook-succeeded"
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

    @test "Verify creation of the prometheus-{{ .Release.Name }} statefulset" {
        verify "there is 1 statefulset named 'prometheus-{{ .Release.Name }}'"
    }

    @test "Verify creation of required custom resource definitions (CRDs) for {{ .Release.Name }}" {
        verify "there is 1 customresourcedefinition named 'prometheuses'"
        verify "there is 1 customresourcedefinition named 'podmonitors'"
    }
{{- end -}}
```

3. **Test Pod Definition**:

   - Create a `test-<plugin-name>.yaml` file in the `templates/tests` directory to define a pod that will run your tests.
   - This test pod will mount the ConfigMap created in the previous step and will execute the test script `run.sh`.

Example:

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
           readOnly: true
   volumes:
     - name: tests
       configMap:
         name: {{ .Release.Name }}-test
   restartPolicy: Never
 {{- end -}}
```

**Note:** You can use this guide to refer for more assertions examples: [bats-detik-examples](https://github.com/bats-core/bats-detik/blob/master/examples/bats/test_kubectl_and_oc.sh).

4. **RBAC Permissions (`test-permissions.yaml`)**:

- Create a `test-permissions.yaml` file in the `templates/tests` directory to define the ServiceAccount and necessary RBAC permissions for the test Pod .
- For an example of how to configure RBAC permissions for your test Pod, please see [test-permissions.yaml](https://github.com/cloudoperators/greenhouse-extensions/blob/main/kube-monitoring/charts/templates/tests/test-permissions.yaml) file in the `kube-monitoring` plugin. The `kube-monitoring` plugin is a good reference for setting up RBAC permissions for your test Pod.

5. **Configure the Test Framework in `values.yaml`**:

   - Add the following configuration to your plugin's `values.yaml` file:

```yaml
testFramework:
enabled: true
image:
  registry: ghcr.io
  repository: cloudoperators/greenhouse-extensions-integration-test
  tag: main
imagePullPolicy: IfNotPresent
```

**Contribution Checklist**

Before submitting a pull request:

- Ensure your plugin's Helm chart includes a `tests` directory.
- Verify the presence of `test-<plugin-name>.yaml`, `test-<plugin-name>-config.yaml`, and `test-permissions.yaml` files.
- Test your plugin thoroughly using `helm test <release-name>` and confirm that all tests pass.
- Include a brief description of the tests in your pull request.

**Important Notes**

- **Test Coverage:** Aim for comprehensive test coverage to ensure your plugin's reliability.
- **Test Isolation:** Design tests that don't interfere with other plugins or production environments.
