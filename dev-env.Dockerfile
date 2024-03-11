# Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Build the manager binary
FROM golang:1.21 as builder
ENV ENVTEST_K8S_VERSION=1.24.1
WORKDIR /workspace

COPY . .
RUN make generate-manifests

# Build dev-env and setup-envtest
RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	make generate build-dev-env GO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	&& GOBIN=/workspace/bin go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest \
	&& cp $(/workspace/bin/setup-envtest use ${ENVTEST_K8S_VERSION} -p path)/* /usr/local/bin


# final image 
FROM alpine:3.19.1
LABEL source_repository="https://github.com/cloudoperators/greenhouse"
ENV KUBEBUILDER_ASSETS=/usr/local/bin
WORKDIR /
COPY --from=builder /workspace/bin/* .
COPY --from=builder /workspace/charts/manager/crds ./config/crd/bases
COPY --from=builder /workspace/charts/idproxy/crds ./charts/idproxy/crds
COPY --from=builder /workspace/charts/manager/templates/webhooks.yaml ./config/webhook/webhooks.yaml
COPY --from=builder /usr/local/bin ./usr/local/bin
RUN apk add --no-cache libc6-compat

CMD /dev-env && \
	echo "proxying ${DEV_ENV_CONTEXT:-cluster-admin}" && \
	kubectl proxy --kubeconfig=/envtest/internal.kubeconfig --context=${DEV_ENV_CONTEXT:-cluster-admin} --port=8090 --v=9 --address="0.0.0.0"
