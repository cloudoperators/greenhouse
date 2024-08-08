# Greenhouse end-to-end testing

## Framework

We are using the [k8s sig controller-runtime envtest package](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest).

Test are written in [Ginkgo](https://onsi.github.io/ginkgo/) making use of the [Gomega](https://onsi.github.io/gomega/) matching/assertion library. We use the same framework for our unit and integration tests.

```bash
make e2e
```

will run the e2e test suite without making assumptions on the infrastructure to test against.

Leveraging envtest, we will basically have three different test scenarios. The following env vars steer these:


| Env Var | Meaning |
| --- | --- | 
| `USE_EXISTING_CLUSTER` | If set to `true`, the e2e test suite will not spin up a local apiserver and etcd. Instead, it will expect an existing greenhouse installation on the cluster inferred from the `TEST_KUBECONFIG` environment variable. |
| `TEST_KUBECONFIG`      | Required when `USE_EXISTING_CLUSTER` is `true`. Points to the remote cluster the e2e test suite is running against. |
| `INTERNAL_KUBECONFIG`  | The path to the kubeconfig file for accessing the Greenhouse cluster itself from the running instance. This is used when `USE_EXISTING_CLUSTER` is set to `true`. KIND makes it necessary to set this separately to the `TEST_KUBECONFIG` as the internal api server address differs to the external. Other setups may not use this. If unset `TEST_KUBECONFIG` is used. |

## Run everything local a.k.a. `USE_EXISTING_CLUSTER = false` or unset

Just running the tests via:

```bash
make e2e-local
```

will spin up a local apiserver and etcd together with a local greenhouse controller. The e2e test suite will assert against this setup.

## Run against an existing greenhouse installation a.k.a. `USE_EXISTING_CLUSTER = true`

We can run our e2e test suite against a running greenhouse installation by exposing some env vars (also see [envtest package](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest#pkg-constants)):

```bash
export USE_EXISTING_CLUSTER=true
```

This will stop envtest from spinning up a local apiserver and etcd and expect an existing greenhouse installation on the cluster infered from the set `TEST_KUBECONFIG` environment variable:

```bash
export TEST_KUBECONFIG=/path/to/greenhouse.kubeconfig
```

To run the e2e test suite against a remote installation:

```bash
make e2e-remote
```

Test setup asserts `TEST_KUBECONFIG` is set and working and will fail otherwise.

### Run against a local Greenhouse installation in KIND cluster a.k.a. `USE_EXISTING_CLUSTER = true` and `INTERNAL_KUBECONFIG` set

We provide a convenience method to run the e2e test suite against a local KIND cluster with a greenhouse installation by running:

```bash
make e2e-local-cluster
```

This will spin up a local KIND cluster, install all relevant CRDs, webhooks and the greenhouse controller.
