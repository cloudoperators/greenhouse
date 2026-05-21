// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	reasonNoSupportGroupClaims   = "no_support_group_claims"
	reasonNoOwnedByLabel         = "no_owned_by_label"
	reasonSupportGroupMismatch   = "support_group_mismatch"
	reasonObjectNotFound         = "object_not_found"
	reasonKindResolutionFailed   = "kind_resolution_failed"
	reasonDecodeError            = "decode_error"
	reasonMissingAttributes      = "missing_attributes"
	reasonServiceAccountNotFound = "service_account_not_found"
)

var (
	authzRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "greenhouse_authz_requests_total",
			Help: "Total number of authorization requests labeled by result and verb.",
		},
		[]string{"result", "verb"},
	)

	authzDeniedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "greenhouse_authz_denied_total",
			Help: "Total number of denied authorization requests by reason.",
		},
		[]string{"reason"},
	)

	authzAccessTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "greenhouse_authz_access_total",
			Help: "Authorization decisions by support_group, resource and result.",
		},
		[]string{"support_group", "resource", "result"},
	)

	authzKubeFetchErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "greenhouse_authz_kube_fetch_errors_total",
			Help: "Total number of errors fetching objects from the Kubernetes API server.",
		},
		[]string{"reason"},
	)

	authzKindResolutionErrorsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "greenhouse_authz_kind_resolution_errors_total",
			Help: "Total number of errors resolving GVR to GVK.",
		},
	)

	authzRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "greenhouse_authz_request_duration_seconds",
			Help:    "End-to-end request duration of authorization requests.",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5},
		},
		[]string{"code", "method"},
	)
)

func init() {
	crmetrics.Registry.MustRegister(
		authzRequestsTotal,
		authzDeniedTotal,
		authzAccessTotal,
		authzKubeFetchErrorsTotal,
		authzKindResolutionErrorsTotal,
		authzRequestDuration,
	)
}

// instrumentHandler wraps h with prometheus duration instrumentation using fine-grained buckets.
func instrumentHandler(h http.Handler) http.Handler {
	return promhttp.InstrumentHandlerDuration(authzRequestDuration, h)
}

func recordAllowed(verb, resource, supportGroup string) {
	authzRequestsTotal.With(prometheus.Labels{"result": "allowed", "verb": verb}).Inc()
	authzAccessTotal.With(prometheus.Labels{"support_group": supportGroup, "resource": resource, "result": "allowed"}).Inc()
}

func recordDenied(verb, resource, reason string, supportGroups []string) {
	authzRequestsTotal.With(prometheus.Labels{"result": "denied", "verb": verb}).Inc()
	authzDeniedTotal.With(prometheus.Labels{"reason": reason}).Inc()
	for _, sg := range supportGroups {
		authzAccessTotal.With(prometheus.Labels{"support_group": sg, "resource": resource, "result": "denied"}).Inc()
	}
}
