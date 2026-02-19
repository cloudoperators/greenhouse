# Authorization Webhook Certificate Management
# This file contains all authz-related make targets for generating and managing
# certificates required by the authorization webhook.

AUTHZ_CLUSTER ?= greenhouse-authz

.PHONY: setup-e2e-authz
setup-e2e-authz: cli authz-certs
	CONTROLLER_ENABLED=$(WITH_CONTROLLER) $(CLI) dev setup -f e2e/authz.config.yaml
	make prepare-e2e-authz

.PHONY: clean-e2e-authz
clean-e2e-authz:
	kind delete cluster --name $(REMOTE_CLUSTER)
	kind delete cluster --name $(AUTHZ_CLUSTER)
	rm -rf $(LOCALBIN)/*.kubeconfig

.PHONY: e2e-local-authz
e2e-local-authz: SCENARIO := authz
e2e-local-authz: prepare-e2e-authz
	GREENHOUSE_ADMIN_KUBECONFIG="$(E2E_RESULT_DIR)/$(AUTHZ_CLUSTER).kubeconfig" \
    	GREENHOUSE_REMOTE_KUBECONFIG="$(E2E_RESULT_DIR)/$(REMOTE_CLUSTER).kubeconfig" \
    	GREENHOUSE_REMOTE_INT_KUBECONFIG="$(E2E_RESULT_DIR)/$(REMOTE_CLUSTER)-int.kubeconfig" \
    	CONTROLLER_LOGS_PATH="$(E2E_RESULT_DIR)/$(SCENARIO)-e2e-pod-logs.txt" \
    	EXECUTION_ENV=$(EXECUTION_ENV) \
		GOMEGA_DEFAULT_EVENTUALLY_TIMEOUT="2m" \
		go test -tags="$(SCENARIO)E2E" $(shell pwd)/e2e/$(SCENARIO) -test.v -ginkgo.v --ginkgo.json-report=$(E2E_REPORT_PATH)

.PHONY: prepare-e2e-authz
prepare-e2e-authz:
	kind get kubeconfig --name $(AUTHZ_CLUSTER) > $(shell pwd)/bin/$(AUTHZ_CLUSTER).kubeconfig
	kind get kubeconfig --name $(REMOTE_CLUSTER) > $(shell pwd)/bin/$(REMOTE_CLUSTER).kubeconfig
	kind get kubeconfig --name $(REMOTE_CLUSTER) --internal > ${PWD}/bin/$(REMOTE_CLUSTER)-int.kubeconfig

# Certs for Authorization Webhook
AUTHZ_CERTS_DIR := bin/authz-certs
AUTHZ_CA_KEY := $(AUTHZ_CERTS_DIR)/ca.key
AUTHZ_CA_CRT := $(AUTHZ_CERTS_DIR)/ca.crt
AUTHZ_SERVER_KEY := $(AUTHZ_CERTS_DIR)/tls.key
AUTHZ_SERVER_CSR := $(AUTHZ_CERTS_DIR)/tls.csr
AUTHZ_SERVER_CRT := $(AUTHZ_CERTS_DIR)/tls.crt
AUTHZ_CLIENT_KEY := $(AUTHZ_CERTS_DIR)/apiserver.key
AUTHZ_CLIENT_CSR := $(AUTHZ_CERTS_DIR)/apiserver.csr
AUTHZ_CLIENT_CRT := $(AUTHZ_CERTS_DIR)/apiserver.crt

AUTHZ_SAN_DNS := DNS:greenhouse-authz,DNS:greenhouse-authz.greenhouse.svc,DNS:greenhouse-authz.greenhouse.svc.cluster.local,IP:127.0.0.1

.PHONY: authz-certs
authz-certs: clean-authz-certs authz-ca authz-server authz-client ## Generate all authorization webhook certificates

$(AUTHZ_CERTS_DIR):
	mkdir -p $(AUTHZ_CERTS_DIR)

.PHONY: authz-ca
authz-ca: $(AUTHZ_CA_CRT) ## Generate CA certificate for authz webhook
$(AUTHZ_CA_CRT): | $(AUTHZ_CERTS_DIR)
	openssl genrsa -out $(AUTHZ_CA_KEY) 4096
	openssl req -x509 -new -nodes -key $(AUTHZ_CA_KEY) -sha256 -days 1095 \
		-subj "/CN=greenhouse-authz-ca" \
		-out $(AUTHZ_CA_CRT)

.PHONY: authz-server
authz-server: $(AUTHZ_SERVER_CRT) ## Generate server certificate for authz webhook

$(AUTHZ_SERVER_CRT): $(AUTHZ_SERVER_CSR) $(AUTHZ_CA_CRT)
	openssl x509 -req -in $(AUTHZ_SERVER_CSR) -CA $(AUTHZ_CA_CRT) -CAkey $(AUTHZ_CA_KEY) -CAcreateserial \
		-out $(AUTHZ_SERVER_CRT) -days 365 -sha256 \
		-extfile <(printf "subjectAltName=$(AUTHZ_SAN_DNS)\nextendedKeyUsage=serverAuth\nkeyUsage=digitalSignature,keyEncipherment\nbasicConstraints=CA:FALSE")

$(AUTHZ_SERVER_CSR): $(AUTHZ_SERVER_KEY)
	openssl req -new -key $(AUTHZ_SERVER_KEY) \
		-subj "/CN=greenhouse-authz.greenhouse.svc" \
		-addext "subjectAltName=$(AUTHZ_SAN_DNS)" \
		-addext "extendedKeyUsage=serverAuth" \
		-addext "keyUsage=digitalSignature,keyEncipherment" \
		-out $(AUTHZ_SERVER_CSR)

$(AUTHZ_SERVER_KEY): | $(AUTHZ_CERTS_DIR)
	openssl genrsa -out $(AUTHZ_SERVER_KEY) 2048

.PHONY: authz-client
authz-client: $(AUTHZ_CLIENT_CRT) ## Generate client certificate for authz webhook

$(AUTHZ_CLIENT_CRT): $(AUTHZ_CLIENT_CSR) $(AUTHZ_CA_CRT)
	openssl x509 -req -in $(AUTHZ_CLIENT_CSR) -CA $(AUTHZ_CA_CRT) -CAkey $(AUTHZ_CA_KEY) -CAcreateserial \
		-out $(AUTHZ_CLIENT_CRT) -days 365 -sha256 \
		-extfile <(printf "extendedKeyUsage=clientAuth\nkeyUsage=digitalSignature,keyEncipherment\nbasicConstraints=CA:FALSE")

$(AUTHZ_CLIENT_CSR): $(AUTHZ_CLIENT_KEY)
	openssl req -new -key $(AUTHZ_CLIENT_KEY) \
		-subj "/CN=kube-apiserver-authz-client" \
		-addext "extendedKeyUsage=clientAuth" \
		-addext "keyUsage=digitalSignature,keyEncipherment" \
		-out $(AUTHZ_CLIENT_CSR)

$(AUTHZ_CLIENT_KEY): | $(AUTHZ_CERTS_DIR)
	openssl genrsa -out $(AUTHZ_CLIENT_KEY) 2048

.PHONY: clean-authz-certs
clean-authz-certs: ## Clean up all authorization webhook certificates
	rm -rf $(AUTHZ_CERTS_DIR)