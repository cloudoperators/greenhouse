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

type PmManager struct {
	logger logr.Logger

	cache  cache.Cache
	reader client.Client

	store *Store

	dnsDomain string
	debugHost string

	clusterInformer *informers.GenericInformer
	pluginInformer  *informers.GenericInformer
}

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

	reader, err := client.New(restCfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("failed to create reader: %w", err)
	}

	pm := &PmManager{
		logger:    logger,
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

func isTerminating(obj lifecycle.RuntimeObject) bool {
	return !obj.GetDeletionTimestamp().IsZero()
}

func (pm *PmManager) Start(ctx context.Context) error {
	return pm.cache.Start(ctx)
}

func (pm *PmManager) WaitForCacheSync(ctx context.Context) error {
	if err := pm.clusterInformer.WaitForSync(ctx); err != nil {
		return err
	}
	return pm.pluginInformer.WaitForSync(ctx)
}

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

func isWorkloadIdentity(cluster *greenhousev1alpha1.Cluster) bool {
	return cluster.Annotations[greenhouseapis.ClusterWorkloadIdentityAnnotation] == greenhouseapis.ClusterWorkloadIdentityEnabled
}

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

func (pm *PmManager) onClusterDelete(obj any) {
	cluster, ok := obj.(*greenhousev1alpha1.Cluster)
	if !ok {
		return
	}
	pm.store.Delete(cluster.Name)
	pm.logger.Info("removed cluster from store", "cluster", cluster.Name)
}

func (pm *PmManager) onPluginSync(ctx context.Context, obj any) {
	plugin, ok := obj.(*greenhousev1alpha1.Plugin)
	if !ok || isTerminating(plugin) {
		return
	}
	pm.rebuildClusterRoutes(ctx, plugin.Namespace, plugin.Spec.ClusterName)
}

func (pm *PmManager) onPluginDelete(ctx context.Context, obj any) {
	plugin, ok := obj.(*greenhousev1alpha1.Plugin)
	if !ok {
		return
	}
	pm.rebuildClusterRoutes(ctx, plugin.Namespace, plugin.Spec.ClusterName)
}

func (pm *PmManager) rebuildClusterRoutes(ctx context.Context, namespace, clusterName string) {
	if clusterName == "" {
		return
	}
	host, ok := pm.store.Host(clusterName)
	if !ok {
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
			u := *apiURL
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

func (pm *PmManager) Ready() bool {
	return pm.clusterInformer.HasSynced() && pm.pluginInformer.HasSynced()
}

func (pm *PmManager) RegisterRoutes(e *echo.Echo, registry prometheus.Registerer) {
	e.Any("/*", echo.WrapHandler(pm.InstrumentHandler(registry)))
}
