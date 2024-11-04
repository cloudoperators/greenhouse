// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/cloudoperators/greenhouse/pkg/common"
)

var (
	clusterFromContext = promhttp.WithLabelFromCtx("cluster", func(ctx context.Context) string {
		cluster, _ := ctx.Value(contextClusterKey{}).(string) //nolint:errcheck
		return cluster
	})

	namespaceFromContext = promhttp.WithLabelFromCtx("namespace", func(ctx context.Context) string {
		namespace, _ := ctx.Value(contextNamespaceKey{}).(string) //nolint:errcheck
		return namespace
	})

	nameFromContext = promhttp.WithLabelFromCtx("name", func(ctx context.Context) string {
		name, _ := ctx.Value(contextNameKey{}).(string) //nolint:errcheck
		return name
	})
)

func InstrumentHandler(pm *ProxyManager, registry prometheus.Registerer) http.Handler {
	next := pm.ReverseProxy()
	requestCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "A counter for requests to the wrapped handler.",
		},
		[]string{"code", "method", "cluster", "namespace", "name"},
	)
	registry.MustRegister(requestCounter)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "A histogram of latencies for requests.",
			Buckets: []float64{.25, .5, 1, 2.5, 5, 10},
		},
		[]string{"code", "method", "cluster", "namespace", "name"},
	)
	registry.MustRegister(requestDuration)

	responseSizeHistogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "A histogram of response sizes for requests.",
			Buckets: []float64{0, 512, 4096, 16384, 65536, 262144, 1048576, 4194304, 16777216},
		},
		[]string{"code", "method", "cluster", "namespace", "name"},
	)
	registry.MustRegister(responseSizeHistogram)

	injector := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if cluster, err := common.ExtractCluster(req.Host); err == nil {
				ctx := req.Context()
				ctx = context.WithValue(ctx, contextClusterKey{}, cluster)
				route, found := pm.GetClusterRoute(cluster, req.Host)
				if found {
					ctx = context.WithValue(ctx, contextNamespaceKey{}, route.namespace)
					ctx = context.WithValue(ctx, contextNameKey{}, route.serviceName)
				}
				req = req.WithContext(ctx)
			}
			next.ServeHTTP(rw, req)
		})
	}

	return injector(
		promhttp.InstrumentHandlerCounter(requestCounter,
			promhttp.InstrumentHandlerDuration(requestDuration,
				promhttp.InstrumentHandlerResponseSize(responseSizeHistogram,
					next,
					clusterFromContext, namespaceFromContext, nameFromContext,
				),
				clusterFromContext, namespaceFromContext, nameFromContext,
			),
			clusterFromContext, namespaceFromContext, nameFromContext,
		),
	)
}
