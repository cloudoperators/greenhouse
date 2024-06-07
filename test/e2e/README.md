# Greenhouse end-to-end testing

## Framework

We are using the [k8s sig controller-runtime envtest package](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest).

Test are written in [Ginkgo](https://onsi.github.io/ginkgo/) making use of the [Gomega](https://onsi.github.io/gomega/) matching/assertion library. We use the same framework for our unit and integration tests.

```bash
make e2e
```

will run the e2e test suite without making assumptions on the infrastructure to test against.

Leveraging envtest, we will have basically two different test scenarios:

## Run everything local

Just running the tests via:

```bash
make e2e-local
```

will spin up a local apiserver and etcd together with a local greenhouse controller. The e2e test suite will assert against this setup.

## Run against an existing greenhouse installation

We can run our e2e test suite against a running greenhouse installation by exposing some env vars (also see [envtest package](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest#pkg-constants)):

```bash
export USE_EXISTING_CLUSTER=true
```

This will stop envtest from spinning up a local apiserver and etcd and expect an existing greenhouse installation on the cluster infered from the set `KUBECONFIG` environment variable:

```bash
export KUBECONFIG=/path/to/greenhouse.kubeconfig
```

To run the e2e test suite against a remote installation:

```bash
make e2e-remote
```

Test setup asserts `KUBECONFIG` is set and working and will fail otherwise.

### Under construction:

We provide a convenience method to run the e2e test suite against a local KIND cluster with a greenhouse installation by running:

```bash
make e2e-local-cluster
```

This will spin up a local KIND cluster, install all relevant CRDs, webhooks and the greenhouse controller.
