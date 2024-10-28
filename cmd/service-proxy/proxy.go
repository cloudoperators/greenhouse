// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"sync"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/common"
)

func NewProxyManager() *ProxyManager {
	return &ProxyManager{
		clusters: make(map[string]clusterRoutes),
	}
}

type ProxyManager struct {
	client client.Client
	logger logr.Logger

	clusters map[string]clusterRoutes
	mu       sync.RWMutex
}

type clusterRoutes struct {
	transport http.RoundTripper
	routes    map[string]*url.URL
}

// contextClusterKey is used to embed a cluster in the context
type contextClusterKey struct {
}

var apiServerProxyPathRegex = regexp.MustCompile(`/api/v1/namespaces/[^/]+/services/[^/]+/proxy/`)

func (pm *ProxyManager) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	var secret = new(v1.Secret)
	err := pm.client.Get(ctx, req.NamespacedName, secret)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}
	if err != nil || secret.DeletionTimestamp != nil {
		// delete cache
		logger.Info("Removing deleted cluster from cache")
		pm.mu.Lock()
		defer pm.mu.Unlock()
		delete(pm.clusters, req.Name)
		return ctrl.Result{}, nil
	}

	restClientGetter, err := clientutil.NewRestClientGetterFromSecret(secret, "")
	if err != nil {
		return ctrl.Result{}, err
	}
	restConfig, err := restClientGetter.ToRESTConfig()
	if err != nil {
		return ctrl.Result{}, err
	}
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if _, ok := pm.clusters[req.Name]; !ok {
		logger.Info("Adding cluster")
	} else {
		logger.Info("Updating cluster")
	}
	cls := clusterRoutes{}
	if cls.transport, err = rest.TransportFor(restConfig); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create transport for cluster %s: %w", req.Name, err)
	}

	cls.routes = make(map[string]*url.URL)

	k8sAPIURL, err := url.Parse(restConfig.Host)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to parse api url: %w", err)
	}

	plugins, err := pm.pluginsForCluster(ctx, req.Name, req.Namespace)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get plugins for cluster %s: %w", req.Name, err)
	}
	for _, plugin := range plugins {
		for url, svc := range plugin.Status.ExposedServices {
			u := *k8sAPIURL // copy URL struct
			proto := "http"
			if svc.Protocol != nil {
				proto = *svc.Protocol
			}
			u.Path = fmt.Sprintf("/api/v1/namespaces/%s/services/%s:%s:%d/proxy", svc.Namespace, proto, svc.Name, svc.Port)
			cls.routes[url] = &u
		}
	}
	logger.Info("Added routes for cluster", "cluster", req.Name, "routes", cls.routes)
	pm.clusters[req.Name] = cls

	return ctrl.Result{}, nil
}

func (pm *ProxyManager) SetupWithManager(name string, mgr ctrl.Manager) error {
	pm.client = mgr.GetClient()
	pm.logger = mgr.GetLogger()

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v1.Secret{}, builder.WithPredicates(
			clientutil.PredicateFilterBySecretType(greenhouseapis.SecretTypeKubeConfig),
			predicate.ResourceVersionChangedPredicate{},
		)).
		// Watch plugins to be notified about exposed services
		Watches(&greenhousev1alpha1.Plugin{}, handler.EnqueueRequestsFromMapFunc(enqueuePluginForCluster), builder.WithPredicates(predicate.Or(predicate.GenerationChangedPredicate{}, predicate.LabelChangedPredicate{}))).
		Complete(pm)
}

// ReverseProxy returns a reverse proxy that will forward requests to the cluster
func (pm *ProxyManager) ReverseProxy() *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Rewrite:        pm.rewrite,
		ModifyResponse: pm.modifyResponse,
		Transport:      pm,
		FlushInterval:  -1,
		ErrorHandler:   pm.errorHandler,
	}
}

func (pm *ProxyManager) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	cluster, ok := req.Context().Value(contextClusterKey{}).(string)

	if !ok {
		return nil, fmt.Errorf("no upstream found for: %s", req.URL.String())
	}
	cls, ok := pm.clusters[cluster]
	if !ok {
		return nil, fmt.Errorf("cluster %s not found", cluster)
	}
	resp, err = cls.transport.RoundTrip(req)
	log.FromContext(req.Context()).Info("Forwarded request", "status", resp.StatusCode, "upstream", req.URL.String())
	return
}

func (pm *ProxyManager) rewrite(req *httputil.ProxyRequest) {
	req.SetXForwarded()

	l := pm.logger.WithValues("host", req.In.Host, "url", req.In.URL.String(), "method", req.In.Method)

	// inject current logger into context before returning
	defer func() {
		req.Out = req.Out.WithContext(log.IntoContext(req.Out.Context(), l))
	}()

	// hostname is expected to have the format $name--$cluster--$namespace.$organisation.$basedomain
	cluster, err := common.ExtractCluster(req.In.Host)
	if err != nil {
		return
	}
	l = l.WithValues("cluster", cluster)

	pm.mu.RLock()
	defer pm.mu.RUnlock()
	cls, found := pm.clusters[cluster]
	if !found {
		return
	}
	backendURL, found := cls.routes["https://"+req.In.Host]
	if !found {
		return
	}
	// set cluster in context
	req.Out = req.Out.WithContext(context.WithValue(req.Out.Context(), contextClusterKey{}, cluster))

	req.SetURL(backendURL)
}

// modifyResponse strips the k8s API server proxy path prepended to the location header during redirects:
// https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apimachinery/pkg/util/proxy/transport.go#L113
func (pm *ProxyManager) modifyResponse(resp *http.Response) error {
	if location := resp.Header.Get("Location"); location != "" {
		location = apiServerProxyPathRegex.ReplaceAllString(location, "/")
		resp.Header.Set("Location", location)
		log.FromContext(resp.Request.Context()).Info("Rewrote location header", "location", location)
	}
	return nil
}

func (pm *ProxyManager) errorHandler(rw http.ResponseWriter, req *http.Request, err error) {
	logger := pm.logger
	if l, err := logr.FromContext(req.Context()); err == nil {
		logger = l
	}
	logger.Info("Proxy failure", "err", err)
	rw.WriteHeader(http.StatusBadGateway)
}

func (pm *ProxyManager) pluginsForCluster(ctx context.Context, cluster, namespace string) ([]greenhousev1alpha1.Plugin, error) {
	plugins := new(greenhousev1alpha1.PluginList)
	if err := pm.client.List(ctx, plugins, &client.ListOptions{Namespace: namespace}); err != nil {
		return nil, fmt.Errorf("failed to list plugins: %w", err)
	}
	configs := make([]greenhousev1alpha1.Plugin, 0)
	for _, cfg := range plugins.Items {
		// ignore deleted configs
		if cfg.DeletionTimestamp != nil {
			continue
		}
		if cfg.Spec.ClusterName == cluster {
			configs = append(configs, cfg)
		}
	}
	return configs, nil
}

func enqueuePluginForCluster(_ context.Context, o client.Object) []ctrl.Request {
	plugin, ok := o.(*greenhousev1alpha1.Plugin)
	if !ok {
		return nil
	}
	// ignore plugins not tied to a cluster
	if plugin.Spec.ClusterName == "" {
		return nil
	}
	return []ctrl.Request{{NamespacedName: types.NamespacedName{Namespace: plugin.Namespace, Name: plugin.Spec.ClusterName}}}
}
