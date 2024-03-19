// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package clientutil

import (
	"fmt"
	"sync"

	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Implements the genericclioptions.RESTClientGetter interface and additionally allows to access the KubeConfig from Bytes/Secret.
type RestClientGetter struct {
	namespace          string
	clientConfigGetter clientcmd.ClientConfigLoader
	restConfig         *rest.Config

	// If set to true, will use persistent client config, rest mapper, discovery client, and
	// propagate them to the places that need them, rather than
	// instantiating them multiple times.
	usePersistentConfig bool
	// Allows increasing burst used for discovery, this is useful
	// in clusters with many registered resources
	discoveryBurst int
	// Allows increasing qps used for discovery, this is useful
	// in clusters with many registered resources
	discoveryQPS float32

	runtimeOpts *RuntimeOptions

	// overrides is used to override entries in the kubeconfig
	overrides *clientcmd.ConfigOverrides

	clientConfig     clientcmd.ClientConfig
	clientConfigLock sync.Mutex

	restMapper     meta.RESTMapper
	restMapperLock sync.Mutex

	discoveryClient     discovery.CachedDiscoveryInterface
	discoveryClientLock sync.Mutex
}

// RuntimeOptions contains the runtime options for the Kubernetes client.
type RuntimeOptions struct {
	// QPS indicates the maximum QPS to the master from this client. Default is 50.
	QPS float32
	// Maximum burst for throttle. Default is 300.
	Burst int
}

// BindFlags will parse the given flagset for runtime option flags and set the runtime options accordingly. Defaults QPS to 50 and Burst to 300 if the flag was not set.
func (o RuntimeOptions) BindFlags(fs *pflag.FlagSet) {
	fs.Float32Var(&o.QPS, "kube-api-qps", 50, "Maximum QPS to the master from this client.")
	fs.IntVar(&o.Burst, "kube-api-burst", 300, "Maximum burst for throttle.")
}

type KubeClientOption func(*RestClientGetter)

// WithDiscoveryOpts allows overwriting the discovery QPS and Burst
func WithDiscoveryOpts(qps float32, burst int) KubeClientOption {
	return func(cg *RestClientGetter) {
		cg.discoveryQPS = qps
		cg.discoveryBurst = burst
	}
}

// WithRuntimeOptions allows overwriting client QPS and Burst
func WithRuntimeOptions(o RuntimeOptions) KubeClientOption {
	return func(cg *RestClientGetter) {
		cg.runtimeOpts = &o
	}
}

func WithOverrides(overrides clientcmd.ConfigOverrides) KubeClientOption {
	return func(cg *RestClientGetter) {
		cg.overrides = &overrides
	}
}

func WithPersistentConfig() KubeClientOption {
	return func(cg *RestClientGetter) {
		cg.usePersistentConfig = true
	}
}

// NewRestClientGetterFromRestConfig returns a RestClientGetter from a in-memory kube config.
func NewRestClientGetterFromRestConfig(cfg *rest.Config, namespace string, opts ...KubeClientOption) *RestClientGetter {
	if namespace == "" {
		namespace = corev1.NamespaceDefault
	}

	// The more groups you have, the more discovery requests you need to make.
	// with a discoveryBurst of 300, we will not be rate-limiting for most clusters but
	// the safeguard will still be here. This config is only used for discovery.
	discoveryBurst := 300

	g := &RestClientGetter{
		restConfig:     cfg,
		namespace:      namespace,
		discoveryBurst: discoveryBurst,
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

func NewRestClientGetterForInCluster(namespace string, opts ...KubeClientOption) (*RestClientGetter, error) {
	cfg, err := ctrl.GetConfig()
	if err != nil {
		return nil, err
	}
	// The more groups you have, the more discovery requests you need to make.
	// with a discoveryBurst of 300, we will not be rate-limiting for most clusters but
	// the safeguard will still be here. This config is only used for discovery.
	discoveryBurst := 300
	g := &RestClientGetter{
		restConfig:     cfg,
		namespace:      namespace,
		discoveryBurst: discoveryBurst,
	}
	for _, opt := range opts {
		opt(g)
	}
	return g, nil
}

// NewRestClientGetterFromSecret returns a RestClientGetter from a secret containing a Kube Config.
func NewRestClientGetterFromSecret(secret *corev1.Secret, namespace string, opts ...KubeClientOption) (*RestClientGetter, error) {
	if err := ValidateSecretForKubeConfig(secret); err != nil {
		return nil, err
	}
	if namespace == "" {
		namespace = corev1.NamespaceDefault
	}

	// The more groups you have, the more discovery requests you need to make.
	// with a discoveryBurst of 300, we will not be rate-limiting for most clusters but
	// the safeguard will still be here. This config is only used for discovery.
	discoveryBurst := 300

	g := &RestClientGetter{
		clientConfigGetter: NewClientConfigLoaderFromSecret(secret),
		namespace:          namespace,
		discoveryBurst:     discoveryBurst,
	}

	for _, opt := range opts {
		opt(g)
	}
	return g, nil
}

// NewRestClientGetterFromBytes returns a RestClientGetter from a []bytes containing a Kube Config.
func NewRestClientGetterFromBytes(config []byte, namespace string, opts ...KubeClientOption) *RestClientGetter {
	if namespace == "" {
		namespace = corev1.NamespaceDefault
	}

	// The more groups you have, the more discovery requests you need to make.
	// with a discoveryBurst of 300, we will not be rate-limiting for most clusters but
	// the safeguard will still be here. This config is only used for discovery.
	discoveryBurst := 300

	g := &RestClientGetter{
		clientConfigGetter: NewClientConfigLoaderFromBytes(config),
		namespace:          namespace,
		discoveryBurst:     discoveryBurst,
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

func (cg *RestClientGetter) ToRESTConfig() (*rest.Config, error) {
	if cg.restConfig != nil {
		if cg.runtimeOpts != nil {
			cg.restConfig.QPS = cg.runtimeOpts.QPS
			cg.restConfig.Burst = cg.runtimeOpts.Burst
		}
		return cg.restConfig, nil
	}

	cfg, err := cg.ToRawKubeConfigLoader().ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("RestClientGetter has no valid KubeConfig: %w", err)
	}
	if cg.runtimeOpts != nil {
		cfg.QPS = cg.runtimeOpts.QPS
		cfg.Burst = cg.runtimeOpts.Burst
	}
	return cfg, nil
}

func (cg *RestClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	if cg.usePersistentConfig {
		return cg.toPersistentDiscoveryClient()
	}
	return cg.toDiscoveryClient()
}

// toPersistentDiscoveryClient returns a memory cached discovery client.
func (cg *RestClientGetter) toPersistentDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	cg.discoveryClientLock.Lock()
	defer cg.discoveryClientLock.Unlock()

	if cg.discoveryClient == nil {
		dc, err := cg.toDiscoveryClient()
		if err != nil {
			return nil, err
		}
		cg.discoveryClient = dc
	}

	return cg.discoveryClient, nil
}

// toDiscoveryClient returns a memory cached discovery client.
func (cg *RestClientGetter) toDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	cfg, err := cg.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	cfg.Burst = cg.discoveryBurst
	cfg.QPS = cg.discoveryQPS

	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return memory.NewMemCacheClient(dc), nil
}

func (cg *RestClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	if cg.usePersistentConfig {
		return cg.toPersistentRESTMapper()
	}
	return cg.toRESTMapper()
}

func (cg *RestClientGetter) toPersistentRESTMapper() (meta.RESTMapper, error) {
	cg.restMapperLock.Lock()
	defer cg.restMapperLock.Unlock()

	if cg.restMapper == nil {
		mapper, err := cg.toRESTMapper()
		if err != nil {
			return nil, err
		}
		cg.restMapper = mapper
	}

	return cg.restMapper, nil
}

func (cg *RestClientGetter) toRESTMapper() (meta.RESTMapper, error) {
	dc, err := cg.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(dc)
	expander := restmapper.NewShortcutExpander(mapper, dc, nil)
	return expander, nil
}

func (cg *RestClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	if cg.usePersistentConfig {
		return cg.toPersistentRawKubeConfigLoader()
	}
	return cg.toRawKubeConfigLoader()
}

func (cg *RestClientGetter) toPersistentRawKubeConfigLoader() clientcmd.ClientConfig {
	cg.clientConfigLock.Lock()
	defer cg.clientConfigLock.Unlock()

	if cg.clientConfig == nil {
		cc := cg.toRawKubeConfigLoader()
		cg.clientConfig = cc
	}

	return cg.clientConfig
}

func (cg *RestClientGetter) toRawKubeConfigLoader() clientcmd.ClientConfig {
	// clientConfigGetter is not set when operating directly on the central
	if cg.clientConfigGetter == nil {
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		// use the standard defaults for this client command
		// DEPRECATED: remove and replace with something more accurate
		loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig

		overrides := &clientcmd.ConfigOverrides{ClusterDefaults: clientcmd.ClusterDefaults}

		overrides.Context.Namespace = cg.namespace

		return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)
	}

	var overrides *clientcmd.ConfigOverrides
	switch {
	case cg.overrides != nil:
		overrides = cg.overrides
	default:
		overrides = &clientcmd.ConfigOverrides{}
		overrides.Context.Namespace = cg.namespace
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(cg.clientConfigGetter, overrides)
}
