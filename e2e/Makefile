SCENARIO ?= cluster
NAME ?= greenhouse-admin

.PHONY: e2e
e2e:
	go test -tags="$(SCENARIO)E2E" ./$(SCENARIO) -coverprofile cover.out -test.v -ginkgo.v

.PHONY: admin-cluster
admin-cluster:
	kind create cluster --name $(NAME)

.PHONY: remote-cluster
remote-cluster:
	kind create cluster --name $(NAME) --config e2e/kind-config.yaml