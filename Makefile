# Image URL to use all building/pushing image targets
IMG ?= ghcr.io/cloudoperators/greenhouse:dev-$(USER)
IMG_LICENSE_EYE ?= ghcr.io/apache/skywalking-eyes/license-eye

MANIFESTS_PATH=$(CURDIR)/charts/manager
CRD_MANIFESTS_PATH=$(MANIFESTS_PATH)/crds
TEMPLATES_MANIFESTS_PATH=$(MANIFESTS_PATH)/templates

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

## Location to install dependencies an GO binaries
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Auto Detect Platform
UNAME_P := $(shell uname -p)
PLATFORM :=
ifeq ($(UNAME_P),x86_64)
	PLATFORM = linux/amd64
endif
ifneq ($(filter arm%,$(UNAME_P)),)
	PLATFORM = linux/arm64
endif

CLI ?= $(LOCALBIN)/greenhousectl

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: generate-all
generate-all: generate generate-manifests generate-documentation  ## Generate code, manifests and documentation.

.PHONY: manifests
manifests: generate-manifests generate-documentation generate-types

.PHONY: generate-manifests
generate-manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) crd paths="./api/..." output:crd:artifacts:config=config/crd/bases
	kustomize build config/crd > $(CRD_MANIFESTS_PATH)/crds.yaml
	(cd $(CRD_MANIFESTS_PATH) && yq -s '(.spec.group | downcase) + "_" + .spec.names.plural' ./crds.yaml --no-doc)
	rm -rf $(CRD_MANIFESTS_PATH)/crds.yaml

	$(CONTROLLER_GEN) rbac:roleName=manager-role webhook paths="./internal/webhook/..." paths="./internal/controller/..." output:artifacts:config=$(TEMPLATES_MANIFESTS_PATH)
	hack/helmify $(TEMPLATES_MANIFESTS_PATH)
	docker run --rm -v $(shell pwd):/github/workspace $(IMG_LICENSE_EYE) -c .github/licenserc.yaml header fix

.PHONY: generate-open-api-spec
generate-open-api-spec: VERSION = main
generate-open-api-spec:
	hack/openapi-generator/generate-openapi-spec-from-crds $(CRD_MANIFESTS_PATH) $(VERSION) docs/reference/api

.PHONY: generate-types
generate-types: generate-open-api-spec## Generate typescript types from CRDs.
	hack/typescript/create-types $(CURDIR)/docs/reference/api/openapi.yaml $(CURDIR)/hack/typescript/metadata.yaml $(CURDIR)/types/typescript/

.PHONY: actiongenerate
actiongenerate: action-controllergen
	$(CONTROLLER_GEN_ACTION) object:headerFile="hack/boilerplate.go.txt" paths="./api/..."
	$(CONTROLLER_GEN_ACTION) object:headerFile="hack/boilerplate.go.txt" paths="./internal/dex/..."

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./api/..."
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./internal/dex/..."

# Default values
GEN_DOCS_API_DIR ?= "./api/greenhouse/v1alpha1" ## -app-dir should be Canonical Path Format so absolute path doesn't work. That's why we don't use $(CURDIR) here.
GEN_DOCS_CONFIG ?= "$(CURDIR)/hack/docs-generator/config.json"
GEN_DOCS_TEMPLATE_DIR ?= "$(CURDIR)/hack/docs-generator/templates"
GEN_DOCS_OUT_FILE ?= "$(CURDIR)/docs/reference/api/index.html"
GEN_CRD_API_REFERENCE_DOCS := $(CURDIR)/hack/docs-generator/gen-crd-api-reference-docs # Define the path to the gen-crd-api-reference-docs binary
.PHONY: check-gen-crd-api-reference-docs
check-gen-crd-api-reference-docs:
	@if [ ! -f $(GEN_CRD_API_REFERENCE_DOCS) ]; then \
		echo "gen-crd-api-reference-docs not found, installing..."; \
		GOBIN=$(LOCALBIN) go install github.com/ahmetb/gen-crd-api-reference-docs@latest; \
	fi

GEN_DOCS ?= $(LOCALBIN)/gen-crd-api-reference-docs
.PHONY: generate-documentation
generate-documentation: check-gen-crd-api-reference-docs
	$(GEN_DOCS) -api-dir=$(GEN_DOCS_API_DIR) -config=$(GEN_DOCS_CONFIG) -template-dir=$(GEN_DOCS_TEMPLATE_DIR) -out-file=$(GEN_DOCS_OUT_FILE)

.PHONY: test
test: generate-manifests generate envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out -v

.PHONY: fmt
fmt: goimports
	GOBIN=$(LOCALBIN) go fmt ./...
	$(GOIMPORTS) -w -local github.com/cloudoperators/greenhouse .

.PHONY: lint
lint: golint
	$(GOLINT) run -v --timeout 5m	

