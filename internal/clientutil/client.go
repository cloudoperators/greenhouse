// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package clientutil

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

// NewK8sClient returns a Kubernetes client with registered schemes for the given config or an error.
func NewK8sClient(cfg *rest.Config) (client.Client, error) {
	sch := runtime.NewScheme()
	for _, addScheme := range lifecycle.KnownSchemes {
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
