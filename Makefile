# Image URL to use all building/pushing image targets
IMG ?= ghcr.io/cloudoperators/greenhouse:dev-$(USER)
IMG_DEV_ENV ?= ghcr.io/cloudoperators/greenhouse-dev-env:dev-$(USER)
IMG_LICENSE_EYE ?= ghcr.io/apache/skywalking-eyes/license-eye
PLATFORM ?=linux/arm64

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.30.3

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
	$(CONTROLLER_GEN) crd paths="./pkg/apis/..." output:crd:artifacts:config=$(CRD_MANIFESTS_PATH)
	$(CONTROLLER_GEN) rbac:roleName=manager-role webhook paths="./pkg/admission/..." paths="./pkg/controllers/..." output:artifacts:config=$(TEMPLATES_MANIFESTS_PATH)
	hack/helmify $(TEMPLATES_MANIFESTS_PATH)
	docker run --rm -v $(shell pwd):/github/workspace $(IMG_LICENSE_EYE) -c .github/licenserc.yaml header fix

.PHONY: generate-open-api-spec
generate-open-api-spec: VERSION = $(shell git rev-parse --short HEAD)
generate-open-api-spec:
	hack/openapi-generator/generate-openapi-spec-from-crds $(CRD_MANIFESTS_PATH) $(VERSION) docs/reference/api

.PHONY: generate-types
generate-types: generate-open-api-spec## Generate typescript types from CRDs.
	hack/typescript/create-types $(CURDIR)/docs/reference/api/openapi.yaml $(CURDIR)/hack/typescript/metadata.yaml $(CURDIR)/ui/types/ 

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./pkg/apis/..."
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./pkg/dex/..."

# Default values
GEN_DOCS_API_DIR ?= "./pkg/apis/greenhouse/v1alpha1" ## -app-dir should be Canonical Path Format so absolute path doesn't work. That's why we don't use $(CURDIR) here.
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

.PHONY: e2e
e2e: 
	go test ./test/e2e/... -coverprofile cover.out -v

.PHONY: e2e-local
e2e-local: generate-manifests generate envtest ## Run e2e tests against mock api.
	unset USE_EXISTING_CLUSTER && KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./test/e2e/... -coverprofile cover.out -v

.PHONY: e2e-remote
e2e-remote: ## Run e2e tests against a remote Greenhouse cluster. TEST_KUBECONFIG must be set.
	USE_EXISTING_CLUSTER=true go test ./test/e2e/... -coverprofile cover.out -v

.PHONY: e2e-local-cluster
e2e-local-cluster: e2e-local-cluster-create  ## Run e2e tests on a local KIND cluster.
	USE_EXISTING_CLUSTER=true TEST_KUBECONFIG=$(shell pwd)/test/e2e/local-cluster/e2e.kubeconfig INTERNAL_KUBECONFIG=$(shell pwd)/test/e2e/local-cluster/e2e.internal.kubeconfig go test ./test/e2e/... -coverprofile cover.out -v

.PHONY: e2e-local-cluster-create
e2e-local-cluster-create:
	cd test/e2e/local-cluster && go run . --dockerImagePlatform=$(PLATFORM)


.PHONY: fmt
fmt: goimports golint
	GOBIN=$(LOCALBIN) go fmt ./...
	$(GOIMPORTS) -w -local github.com/cloudoperators/greenhouse .
	$(GOLINT) run -v --timeout 5m

.PHONY: check
check: fmt test

##@ Build

.PHONY: build
build: generate build-greenhouse build-idproxy build-team-membership build-cors-proxy build-greenhousectl build-service-proxy

build-%: GIT_BRANCH  = $(shell git rev-parse --abbrev-ref HEAD)
build-%: GIT_COMMIT  = $(shell git rev-parse --short HEAD)
build-%: GIT_STATE   = $(shell if git diff --quiet; then echo clean; else echo dirty; fi)
build-%: BUILD_DATE  = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
build-%:
	go build -ldflags "-s -w -X github.com/cloudoperators/greenhouse/pkg/version.GitBranch=$(GIT_BRANCH) -X github.com/cloudoperators/greenhouse/pkg/version.GitCommit=$(GIT_COMMIT) -X github.com/cloudoperators/greenhouse/pkg/version.GitState=$(GIT_STATE) -X github.com/cloudoperators/greenhouse/pkg/version.BuildDate=$(BUILD_DATE)" -o bin/$* ./cmd/$*/

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/greenhouse/

.PHONY: docker-build
docker-build:
	docker build --platform ${PLATFORM} -t ${IMG} .

.PHONY: docker-build-dev-env
docker-build-dev-env:
	docker build --platform ${PLATFORM} -t ${IMG_DEV_ENV} -f Dockerfile.dev-env .

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
GOIMPORTS ?= $(LOCALBIN)/goimports
GOLINT ?= $(LOCALBIN)/golangci-lint
ENVTEST ?= $(LOCALBIN)/setup-envtest
HELMIFY ?= $(LOCALBIN)/helmify

## Tool Versions
KUSTOMIZE_VERSION ?= v5.4.2
CONTROLLER_TOOLS_VERSION ?= v0.15.0
GOLINT_VERSION ?= v1.60.1
GINKGOLINTER_VERSION ?= v0.16.2

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN)

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: goimports
goimports: $(GOIMPORTS)
$(GOIMPORTS): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install golang.org/x/tools/cmd/goimports@latest

.PHONY: golint
golint: $(GOLINT)
$(GOLINT): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLINT_VERSION)
	GOBIN=$(LOCALBIN) go install github.com/nunnatsa/ginkgolinter/cmd/ginkgolinter@$(GINKGOLINTER_VERSION)

.PHONY: serve-docs
serve-docs: generate-manifests
ifeq (, $(shell which hugo))
	@echo "Hugo is not installed in your machine. Please install it to serve the documentation locally. Please refer to https://gohugo.io/installation/ for installation instructions."
else
	cd website && hugo server
endif