.PHONY: check
check: fmt lint test

##@ Build CLI Locally
.PHONY: cli
cli: $(CLI)
$(CLI): $(LOCALBIN)
	test -s $(LOCALBIN)/greenhousectl || echo "Building Greenhouse CLI..." && make build-greenhousectl

##@ Build
.PHONY: action-build
action-build: build-greenhouse build-idproxy build-cors-proxy build-greenhousectl build-service-proxy

.PHONY: build
build: generate build-greenhouse build-idproxy build-cors-proxy build-greenhousectl build-service-proxy

build-%: GIT_BRANCH  = $(shell git rev-parse --abbrev-ref HEAD)
build-%: GIT_COMMIT  = $(shell git rev-parse --short HEAD)
build-%: GIT_STATE   = $(shell if git diff --quiet; then echo clean; else echo dirty; fi)
build-%: BUILD_DATE  = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
build-%:
	go build -ldflags "-s -w -X github.com/cloudoperators/greenhouse/internal/version.GitBranch=$(GIT_BRANCH) -X github.com/cloudoperators/greenhouse/internal/version.GitCommit=$(GIT_COMMIT) -X github.com/cloudoperators/greenhouse/internal/version.GitState=$(GIT_STATE) -X github.com/cloudoperators/greenhouse/internal/version.BuildDate=$(BUILD_DATE)" -o bin/$* ./cmd/$*/

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/greenhouse/

.PHONY: docker-build
docker-build:
	docker build --platform ${PLATFORM} -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: kustomize-build-crds
kustomize-build-crds: generate-manifests kustomize
	$(KUSTOMIZE) build $(CRD_MANIFESTS_PATH)
	
##@ Build Dependencies

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
CONTROLLER_GEN_ACTION ?= $(LOCALBIN)/controller-gen
GOIMPORTS ?= $(LOCALBIN)/goimports
GOLINT ?= $(LOCALBIN)/golangci-lint
ENVTEST ?= $(LOCALBIN)/setup-envtest
ENVTEST_ACTION ?= $(LOCALBIN)/setup-envtest
HELMIFY ?= $(LOCALBIN)/helmify

## Tool Versions
KUSTOMIZE_VERSION ?= 5.6.0
CONTROLLER_TOOLS_VERSION ?= 0.17.2
GOLINT_VERSION ?= 1.64.8
GINKGOLINTER_VERSION ?= 0.19.1
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION ?= 1.31.0

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN)

.PHONY: action-controllergen
action-controllergen:: $(CONTROLLER_GEN_ACTION) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN_ACTION):: $(LOCALBIN)
	GOMODCACHE=$(shell pwd)/tmp GOPATH=$(shell pwd) go install -modcacherw sigs.k8s.io/controller-tools/cmd/controller-gen@v$(CONTROLLER_TOOLS_VERSION)
	GOMODCACHE=$(shell pwd)/tmp go clean -modcache
	rm -rf $(shell pwd)/pkg/sumdb/

.PHONY: controller-gen
controller-gen:: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN):: $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@v$(CONTROLLER_TOOLS_VERSION)

.PHONY: action-envtest
action-envtest:: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST_ACTION):: $(LOCALBIN)
	GOMODCACHE=$(shell pwd)/tmp GOPATH=$(shell pwd) go install -modcacherw sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
	GOMODCACHE=$(shell pwd)/tmp go clean -modcache
	rm -rf $(shell pwd)/pkg/sumdb/

.PHONY: envtest
envtest:: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST):: $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: goimports
goimports: $(GOIMPORTS)
$(GOIMPORTS): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install golang.org/x/tools/cmd/goimports@latest

.PHONY: golint
golint: $(GOLINT)
$(GOLINT): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@v$(GOLINT_VERSION)
	GOBIN=$(LOCALBIN) go install github.com/nunnatsa/ginkgolinter/cmd/ginkgolinter@v$(GINKGOLINTER_VERSION)

.PHONY: serve-docs
serve-docs: generate-manifests
ifeq (, $(shell which hugo))
	@echo "Hugo is not installed in your machine. Please install it to serve the documentation locally. Please refer to https://gohugo.io/installation/ for installation instructions."
else
	cd website && hugo server
endif

SCENARIO ?= cluster
ADMIN_CLUSTER ?= greenhouse-admin
REMOTE_CLUSTER ?= greenhouse-remote
EXECUTION_ENV ?= LOCAL
ADMIN_NAMESPACE ?= greenhouse
ADMIN_RELEASE ?= greenhouse
ADMIN_CHART_PATH ?= charts/manager
E2E_REPORT_PATH="$(shell pwd)/bin/$(SCENARIO)-e2e-report.json"
PLUGIN_DIR ?=
GREENHOUSE_ORG ?= demo
WEBHOOK_ONLY ?= false
DEV_MODE ?= false
INTERNAL ?= -int

