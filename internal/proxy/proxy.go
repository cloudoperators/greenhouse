// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/go-logr/logr"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlconfig "sigs.k8s.io/controller-runtime/pkg/client/config"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/informers"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

const (
	envDNSDomain   = "GREENHOUSE_DNS_DOMAIN"
	envDebugDomain = "DEBUG_DOMAIN"
)

// PmManager owns the per-cluster transport store and serves proxied requests.
type PmManager struct {
	logger logr.Logger

	// baseCtx is the manager lifetime context (signal-canceled on shutdown). It
	// is the parent for token minting so in-flight mints abort on shutdown.
	baseCtx context.Context

	// cache drives the Cluster and Plugin informers.
	cache cache.Cache
	// reader is an uncached client used to fetch the workload-identity ConfigMap
	// on demand (it has no labels, so it cannot be selected by an informer) and
	// to mint ServiceAccount tokens.
	reader client.Client

	// store holds the per-cluster transports and routes.
	store *Store

	// dnsDomain is used to construct exposed service URLs.
	dnsDomain string
	// debugHost overrides the domain for debugging exposed service URLs locally.
	debugHost string

	clusterInformer *informers.GenericInformer
	pluginInformer  *informers.GenericInformer
}

// NewProxyManager builds a raw controller-runtime cache (no manager) over the
// Cluster and Plugin resources and wires informer event handlers that keep the
// per-cluster transport store in sync.
func NewProxyManager(ctx context.Context, logger logr.Logger) (*PmManager, error) {
	restCfg, err := ctrlconfig.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load rest config: %w", err)
	}

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(greenhousev1alpha1.AddToScheme(scheme))

	namespace, err := clientutil.GetEnv("KUBENAMESPACE")
	if err != nil {
		return nil, err
	}
	opts := cache.Options{
		Scheme:            scheme,
		DefaultNamespaces: map[string]cache.Config{namespace: {}},
	}

	c, err := cache.New(restCfg, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}

	// Uncached reader for the label-less workload-identity ConfigMap.
	reader, err := client.New(restCfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("failed to create reader: %w", err)
	}

	pm := &PmManager{
		logger:    logger,
		baseCtx:   ctx,
		cache:     c,
		reader:    reader,
		store:     NewStore(),
		dnsDomain: os.Getenv(envDNSDomain),
		debugHost: os.Getenv(envDebugDomain),
	}

	if err := pm.registerInformers(ctx); err != nil {
		return nil, err
	}
	return pm, nil
}

// registerInformers wires the Cluster and Plugin event handlers. Add and update
// funnel to a single sync handler; updates carrying a deletion timestamp are
// skipped since the delete handler will fire. ctx is captured by the handlers
// for the client reads they trigger.
func (pm *PmManager) registerInformers(ctx context.Context) error {
	clusterInformer, err := informers.New(ctx, pm.cache, &greenhousev1alpha1.Cluster{}, informers.EventHandlers{
		AddFunc:    func(obj any) { pm.onClusterSync(ctx, obj) },
		UpdateFunc: func(_, newObj any) { pm.onClusterSync(ctx, newObj) },
		DeleteFunc: pm.onClusterDelete,
	})
	if err != nil {
		return err
	}

	pluginInformer, err := informers.New(ctx, pm.cache, &greenhousev1alpha1.Plugin{}, informers.EventHandlers{
		AddFunc:    func(obj any) { pm.onPluginSync(ctx, obj) },
		UpdateFunc: func(_, newObj any) { pm.onPluginSync(ctx, newObj) },
		DeleteFunc: func(obj any) { pm.onPluginDelete(ctx, obj) },
	})
	if err != nil {
		return err
	}

	pm.clusterInformer = clusterInformer
	pm.pluginInformer = pluginInformer
	return nil
}

// isTerminating reports whether the object is being deleted.
func isTerminating(obj lifecycle.RuntimeObject) bool {
	return !obj.GetDeletionTimestamp().IsZero()
}

// Start runs the cache and blocks until ctx is cancelled. It returns once the
// informers have stopped.
func (pm *PmManager) Start(ctx context.Context) error {
	return pm.cache.Start(ctx)
}

// WaitForCacheSync blocks until the informers have populated, so the proxy does
// not start serving before the store is warm.
func (pm *PmManager) WaitForCacheSync(ctx context.Context) error {
	if err := pm.clusterInformer.WaitForSync(ctx); err != nil {
		return err
	}
	return pm.pluginInformer.WaitForSync(ctx)
}

// onClusterSync builds or rebuilds the transport for a cluster on add/update.
// It selects the transport based on the cluster's connectivity: workload-identity
// clusters get a token-refreshing transport built from the Flux ConfigMap, while
// kubeconfig/static-OIDC clusters get a transport built from the Secret.
func (pm *PmManager) onClusterSync(ctx context.Context, obj any) {
	cluster, ok := obj.(*greenhousev1alpha1.Cluster)
	if !ok || isTerminating(cluster) {
		return
	}

	var (
		transport http.RoundTripper
		host      string
		err       error
	)
	if isWorkloadIdentity(cluster) {
		transport, host, err = pm.workloadIdentityTransport(ctx, cluster)
	} else {
		transport, host, err = pm.kubeConfigTransport(ctx, cluster)
	}
	if err != nil {
		pm.logger.Error(err, "failed to build transport for cluster", "cluster", cluster.Name)
		return
	}
	pm.store.SetTransport(cluster.Name, transport, host)
	pm.logger.Info("added cluster to store", "cluster", cluster.Name, "host", host)
	pm.rebuildClusterRoutes(ctx, cluster.Namespace, cluster.Name)
}

