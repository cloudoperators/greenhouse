## End-to-End Tests for Greenhouse

- Tests are designed by following the guideline for [E2E Test Framework for Kubernetes](https://github.com/kubernetes-sigs/e2e-framework/blob/v0.3.0/docs/design/README.md)

- `e2e_test.go` is the main entrypoint to run the tests that are defined the other files under the same folder.

### Requirements
- Docker
- [kind](https://kind.sigs.k8s.io/)

##3 Example Run
```
$ go test -v
INFO[0031] Initializing the test environment variables  
INFO[0031] Building docker image: greenhouse:e2e-latest 
INFO[0045] Docker image built: greenhouse:e2e-latest    
INFO[0052] Installing Greenhouse Controller Manager to the central cluster 
INFO[0064] Installing Greenhouse Controller Manager to the central cluster: Done! 
INFO[0064] Deploying organization resource to the central cluster 
INFO[0067] Namespace is created automatically for organization 
=== RUN   TestClusterBootstrap
=== RUN   TestClusterBootstrap/Cluster_onboarding
INFO[0067] Creating temporary file for central cluster kubeconfig (with external access) 
INFO[0067] Exporting kubeconfig file for central cluster 
INFO[0067] Creating kubeconfig secret for cluster       
=== RUN   TestClusterBootstrap/Cluster_onboarding/Cluster_with_ready_status
INFO[0070] Cluster status is ready                      
--- PASS: TestClusterBootstrap (3.10s)
    --- PASS: TestClusterBootstrap/Cluster_onboarding (3.10s)
        --- PASS: TestClusterBootstrap/Cluster_onboarding/Cluster_with_ready_status (3.01s)
PASS
ok      github.com/cloudoperators/greenhouse/test/e2e   71.990s
```