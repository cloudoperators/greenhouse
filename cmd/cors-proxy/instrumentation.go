// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func instrumentHandler(next http.Handler, registry *prometheus.Registry) http.Handler {
	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Number of HTTP requests",
		},
		[]string{"code", "method"},
	)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "A histogram of latencies for requests.",
			Buckets: []float64{.25, .5, 1, 2.5, 5, 10},
		},
		[]string{"code", "method"},
	)

	responseSize := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "A histogram of response sizes for requests.",
			Buckets: []float64{0, 512, 4096, 16384, 65536, 262144, 1048576, 4194304, 16777216},
		},
		[]string{"code", "method"},
	)

	// Register all of the metrics in the standard registry.
	registry.MustRegister(requestsTotal, requestDuration, responseSize)

	return promhttp.InstrumentHandlerCounter(
		requestsTotal, promhttp.InstrumentHandlerDuration(
			requestDuration,
			promhttp.InstrumentHandlerResponseSize(
				responseSize,
				next,
			),
		),
	)
}
