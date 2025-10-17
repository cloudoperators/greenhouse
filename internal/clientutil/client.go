// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package clientutil

import (
	"context"
	"fmt"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	sourcev2 "github.com/fluxcd/source-watcher/api/v2/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
	dexapi "github.com/cloudoperators/greenhouse/internal/dex/api"
)

var KnownSchemes = []func(*runtime.Scheme) error{
	clientgoscheme.AddToScheme,
	apiextensionsv1.AddToScheme,
	greenhousev1alpha1.AddToScheme,
	greenhousev1alpha2.AddToScheme,
	dexapi.AddToScheme,
	sourcev1.AddToScheme,
	helmv2.AddToScheme,
	kustomizev1.AddToScheme,
	sourcev2.AddToScheme,
}

// NewK8sClient returns a Kubernetes client with registered schemes for the given config or an error.
func NewK8sClient(cfg *rest.Config) (client.Client, error) {
	sch := runtime.NewScheme()
	for _, addScheme := range KnownSchemes {
		if err := addScheme(sch); err != nil {
			return nil, fmt.Errorf("failed to add user scheme: %w", err)
		}
	}
	return client.New(cfg, client.Options{Scheme: sch})
}

// NewK8sClientFromRestClientGetter returns a Kubernetes client with registered schemes for the given RESTClientGetter or an error.
func NewK8sClientFromRestClientGetter(restClientGetter genericclioptions.RESTClientGetter) (client.Client, error) {
	cfg, err := restClientGetter.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	return NewK8sClient(cfg)
}

// NewK8sClientFromCluster returns a client.Client based on the given clusters kubeconfig secret.
func NewK8sClientFromCluster(ctx context.Context, c client.Client, cluster *greenhousev1alpha1.Cluster) (client.Client, error) {
	secret := new(corev1.Secret)
	if err := c.Get(ctx, types.NamespacedName{Name: cluster.GetSecretName(), Namespace: cluster.GetNamespace()}, secret); err != nil {
		return nil, err
	}

	restClientGetter, err := NewRestClientGetterFromSecret(secret, cluster.GetNamespace(), WithPersistentConfig())
	if err != nil {
		return nil, err
	}

	remoteRestClient, err := NewK8sClientFromRestClientGetter(restClientGetter)
	if err != nil {
		return nil, err
	}
	return remoteRestClient, nil
}
