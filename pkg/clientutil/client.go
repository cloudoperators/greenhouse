// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package clientutil

import (
	"net/http"
	"net/url"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	extensionsgreenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/extensions.greenhouse/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	dexapi "github.com/cloudoperators/greenhouse/pkg/dex/api"
)

var Scheme = runtime.NewScheme()

func init() {
	for _, addToSchemeFunc := range []func(*runtime.Scheme) error{
		clientgoscheme.AddToScheme,
		apiextensionsv1.AddToScheme,
		greenhousev1alpha1.AddToScheme,
		extensionsgreenhousev1alpha1.AddToScheme,
		dexapi.AddToScheme,
	} {
		utilruntime.Must(addToSchemeFunc(Scheme))
	}
}

// NewK8sClient returns a Kubernetes client with registered schemes for the given config or an error.
func NewK8sClient(cfg *rest.Config) (client.Client, error) {
	return client.New(cfg, client.Options{Scheme: Scheme})
}

// NewK8sClientFromRestClientGetter returns a Kubernetes client with registered schemes for the given RESTClientGetter or an error.
func NewK8sClientFromRestClientGetter(restClientGetter genericclioptions.RESTClientGetter) (client.Client, error) {
	cfg, err := restClientGetter.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	return NewK8sClient(cfg)
}

func NewHeadscaleK8sClientFromRestClientGetter(restClientGetter genericclioptions.RESTClientGetter, proxy, headscaleAddress string) (client.Client, error) {
	cfg, err := restClientGetter.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	cfg.Host = "https://" + headscaleAddress
	cfg.TLSClientConfig.ServerName = "127.0.0.1"
	cfg.Proxy = func(req *http.Request) (*url.URL, error) {
		return url.Parse(proxy)
	}
	return NewK8sClient(cfg)
}
