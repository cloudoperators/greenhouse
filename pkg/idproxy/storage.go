// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package idproxy

import (
	"fmt"
	"os"

	"github.com/dexidp/dex/pkg/log"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/kubernetes"
	"github.com/dexidp/dex/storage/kubernetes/k8sapi"
	"github.com/ghodss/yaml"
)

func NewKubernetesStorage(kubeconfig, kubecontext, namespace string, logger log.Logger) (storage.Storage, error) {
	var storageConfig = kubernetes.Config{InCluster: true}
	if kubeconfig != "" {
		k, err := generateKubeConfig(kubeconfig, kubecontext, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to generate kubeconfig: %w", err)
		}
		file, err := os.CreateTemp("", "idproxy")
		if err != nil {
			return nil, fmt.Errorf("failed ot create temp file: %w", err)
		}
		_, err = file.Write(k)
		file.Close()
		defer os.Remove(file.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to write temporary kubeconfig %s: %w", file.Name(), err)
		}
		storageConfig.InCluster = false
		storageConfig.KubeConfigFile = file.Name()
	}

	return storageConfig.Open(logger)
}

func generateKubeConfig(kubeconfig, kubecontext, namespace string) ([]byte, error) {
	data, err := os.ReadFile(kubeconfig)

	if err != nil {
		return nil, err
	}

	var config k8sapi.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", kubeconfig, err)
	}
	context, err := getContext(&config, kubecontext)
	if err != nil {
		return nil, err
	}
	if namespace != "" {
		context.Context.Namespace = namespace
	}
	authinfo, err := getAuthInfo(&config, context.Context.AuthInfo)
	if err != nil {
		return nil, err
	}
	cluster, err := getCluster(&config, context.Context.Cluster)
	if err != nil {
		return nil, err
	}

	newConfig := k8sapi.Config{
		CurrentContext: context.Name,
		Contexts:       []k8sapi.NamedContext{context},
		Clusters:       []k8sapi.NamedCluster{cluster},
		AuthInfos:      []k8sapi.NamedAuthInfo{authinfo},
	}

	return yaml.Marshal(newConfig)
}

func getContext(c *k8sapi.Config, context string) (k8sapi.NamedContext, error) {
	if context == "" {
		context = c.CurrentContext
	}
	for _, c := range c.Contexts {
		if c.Name == context {
			return c, nil
		}
	}
	return k8sapi.NamedContext{}, fmt.Errorf("context %#v not found", context)
}

func getAuthInfo(c *k8sapi.Config, name string) (k8sapi.NamedAuthInfo, error) {
	for _, a := range c.AuthInfos {
		if a.Name == name {
			if a.AuthInfo.AuthProvider != nil {
				if a.AuthInfo.AuthProvider.Name == "oidc" {
					a.AuthInfo.Token = a.AuthInfo.AuthProvider.Config["id-token"]
				}
				a.AuthInfo.AuthProvider = nil
			}
			return a, nil
		}
	}
	return k8sapi.NamedAuthInfo{}, fmt.Errorf("authinfo %#v not found", name)
}

func getCluster(c *k8sapi.Config, name string) (k8sapi.NamedCluster, error) {
	for _, c := range c.Clusters {
		if c.Name == name {
			return c, nil
		}
	}
	return k8sapi.NamedCluster{}, fmt.Errorf("cluster %#v not found", name)
}
