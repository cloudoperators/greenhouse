// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package setup

import (
	"errors"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewKubeClientFromConfig - creates a new Kubernetes CRUD client from a given kubeconfig
func NewKubeClientFromConfig(configStr string) (client.Client, error) {
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(configStr))
	if err != nil {
		return nil, err
	}
	return client.New(config, client.Options{Scheme: clientgoscheme.Scheme})
}

// NewKubeClientFromContext - creates a new Kubernetes CRUD client from the kubeconfig current-context
func NewKubeClientFromContext() (client.Client, error) {
	var kubeConfigPath string
	if pathFromEnv := os.Getenv(clientcmd.RecommendedConfigPathEnvVar); pathFromEnv != "" {
		kubeConfigPath = pathFromEnv
	} else {
		kubeConfigPath = path.Join(homedir.HomeDir(),
			clientcmd.RecommendedHomeDir,
			clientcmd.RecommendedFileName)
		if _, err := os.Stat(kubeConfigPath); err != nil {
			if os.IsNotExist(err) {
				return nil, errors.New("kubeconfig was not found in " + kubeConfigPath)
			} else {
				return nil, err
			}
		}
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, err
	}
	return client.New(config, client.Options{Scheme: clientgoscheme.Scheme})
}
