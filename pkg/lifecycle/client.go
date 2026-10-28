// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

// ClientForCluster returns a controller-runtime client for the given remote cluster.
// Callers must provide at least one addToScheme function; an empty scheme will cause resource operations to fail.
func ClientForCluster(ctx context.Context, kubeClient client.Client, cluster *greenhouseapisv1alpha1.Cluster, addToScheme ...func(*runtime.Scheme) error) (client.Client, error) {
	cfg, err := NewRemoteKubeCfg(ctx, kubeClient, cluster)
	if err != nil {
		return nil, err
	}
	return clientFromConfig(cfg, addToScheme...)
}

// RestConfigForCluster returns a rest.Config for the given remote cluster.
func RestConfigForCluster(ctx context.Context, kubeClient client.Client, cluster *greenhouseapisv1alpha1.Cluster) (*rest.Config, error) {
	return NewRemoteKubeCfg(ctx, kubeClient, cluster)
}

func clientFromConfig(cfg *rest.Config, addToScheme ...func(*runtime.Scheme) error) (client.Client, error) {
	sch := runtime.NewScheme()
	for _, fn := range addToScheme {
		if err := fn(sch); err != nil {
			return nil, fmt.Errorf("scheme registration: %w", err)
		}
	}
	return client.New(cfg, client.Options{Scheme: sch})
}
