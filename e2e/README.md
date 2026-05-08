## Greenhouse E2E Tests

Greenhouse E2E tests are self-contained tests that can run against local k8s clusters such as KinD, K3D etc. or against
real k8s clusters.

We recommend using KinD clusters for local development and testing as it provides an easy way to communicate between two
`docker` containers without the need for additional networking.

The tests are isolated based on scenarios where each scenario targets a controller in Greenhouse.

The tests are written using Ginkgo framework.

## Writing E2E Tests

To write an E2E test, create a new folder in `e2e` directory in the root of the project.

The folder name should be `CLI` friendly and should be easily identifiable as to what controller it is targeting.

For example, if the e2e test is for `Cluster` Onboarding, the folder name could be `cluster`.

Inside the folder, create a new file with the name `e2e_test.go`.

The test file should have build tags starting with the name of the folder, followed by `E2E` in uppercase

Example:

```go
//go:build clusterE2E

package cluster
...

```

Register the test suite in `e2e_test.go`

Example:

```go
func TestE2e(t *testing.T) {
RegisterFailHandler(Fail)
RunSpecs(t, "Cluster E2E Suite")
}
```

Ensure you have a `BeforeSuite` to setup the test environment

Example:

```go
//go:build clusterE2E

package cluster

import (
...
"github.com/cloudoperators/greenhouse/e2e/shared"
...
)

var (
	env              *shared.TestEnv
	ctx              context.Context
	adminClient      client.Client
	remoteClient     client.Client
	remoteRestClient *clientutil.RestClientGetter
	testStartTime    time.Time
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cluster E2E Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx = context.Background()
	env = shared.NewExecutionEnv(greenhousev1alpha1.AddToScheme).WithOrganization(ctx, "./testdata/organization.yaml")
	adminClient = env.GetClient(shared.AdminClient)
	Expect(adminClient).ToNot(BeNil(), "admin client should not be nil")
	remoteClient = env.GetClient(shared.RemoteClient)
	Expect(remoteClient).ToNot(BeNil(), "remote client should not be nil")
	remoteRestClient = env.GetRESTClient(shared.RemoteRESTClient)
	Expect(remoteRestClient).ToNot(BeNil(), "remote rest client should not be nil")
	testStartTime = time.Now().UTC()
})
```

Once you have finished your test `Describe` block, ensure you have an `AfterSuite` to teardown the test environment

It is recommended and best practice to tear down the resources created during the test, so that the test environment is
clean for the next test run.

This is very helpful when running the tests locally and very important when running the tests against real k8s clusters.

Example:

```go
var _ = AfterSuite(func () {
shared.OffBoardRemoteCluster(ctx, adminClient, remoteClient, testStartTime, remoteClusterName, env.TestNamespace)
env.GenerateGreenhouseControllerLogs(ctx, testStartTime)
})
```

If the controller is using flux resources, you can output the logs for flux controllers as well by using the following function:

```go
env.GenerateFluxControllerLogs(ctx context.Context, controllerName string, startTime time.Time)
````

Example:

```go
env.GenerateFluxControllerLogs(ctx, "kustomize-controller", testStartTime)
env.GenerateFluxControllerLogs(ctx, "source-controller", testStartTime)
env.GenerateFluxControllerLogs(ctx, "helm-controller", testStartTime)
```

You can take a look at helper functions available in [shared](./shared), which is meant to be the common place to
have reusable functions across all e2e tests.

### Running E2E Tests Locally

1. Setup KinD clusters and greenhouse operator by running `make setup-e2e` (Skip if you already have a running version)
2. Run the tests by executing
   `make e2e-local SCENARIO=<scenario> ADMIN_CLUSTER=<admin-cluster-name> REMOTE_CLUSTER=<remote-cluster-name>`

```shell

If you already have 2 k8s clusters running locally, you can run the tests using the following command:

```shell
make e2e-local SCENARIO=cluster ADMIN_CLUSTER=<kind-cluster-name> REMOTE_CLUSTER=<kind-cluster-name>
```

> [!NOTE]  
> There sane defaults provided in the `Makefile` for `SCENARIO`, `ADMIN_CLUSTER` and `REMOTE_CLUSTER`
> if you are using defaults then you only need to execute `make e2e-local SCENARIO=<scenario-name>`
> The `ADMIN_CLUSTER` is the cluster that has Greenhouse CRDs installed and the controller manager running.
> The `REMOTE_CLUSTER` is the cluster that is being onboarded to the admin cluster to manage Greenhouse resources such
> as `Plugins`, `RBACs` etc.

> [!WARNING]
> If you have made changes to operator code and want to run the tests, you need to re-execute `make setup-e2e` to
> ensure the operator image is built with the latest changes and loaded into the KinD clusters.

