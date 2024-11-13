// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package klient

import (
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewKubeClientFromConfig - creates a new Kubernetes CRUD client from a given kubeconfig
func NewKubeClientFromConfig(configStr string) (client.Client, error) {
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(configStr))
	if err != nil {
		return nil, err
	}
	return client.New(config, client.Options{Scheme: scheme.Scheme})
}
