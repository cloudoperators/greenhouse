// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"context"
	"fmt"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	sourcev2 "github.com/fluxcd/source-watcher/api/v2/v1beta1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	greenhouseapisv1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
	dexapi "github.com/cloudoperators/greenhouse/internal/dex/api"
)

var KnownSchemes = []func(*runtime.Scheme) error{
	clientgoscheme.AddToScheme,
	apiextensionsv1.AddToScheme,
	greenhouseapisv1alpha1.AddToScheme,
	greenhouseapisv1alpha2.AddToScheme,
	dexapi.AddToScheme,
	sourcev1.AddToScheme,
	helmv2.AddToScheme,
	kustomizev1.AddToScheme,
	sourcev2.AddToScheme,
}

// NewRemoteClient returns a controller-runtime client for the given remote cluster.
// Callers must provide at least one addToScheme function; an empty scheme will cause resource operations to fail.
func NewRemoteClient(ctx context.Context, kubeClient client.Client, cluster *greenhouseapisv1alpha1.Cluster, addToScheme ...func(*runtime.Scheme) error) (client.Client, error) {
	cfg, err := NewRemoteKubeCfg(ctx, kubeClient, cluster)
	if err != nil {
		return nil, err
	}
	return clientFromConfig(cfg, addToScheme...)
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
