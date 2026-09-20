// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package proxy

import (
	"context"
	"fmt"
	stdlog "log"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strings"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudoperators/greenhouse/internal/common"
)

type logrWriter struct {
	logger logr.Logger
}

func (w *logrWriter) Write(p []byte) (int, error) {
	w.logger.Info("reverse proxy internal", "msg", strings.TrimRight(string(p), "\n"))
	return len(p), nil
}

// contextClusterKey carries the resolved cluster name from rewrite to RoundTrip.
// httputil.ReverseProxy rewrites the outgoing Host to the upstream apiserver
// before RoundTrip runs, so the original cluster host is no longer available
// there; the context is the only channel between the two hooks.
type contextClusterKey struct{}

// apiServerProxyPathRegex matches the kube-apiserver service-proxy path prefix
// the apiserver prepends to redirect Location headers.
var apiServerProxyPathRegex = regexp.MustCompile(`/api/v1/namespaces/[^/]+/services/[^/]+/proxy/`)

// ReverseProxy returns the reverse proxy that forwards requests to the cluster
// resolved for the incoming host.
func (pm *PmManager) ReverseProxy() *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Rewrite:        pm.rewrite,
		ModifyResponse: pm.modifyResponse,
		Transport:      pm,
		FlushInterval:  -1,
		ErrorHandler:   pm.errorHandler,
		ErrorLog:       stdlog.New(&logrWriter{logger: pm.logger}, "", 0),
	}
}

// RoundTrip forwards the rewritten request using the transport stored for the
// cluster carried in the request context.
func (pm *PmManager) RoundTrip(req *http.Request) (*http.Response, error) {
	cluster, ok := req.Context().Value(contextClusterKey{}).(string)
	if !ok {
		return nil, fmt.Errorf("no upstream found for: %s", req.URL.String())
	}
	transport, ok := pm.store.Transport(cluster)
	if !ok {
		return nil, fmt.Errorf("cluster %s not found", cluster)
	}
	resp, err := transport.RoundTrip(req)
	// errors are logged by pm.errorHandler
	if err == nil {
		log.FromContext(req.Context()).Info("forwarded request", "status", resp.StatusCode, "upstreamServiceRouteURL", req.URL.String())
	}
	return resp, err
}

// rewrite resolves the cluster and route for the incoming host, points the
// outgoing request at the upstream apiserver service-proxy URL, and injects the
// cluster into the outgoing context for RoundTrip.
func (pm *PmManager) rewrite(req *httputil.ProxyRequest) {
	req.SetXForwarded()

	l := pm.logger.WithValues(
		"incomingHost", req.In.Host,
		"incomingRequestURL", req.In.URL.String(),
		"incomingMethod", req.In.Method,
	)
	defer func() {
		req.Out = req.Out.WithContext(log.IntoContext(req.Out.Context(), l))
	}()

	cluster, err := common.ExtractCluster(req.In.Host)
	if err != nil {
		l.Error(err, "failed to extract cluster from host", "host", req.In.Host)
		return
	}

	r, ok := pm.store.Route(cluster, "https://"+req.In.Host)
	if !ok {
		l.Info("no route found for cluster and URL", "cluster", cluster, "incomingRequestURL", req.In.URL.String())
		return
	}
	upstream := r.url

	if !strings.HasPrefix(req.Out.URL.Path, upstream.Path) {
		req.Out.URL.Path = strings.TrimSuffix(upstream.Path, "/") + req.Out.URL.Path
	}
	req.Out.URL.Scheme = upstream.Scheme
	req.Out.URL.Host = upstream.Host
	req.Out.Host = upstream.Host

	ctx := context.WithValue(req.Out.Context(), contextClusterKey{}, cluster)
	ctx = log.IntoContext(ctx, l)
	req.Out = req.Out.WithContext(ctx)

	l.Info("request rewrite completed",
		"cluster", cluster,
		"namespace", r.namespace,
		"serviceName", r.serviceName,
		"upstreamServiceRouteURL", req.Out.URL.String(),
	)
}

// modifyResponse strips the kube-apiserver service-proxy path prefix that the
// apiserver prepends to redirect Location headers.
func (pm *PmManager) modifyResponse(resp *http.Response) error {
	logger := log.FromContext(resp.Request.Context())
	logger.Info("modifying response", "statusCode", resp.StatusCode, "originalLocation", resp.Header.Get("Location"))

	if location := resp.Header.Get("Location"); location != "" {
		location = apiServerProxyPathRegex.ReplaceAllString(location, "/")
		resp.Header.Set("Location", location)
		logger.Info("rewrote location header", "location", location)
	}
	return nil
}

// errorHandler logs the proxy failure and returns a bad gateway.
func (pm *PmManager) errorHandler(rw http.ResponseWriter, req *http.Request, err error) {
	logger := pm.logger
	if l, lerr := logr.FromContext(req.Context()); lerr == nil {
		logger = l
	}
	logger.Info("proxy failure", "err", err)
	rw.WriteHeader(http.StatusBadGateway)
}
