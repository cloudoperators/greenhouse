#!/usr/bin/env bash

# SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT="${REPO_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
REGISTRY_CONTAINER_NAME="registry"
REGISTRY_PORT=5000

# Detect host architecture.
ARCH="$(uname -m)"
case "${ARCH}" in
  x86_64)        REGISTRY_IMAGE="ghcr.io/project-zot/zot-linux-amd64:latest" ;;
  arm64|aarch64) REGISTRY_IMAGE="ghcr.io/project-zot/zot-linux-arm64:latest" ;;
  *) echo "Unsupported architecture: ${ARCH}"; exit 1 ;;
esac
REGISTRY_CONFIG="${REPO_ROOT}/registry/config.json"

# 1. Start registry container if not already running.
if ! docker inspect "${REGISTRY_CONTAINER_NAME}" &>/dev/null; then
  echo "Starting registry container..."
  docker run -d \
    --name "${REGISTRY_CONTAINER_NAME}" \
    --restart=unless-stopped \
    -p "${REGISTRY_PORT}:${REGISTRY_PORT}" \
    -v "${REGISTRY_CONFIG}:/etc/zot/config.json" \
    "${REGISTRY_IMAGE}"
else
  echo "Registry container already running."
fi

# 2. Connect registry to the kind network.
if docker network inspect "kind" &>/dev/null; then
  if docker network inspect "kind" --format '{{range .Containers}}{{.Name}} {{end}}' | grep -qw "${REGISTRY_CONTAINER_NAME}"; then
    echo "Registry already connected to network kind."
  else
    echo "Connecting registry to network kind..."
    docker network connect "kind" "${REGISTRY_CONTAINER_NAME}"
  fi
else
  echo "Network kind not found, skipping."
fi

# 3. Configure containerd on each node to use the registry.
for cluster in greenhouse-admin greenhouse-remote greenhouse-authz; do
  if ! kind get clusters | grep -qw "${cluster}"; then
    echo "Cluster ${cluster} not found, skipping node configuration."
    continue
  fi

  for node in $(kind get nodes --name "${cluster}"); do
    echo "Configuring containerd registry on node ${node}..."

    # Ensure certs.d config_path is set in containerd config (idempotent).
    if ! docker exec "${node}" grep -q "config_path.*certs.d" /etc/containerd/config.toml; then
      docker exec "${node}" bash -c 'cat >> /etc/containerd/config.toml <<EOF

[plugins."io.containerd.grpc.v1.cri".registry]
  config_path = "/etc/containerd/certs.d"
EOF'
      docker exec "${node}" systemctl restart containerd
    fi

    # Write hosts.toml for plain HTTP registry.
    docker exec "${node}" mkdir -p "/etc/containerd/certs.d/registry:${REGISTRY_PORT}"
    cat <<EOF | docker exec -i "${node}" tee "/etc/containerd/certs.d/registry:${REGISTRY_PORT}/hosts.toml" > /dev/null
server = "http://registry:${REGISTRY_PORT}"

[host."http://registry:${REGISTRY_PORT}"]
  capabilities = ["pull", "resolve"]
EOF
  done

  # 4. Apply ConfigMap documenting the local registry (kind docs).
  kubectl --context "kind-${cluster}" apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "registry:${REGISTRY_PORT}"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF
done
