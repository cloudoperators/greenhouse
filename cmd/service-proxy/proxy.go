// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// The ProxyManager struct is used to manage the reverse proxy and the cluster routes.
// The Reconcile method is used to add or update the cluster routes.
// When reconciling cluster routes from a Plugin with an exposed service, the ProxyManager persists the transport and URL necessary to proxy an incoming request in the clusters map.
// The transport is created using the credentials from the Secret associated with the cluster.
// The URL is created using the k8s API server proxy: https://kubernetes.io/docs/tasks/access-application-cluster/access-cluster-services/#discovering-builtin-services
// Entries are saved by cluster and exposed URL. E.g., if a Plugin exposes a service with the URL "https://cluster1--1234567.example.com" on cluster-1, the route is saved in
// clusters["cluster-1"]clusterRoutes{
//   transport: net/http.RoundTripper{$TransportCreatedFromClusterKubeConfig},
//   routes: map[string]route ["https://cluster-1--1234567.organisation.basedomain"]route{
//     url: *net/url.URL {$BackenURL},
//     serviceName: $serviceName,
//     namespace: $serviceNamespace
//   }
// }.

package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
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

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/common"
)

func NewProxyManager() *ProxyManager {
	return &ProxyManager{
		clusters: make(map[string]clusterRoutes),
	}
}

type ProxyManager struct {
	client   client.Client
	logger   logr.Logger
	clusters map[string]clusterRoutes
	mu       sync.RWMutex
}

type clusterRoutes struct {
	transport http.RoundTripper
	routes    map[string]route
}

// route holds the url the request should be forwarded to and the service name and namespace as metadata
type route struct {
	url         *url.URL
	serviceName string
	namespace   string
}

// contextClusterKey is used to embed a cluster in the context
type contextClusterKey struct {
}

// contextNamespaceKey is used to embed a namespace in the context
type contextNamespaceKey struct {
}

// contextNameKey is used to embed a name in the context
type contextNameKey struct {
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

	cls.routes = make(map[string]route)

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
			if svc.Type != greenhousev1alpha1.ServiceTypeService {
				continue
			}

			u := *k8sAPIURL // copy URL struct

			if svc.Protocol != nil && *svc.Protocol == "https" {
				// For HTTPS, format should be: https:<service_name>:<port>
				u.Path = fmt.Sprintf("/api/v1/namespaces/%s/services/https:%s:%d/proxy", svc.Namespace, svc.Name, svc.Port)
			} else {
				// For HTTP, format should be: <service_name>:<port>
				u.Path = fmt.Sprintf("/api/v1/namespaces/%s/services/%s:%d/proxy", svc.Namespace, svc.Name, svc.Port)
			}
			cls.routes[url] = route{url: &u, namespace: svc.Namespace, serviceName: svc.Name}
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
			clientutil.PredicateFilterBySecretTypes(greenhouseapis.SecretTypeKubeConfig, greenhouseapis.SecretTypeOIDCConfig),
		)).
		// Watch plugins to be notified about exposed services
		Watches(&greenhousev1alpha1.Plugin{}, handler.EnqueueRequestsFromMapFunc(enqueuePluginForCluster)).
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

// RoundTrip executes the rewritten request and uses the transport created when reconciling the cluster with respective credentials
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
	// errors are logged by pm.Errorhandler
	if err == nil {
		log.FromContext(req.Context()).Info("Forwarded request", "status", resp.StatusCode, "upstreamServiceRouteURL", req.URL.String())
	}
	return
}

func (pm *ProxyManager) rewrite(req *httputil.ProxyRequest) {
	req.SetXForwarded()

	// Create a logger with relevant request details
	l := pm.logger.WithValues(
		"incomingHost", req.In.Host,
		"incomingRequestURL", req.In.URL.String(),
		"incomingMethod", req.In.Method,
	)

	// inject current logger into context before returning
	defer func() {
		req.Out = req.Out.WithContext(log.IntoContext(req.Out.Context(), l))
	}()

	// Extract cluster from the incoming request host
	cluster, err := common.ExtractCluster(req.In.Host)
	if err != nil {
		l.Error(err, "Failed to extract cluster from host", "host", req.In.Host)
		return
	}

	// Retrieve the upstream service route for the cluster
	route, ok := pm.GetClusterRoute(cluster, "https://"+req.In.Host)
	if !ok {
		l.Info("No route found for cluster and URL", "cluster", cluster, "incomingRequestURL", req.In.URL.String())
		return
	}
	upstreamServiceRouteURL := route.url

	// Ensure the outgoing request URL is properly updated
	if !strings.HasPrefix(req.Out.URL.Path, upstreamServiceRouteURL.Path) {
		// Append the original request path to the upstream service route URL path
		req.Out.URL.Path = strings.TrimSuffix(upstreamServiceRouteURL.Path, "/") + req.Out.URL.Path
	}

	// Set the correct upstream service route URL details
	req.Out.URL.Scheme = upstreamServiceRouteURL.Scheme
	req.Out.URL.Host = upstreamServiceRouteURL.Host
	req.Out.Host = upstreamServiceRouteURL.Host

	// Inject the cluster into the outgoing request context
	ctx := context.WithValue(req.Out.Context(), contextClusterKey{}, cluster)
	ctx = log.IntoContext(ctx, l)

	req.Out = req.Out.WithContext(ctx)

	// Log the successful rewrite for debugging purposes
	l.Info("Request rewrite completed",
		"cluster", cluster,
		"namespace", route.namespace,
		"serviceName", route.serviceName,
		"upstreamServiceRouteURL", req.Out.URL.String(),
	)
}

// modifyResponse strips the k8s API server proxy path prepended to the location header during redirects:
// https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apimachinery/pkg/util/proxy/transport.go#L113
func (pm *ProxyManager) modifyResponse(resp *http.Response) error {
	logger := log.FromContext(resp.Request.Context())
	logger.Info("Modifying response", "statusCode", resp.StatusCode, "originalLocation", resp.Header.Get("Location"))

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

// GetClusterRoute returns the route information for a given cluster and incoming URL
func (pm *ProxyManager) GetClusterRoute(cluster, inURL string) (*route, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	cls, ok := pm.clusters[cluster]
	if !ok {
		return nil, false
	}
	getRoute, ok := cls.routes[inURL]
	if !ok {
		return nil, false
	}
	return &getRoute, true
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