.PHONY: setup
setup: cli setup-manager setup-dashboard setup-demo

.PHONY: setup-webhook-dev
setup-webhook-dev:
	DEV_MODE=true make setup-manager

.PHONY: setup-controller-dev
setup-controller-dev:
	WEBHOOK_ONLY=true make setup-manager && INTERNAL= make setup-demo

.PHONY: setup-manager
setup-manager: cli
	PLUGIN_PATH=$(PLUGIN_DIR) $(CLI) dev setup -f dev-env/dev.config.yaml d=$(DEV_MODE) e=WEBHOOK_ONLY=$(WEBHOOK_ONLY)

.PHONY: setup-dashboard
setup-dashboard: cli
	$(CLI) dev setup dashboard -f dev-env/ui.config.yaml

.PHONY: setup-demo
setup-demo: prepare-e2e samples
	kubectl create secret generic kind-$(REMOTE_CLUSTER) \
		--from-literal=kubeconfig="$$(cat ${PWD}/bin/$(REMOTE_CLUSTER)$(INTERNAL).kubeconfig)" \
		--namespace=$(GREENHOUSE_ORG) \
		--type="greenhouse.sap/kubeconfig" \
		--dry-run=client -o yaml | kubectl apply -f -

.PHONY: samples
samples: kustomize
	$(KUSTOMIZE) build dev-env/samples | kubectl apply -n $(GREENHOUSE_ORG) --kubeconfig=$(shell pwd)/bin/$(ADMIN_CLUSTER).kubeconfig -f -
	while true; do \
		if kubectl get organizations $(GREENHOUSE_ORG) --kubeconfig=$(shell pwd)/bin/$(ADMIN_CLUSTER).kubeconfig -o json | \
			jq -e '.status.statusConditions.conditions[] | select(.type == "Ready") | select(.status == "True")' > /dev/null; then \
			echo "Organization is ready"; \
			exit 0; \
		fi; \
		sleep 5; \
	done

.PHONY: setup-e2e
setup-e2e: cli
	$(CLI) dev setup -f e2e/config.yaml
	make prepare-e2e

.PHONY: clean-e2e
clean-e2e:
	kind delete cluster --name $(REMOTE_CLUSTER)
	kind delete cluster --name $(ADMIN_CLUSTER)
	rm -v $(LOCALBIN)/*.kubeconfig

.PHONY: e2e
e2e:
	GOMEGA_DEFAULT_EVENTUALLY_TIMEOUT="2m" \
		go test -tags="$(SCENARIO)E2E" ${PWD}/e2e/$(SCENARIO) -mod=readonly -test.v -ginkgo.v --ginkgo.json-report=$(E2E_REPORT_PATH)

.PHONY: e2e-local
e2e-local: prepare-e2e
	GREENHOUSE_ADMIN_KUBECONFIG="$(shell pwd)/bin/$(ADMIN_CLUSTER).kubeconfig" \
    	GREENHOUSE_REMOTE_KUBECONFIG="$(shell pwd)/bin/$(REMOTE_CLUSTER).kubeconfig" \
    	GREENHOUSE_REMOTE_INT_KUBECONFIG="$(shell pwd)/bin/$(REMOTE_CLUSTER)-int.kubeconfig" \
    	CONTROLLER_LOGS_PATH="$(shell pwd)/bin/$(SCENARIO)-e2e-pod-logs.txt" \
    	EXECUTION_ENV=$(EXECUTION_ENV) \
		GOMEGA_DEFAULT_EVENTUALLY_TIMEOUT="2m" \
		go test -tags="$(SCENARIO)E2E" $(shell pwd)/e2e/$(SCENARIO) -test.v -ginkgo.v --ginkgo.json-report=$(E2E_REPORT_PATH)

.PHONY: prepare-e2e
prepare-e2e:
	kind get kubeconfig --name $(ADMIN_CLUSTER) > $(shell pwd)/bin/$(ADMIN_CLUSTER).kubeconfig
	kind get kubeconfig --name $(REMOTE_CLUSTER) > $(shell pwd)/bin/$(REMOTE_CLUSTER).kubeconfig
	kind get kubeconfig --name $(REMOTE_CLUSTER) --internal > ${PWD}/bin/$(REMOTE_CLUSTER)-int.kubeconfig

.PHONY: list-scenarios
list-scenarios:
	find $(shell pwd)/e2e -type f -name 'e2e_test.go' -exec dirname {} \; | xargs -n 1 basename | jq -R -s -c 'split("\n")[:-1]'

.PHONY: dev-docs
dev-docs:
	go run -tags="dev" -mod=mod dev-env/docs.go

# Download and install mockery locally via `brew install mockery`
MOCKERY := $(shell which mockery)
mockery:
	# will look into .mockery.yaml for configuration
	$(MOCKERY)
