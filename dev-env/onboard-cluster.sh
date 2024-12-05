#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
# SPDX-License-Identifier: Apache-2.0


set -o errexit
set -o pipefail

KIND_CLUSTER_VERSION="kindest/node:v1.29.2"
KUBECONFIG_DIR="./envtest"


function prepare_cluster(){
  rm -f ${REMOTE_CLUSTER_KUBECONFIG}
	kind delete cluster --name "${REMOTE_CLUSTER_NAME}"
	kind create cluster --name "${REMOTE_CLUSTER_NAME}" --kubeconfig="${REMOTE_CLUSTER_KUBECONFIG}" --image ${KIND_CLUSTER_VERSION}
 
  KIND_SERVER="https://${REMOTE_CLUSTER_NAME}-control-plane:6443"
	kubectl --kubeconfig="${REMOTE_CLUSTER_KUBECONFIG}" config set-cluster "kind-${REMOTE_CLUSTER_NAME}" --server="${KIND_SERVER}"

  echo "connecting ${REMOTE_CLUSTER_NAME} to dev-env_default network"
  docker network connect "dev-env_default" "${REMOTE_CLUSTER_NAME}-control-plane"
}

function onboard_cluster(){
  echo "Creating secret on dev-env cluster to onboard ${REMOTE_CLUSTER_NAME}"
  kubectl --kubeconfig="${KUBECONFIG_DIR}/kubeconfig" --namespace=test-org delete secret ${REMOTE_CLUSTER_NAME} --ignore-not-found=true
  kubectl --kubeconfig="${KUBECONFIG_DIR}/kubeconfig" --namespace=test-org create secret generic ${REMOTE_CLUSTER_NAME} --type=greenhouse.sap/kubeconfig --from-file=kubeconfig="${REMOTE_CLUSTER_KUBECONFIG}"
}

REMOTE_CLUSTER_NAME=$1
if [[ -z "${REMOTE_CLUSTER_NAME}" ]]; then
  REMOTE_CLUSTER_NAME="remote-cluster"
fi
REMOTE_CLUSTER_KUBECONFIG="${KUBECONFIG_DIR}/${REMOTE_CLUSTER_NAME}.kubeconfig"

prepare_cluster
onboard_cluster

echo "Exporting kind kubeconfig for ${KUBECONFIG_DIR}/${REMOTE_CLUSTER_NAME}-kubeconfig."
kind export kubeconfig --name remote-cluster --kubeconfig ${KUBECONFIG_DIR}/${REMOTE_CLUSTER_NAME}-kubeconfig