// isWorkloadIdentity reports whether the cluster uses Flux object-level workload
// identity, in which case connection config lives in a ConfigMap rather than the
// Secret and tokens are minted per-use.
func isWorkloadIdentity(cluster *greenhousev1alpha1.Cluster) bool {
	return cluster.Annotations[greenhouseapis.ClusterWorkloadIdentityAnnotation] == greenhouseapis.ClusterWorkloadIdentityEnabled
}

// kubeConfigTransport builds a transport for a kubeconfig or static-OIDC cluster
// from the credentials stored in its Secret.
func (pm *PmManager) kubeConfigTransport(ctx context.Context, cluster *greenhousev1alpha1.Cluster) (http.RoundTripper, string, error) {
	restCfg, err := lifecycle.NewRemoteKubeCfg(ctx, pm.reader, cluster)
	if err != nil {
		return nil, "", fmt.Errorf("failed to build rest config: %w", err)
	}
	transport, err := rest.TransportFor(restCfg)
	if err != nil {
		return nil, "", fmt.Errorf("failed to build transport: %w", err)
	}
	return transport, restCfg.Host, nil
}

// onClusterDelete drops a cluster's transport and routes from the store.
func (pm *PmManager) onClusterDelete(obj any) {
	cluster, ok := obj.(*greenhousev1alpha1.Cluster)
	if !ok {
		return
	}
	pm.store.Delete(cluster.Name)
	pm.logger.Info("removed cluster from store", "cluster", cluster.Name)
}

// onPluginSync recomputes the routes for a plugin's cluster on add/update.
func (pm *PmManager) onPluginSync(ctx context.Context, obj any) {
	plugin, ok := obj.(*greenhousev1alpha1.Plugin)
	if !ok || isTerminating(plugin) {
		return
	}
	pm.rebuildClusterRoutes(ctx, plugin.Namespace, plugin.Spec.ClusterName)
}

// onPluginDelete recomputes the routes for the deleted plugin's cluster from the
// plugins that remain.
func (pm *PmManager) onPluginDelete(ctx context.Context, obj any) {
	plugin, ok := obj.(*greenhousev1alpha1.Plugin)
	if !ok {
		return
	}
	pm.rebuildClusterRoutes(ctx, plugin.Namespace, plugin.Spec.ClusterName)
}

// rebuildClusterRoutes lists all plugins bound to the cluster and rebuilds the
// full route set from their exposed services, keyed by the exposed URL.
func (pm *PmManager) rebuildClusterRoutes(ctx context.Context, namespace, clusterName string) {
	if clusterName == "" {
		return
	}
	host, ok := pm.store.Host(clusterName)
	if !ok {
		// Transport for the cluster is not built yet; the cluster sync will run
		// and a later plugin resync rebuilds the routes.
		pm.logger.Info("no transport for cluster yet, skipping route rebuild", "cluster", clusterName)
		return
	}
	apiURL, err := url.Parse(host)
	if err != nil {
		pm.logger.Error(err, "failed to parse api url", "cluster", clusterName, "host", host)
		return
	}

	routes := make(map[string]route)
	pluginList := &greenhousev1alpha1.PluginList{}
	if err := pm.cache.List(ctx, pluginList, client.InNamespace(namespace)); err != nil {
		pm.logger.Error(err, "failed to list plugins", "cluster", clusterName)
		return
	}
	for i := range pluginList.Items {
		plugin := &pluginList.Items[i]
		if plugin.Spec.ClusterName != clusterName || isTerminating(plugin) {
			continue
		}
		for exposedURL, svc := range plugin.Status.ExposedServices {
			if svc.Type != greenhousev1alpha1.ServiceTypeService {
				continue
			}
			if pm.debugHost != "" && pm.dnsDomain != "" {
				exposedURL = strings.ReplaceAll(exposedURL, pm.dnsDomain, pm.debugHost)
			}
			u := *apiURL // copy
			if svc.Protocol != nil && *svc.Protocol == "https" {
				u.Path = fmt.Sprintf("/api/v1/namespaces/%s/services/https:%s:%d/proxy", svc.Namespace, svc.Name, svc.Port)
			} else {
				u.Path = fmt.Sprintf("/api/v1/namespaces/%s/services/%s:%d/proxy", svc.Namespace, svc.Name, svc.Port)
			}
			routes[exposedURL] = route{url: &u, namespace: svc.Namespace, serviceName: svc.Name}
		}
	}
	pm.store.SetRoutes(clusterName, routes)
	pm.logger.Info("rebuilt routes for cluster", "cluster", clusterName, "routes", len(routes))
}

// Ready reports whether both informers have completed their initial sync, so
// the proxy is not marked ready while the store is still cold.
func (pm *PmManager) Ready() bool {
	return pm.clusterInformer.HasSynced() && pm.pluginInformer.HasSynced()
}

// RegisterRoutes mounts the proxy handlers on the given Echo instance.
func (pm *PmManager) RegisterRoutes(e *echo.Echo, registry prometheus.Registerer) {
	e.Any("/*", echo.WrapHandler(pm.InstrumentHandler(registry)))
}
