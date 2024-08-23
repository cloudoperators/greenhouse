// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package setup

import (
	"errors"
	"fmt"
	"github.com/cloudoperators/greenhouse/pkg/internal/local/utils"
	"strings"
)

// createKindCluster - creates a kind cluster with the given name
// if the cluster already exists, it sets the context to the existing cluster
func createKindCluster(clusterName string) error {
	exists, err := kindClusterExists(clusterName)
	if err != nil {
		return err
	} else if exists {
		utils.Logf("kind cluster with name %s already exists", clusterName)
		return utils.Shell{
			Cmd: "kubectl config set-context kind-${name}",
			Vars: map[string]string{
				"name": clusterName,
			},
		}.Exec()
	}
	return utils.Shell{
		Cmd: "kind create cluster --name ${name}",
		Vars: map[string]string{
			"name": clusterName,
		},
	}.Exec()
}

// deleteKindCluster - deletes a kind cluster with the given name
// if the cluster does not exist, it does nothing
func deleteKindCluster(clusterName string) error {
	exists, err := kindClusterExists(clusterName)
	if err != nil {
		return err
	} else if !exists {
		utils.Logf("kind cluster with name %s does not exist", clusterName)
		return nil
	}
	return utils.Shell{
		Cmd: "kind delete cluster --name ${name}",
		Vars: map[string]string{
			"name": clusterName,
		},
	}.Exec()
}

// kindClusterExists - checks if a kind cluster with the given name exists
func kindClusterExists(clusterName string) (bool, error) {
	clusters, err := getKindClusters()
	if err != nil {
		return false, fmt.Errorf("failed to check if cluster exists: %w", err)
	}
	utils.Logf("checking if cluster %s exists...", clusterName)
	for _, c := range clusters {
		if c == clusterName {
			return true, nil
		}
	}
	return false, nil
}

// getKindClusters - returns a list of all kind clusters
func getKindClusters() ([]string, error) {
	result, err := utils.Shell{
		Cmd: "kind get clusters",
	}.ExecWithResult()
	if err != nil {
		return nil, err
	}
	return strings.FieldsFunc(result, func(r rune) bool {
		return r == '\n'
	}), nil
}

// createNamespace - creates a namespace with the given name
func createNamespace(namespaceName string) error {
	if strings.TrimSpace(namespaceName) == "" {
		return errors.New("namespace name cannot be empty")
	}
	return utils.ShellPipe{
		Shells: []utils.Shell{
			{
				Cmd: "kubectl create namespace ${namespace} --dry-run=client -o yaml",
				Vars: map[string]string{
					"namespace": namespaceName,
				},
			},
			{
				Cmd: "kubectl apply -f -",
			},
		},
	}.Exec()
}

// getKubeCfg - get kind cluster kubeconfig
// if internal is true, it returns the internal kubeconfig of the cluster
func getKubeCfg(clusterName string, internal bool) (string, error) {
	sh := utils.Shell{}
	sh.Cmd = fmt.Sprintf("kind get kubeconfig --name %s", clusterName)
	if internal {
		sh.Cmd += " --internal"
	}
	return sh.ExecWithResult()
}

// loadImage - loads a docker image into a kind cluster
func loadImage(image string, clusterName string) error {
	sh := utils.Shell{
		Cmd: "kind load docker-image ${image} --name ${cluster}",
		Vars: map[string]string{
			"image":   image,
			"cluster": clusterName,
		},
	}
	return sh.Exec()
}