### Running E2E Tests Against Real K8s Clusters

You need to set the following environment variables to run the tests against real k8s clusters:

```shell
# This kubeconfig will be used to communicate with the admin cluster, where the Greenhouse CRDs are installed and the controller manager is running
export GREENHOUSE_ADMIN_KUBECONFIG=<path-to-kubeconfig>

# This kubeconfig will be used to communicate with the remote cluster, which is being onboarded to the admin cluster
export GREENHOUSE_REMOTE_KUBECONFIG=<path-to-kubeconfig>

# The Execution Env must be "GARDENER" when running against real k8s clusters otherwise the tests will fail as it will attempt to run the tests against KinD clusters
export EXECUTION_ENV=GARDENER
```

(Optional) if you want to get pod logs for the controller manager and webhooks, you can set the following environment
variables:

```shell
# This will save the controller logs to the specified path, ensure the path is writable and directory exists
export CONTROLLER_LOGS_PATH=<path-to-save-the-file>
```

> [!NOTE]
> The provided GREENHOUSE_ADMIN_KUBECONFIG should have permissions to read pod logs

Once the environment variables are set, you can run the tests using the following command:

```shell
make e2e SCENARIO=cluster
```

### E2E Workflow

You can find the E2E workflow in the `.github/workflows/ci-e2e-test.yaml`

The workflow detects e2e scenarios based on the folder names in the `e2e` directory, which contain `e2e_test.go` file (
otherwise ignores the dir)
and generates a matrix for the scenarios to run against `KinD` clusters targeting different kubernetes versions for
remote cluster

It will be multi-matrix workflow where each scenario will run the tests against a remote cluster with a different k8s
version.

[E2E Workflow in Action](https://github.com/cloudoperators/greenhouse/actions/workflows/ci-e2e-test.yaml)

### Debugging E2E Tests

You can use standard Go debugging tools to debug the tests.

Example (Goland):

1. Go to Goland settings and select `Go` -> `Build Tags`
2. In the `Custom tags` field, add the build tags for the test you want to debug (e.g. `clusterE2E` - space separated
   from other tags)
3. Right-click on the test file, select `More Run/Debug` -> `Modify Run Configuration`
4. Set the environment variables and start debugging

### Tips

Pod logs for the controller manager and webhooks can be very long and can be difficult to read especially when you are
running multiple tests.
If you want to get logs for a specific test and you have set the `CONTROLLER_LOGS_PATH` environment variable, you can
extract the logs for a specific time.

Example:

```go
//go:build clusterE2E

package cluster

import (
...)

var (
	env              *e2e.TestEnv
	ctx              context.Context
	adminClient      client.Client
	remoteClient     client.Client
	remoteRestClient *clientutil.RestClientGetter
	testStartTime    time.Time // variable to set the test start time
)

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx = context.Background()
	env = e2e.NewExecutionEnv(greenhousev1alpha1.AddToScheme).WithOrganization(ctx, "./testdata/organization.yaml")
	adminClient = env.GetClient(e2e.AdminClient)
	remoteClient = env.GetClient(e2e.RemoteClient)
	remoteRestClient = env.GetRESTClient(e2e.RemoteRESTClient)
	testStartTime = time.Now().UTC() // set the test start time in BeforeSuite
})

var _ = AfterSuite(func() {
	expect.OffBoardRemoteCluster(ctx, adminClient, remoteClient, testStartTime, expect.RemoteClusterName, env.TestNamespace)
	env.GenerateControllerLogs(ctx, testStartTime) // use env.GenerateControllerLogs to extract logs from the controller manager since the test start time
})
```

This is not foolproof as there could be reconciliations happening on different resources, especially in a real cluster,
but it can help in narrowing down the logs to a specific test run.

**Eventually Timeout**

If `Eventually` is used in tests, ensure that the result being expected happens quickly.

> The test is started with the env GOMEGA_DEFAULT_EVENTUALLY_TIMEOUT="2m"

If some result is expected to happen, but takes longer than 2m, the timeout for `Eventually` can be increased by passing
the timeout value as the first argument.

Example:

```go
Eventually(func () bool {
// your code here
}, 150*time.Second, 10*time.Second).Should(BeTrue())
```

This will wait for 150 seconds for the condition to be true, polling every 10 seconds.

Alternatively, you can use `WaitUntilResourceReadyOrNotReady` in [shared](./shared/e2e.go) to wait for a resource to be
ready or not ready.
`WaitUntilResourceReadyOrNotReady` has a timeout of 3 minutes with exponential backoff and will wait for a resource to
be ready or not ready.

> Note: You can only use `WaitUntilResourceReadyOrNotReady` for resources that use the `lifecycle.Reconcile` interface.
