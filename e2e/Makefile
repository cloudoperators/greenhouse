SCENARIO ?= cluster
NAME ?=
ADMIN_CLUSTER ?=greenhouse-admin
REMOTE_CLUSTER ?=greenhouse-remote
EXECUTION_ENV ?= LOCAL

.PHONY: e2e-local
e2e-local: prepare
	GREENHOUSE_ADMIN_KUBECONFIG="${PWD}/../bin/$(ADMIN_CLUSTER).kubeconfig" \
    	GREENHOUSE_REMOTE_KUBECONFIG="${PWD}/../bin/$(REMOTE_CLUSTER).kubeconfig" \
    	GREENHOUSE_REMOTE_INT_KUBECONFIG="${PWD}/../bin/$(REMOTE_CLUSTER)-int.kubeconfig" \
    	CONTROLLER_LOGS_PATH="${PWD}/../bin/$(SCENARIO)-e2e-pod-logs.txt" \
    	EXECUTION_ENV=$(EXECUTION_ENV) \
		go test -tags="$(SCENARIO)E2E" ./$(SCENARIO) -test.v -ginkgo.v

.PHONY: e2e
e2e:
	go test -tags="$(SCENARIO)E2E" ./$(SCENARIO) -test.v -ginkgo.v

.PHONY: admin-cluster
admin-cluster:
	kind create cluster --name $(ADMIN_CLUSTER)

.PHONY: remote-cluster
remote-cluster:
	kind create cluster --name $(REMOTE_CLUSTER) --config kind-config.yaml

.PHONY: delete-cluster
delete-cluster:
	kind delete cluster --name $(NAME)

.PHONY: prepare
prepare:
	kind get kubeconfig --name $(ADMIN_CLUSTER) > ../bin/$(ADMIN_CLUSTER).kubeconfig
	kind get kubeconfig --name $(REMOTE_CLUSTER) > ../bin/$(REMOTE_CLUSTER).kubeconfig
	kind get kubeconfig --name $(REMOTE_CLUSTER) --internal > ../bin/$(REMOTE_CLUSTER)-int.kubeconfig