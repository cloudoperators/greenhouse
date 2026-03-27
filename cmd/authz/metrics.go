// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	reasonNoSupportGroupClaims = "no_support_group_claims"
	reasonNoOwnedByLabel       = "no_owned_by_label"
	reasonSupportGroupMismatch = "support_group_mismatch"
	reasonObjectNotFound       = "object_not_found"
	reasonKindResolutionFailed = "kind_resolution_failed"
	reasonDecodeError          = "decode_error"
	reasonMissingAttributes    = "missing_attributes"
)

var (
	authzRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "greenhouse_authz_requests_total",
			Help: "Total number of authorization requests labeled by result and verb.",
		},
		[]string{"result", "verb"},
	)

	authzRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "greenhouse_authz_request_duration_seconds",
			Help:    "Duration of authorization requests in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"result"},
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
)

func init() {
	crmetrics.Registry.MustRegister(
		authzRequestsTotal,
		authzRequestDuration,
		authzDeniedTotal,
		authzAccessTotal,
		authzKubeFetchErrorsTotal,
		authzKindResolutionErrorsTotal,
	)
}

func recordAllowed(verb, resource, supportGroup string, start time.Time) {
	authzRequestsTotal.With(prometheus.Labels{"result": "allowed", "verb": verb}).Inc()
	authzRequestDuration.With(prometheus.Labels{"result": "allowed"}).Observe(time.Since(start).Seconds())
	authzAccessTotal.With(prometheus.Labels{"support_group": supportGroup, "resource": resource, "result": "allowed"}).Inc()
}

func recordDenied(verb, resource, reason string, supportGroups []string, start time.Time) {
	authzRequestsTotal.With(prometheus.Labels{"result": "denied", "verb": verb}).Inc()
	authzRequestDuration.With(prometheus.Labels{"result": "denied"}).Observe(time.Since(start).Seconds())
	authzDeniedTotal.With(prometheus.Labels{"reason": reason}).Inc()
	for _, sg := range supportGroups {
		authzAccessTotal.With(prometheus.Labels{"support_group": sg, "resource": resource, "result": "denied"}).Inc()
	}
}
