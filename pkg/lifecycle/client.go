// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"context"
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

var knownSchemes = []func(*runtime.Scheme) error{
	clientgoscheme.AddToScheme,
	apiextensionsv1.AddToScheme,
}

// NewRemoteClient returns a controller-runtime client for the given remote cluster.
func NewRemoteClient(ctx context.Context, kubeClient client.Client, cluster *greenhouseapisv1alpha1.Cluster) (client.Client, error) {
	cfg, err := NewRemoteKubeCfg(ctx, kubeClient, cluster)
	if err != nil {
		return nil, err
	}
	sch := runtime.NewScheme()
	for _, addScheme := range knownSchemes {
		if err := addScheme(sch); err != nil {
			return nil, fmt.Errorf("scheme registration: %w", err)
		}
	}
	return client.New(cfg, client.Options{Scheme: sch})
}
