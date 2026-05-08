// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"strings"

	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

const (
	ReasonOK            = "OK"
	ReasonAuthError     = "AuthError"
	ReasonSecretMissing = "SecretMissing"
	ReasonUnknown       = "Unknown"
)

var (
	// readyGauge indicates whether the catalog is ready with a reason field
	// Reason values: OK (ready), AuthError (auth issues), SecretMissing (missing secrets)
	readyGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "greenhouse_catalog_ready",
			Help: "Indicates whether the catalog is ready (1 = ready, 0 = not ready) with reason field (OK, AuthError, SecretMissing, Unknown)",
		},
		[]string{"catalog", "organization", "owned_by", "reason"})
)

func init() {
	crmetrics.Registry.MustRegister(readyGauge)
}

// updateCatalogReadyMetric updates the catalog ready metric based on the catalog's ready condition
// It determines the reason from the catalog status and sets the appropriate metric value
func updateCatalogReadyMetric(catalog *greenhousev1alpha1.Catalog) {
	ownedBy := ""
	if catalog.Labels != nil {
		ownedBy = catalog.Labels[greenhouseapis.LabelKeyOwnedBy]
	}

	// Determine the current state and reason
	readyCondition := catalog.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
	isReady := readyCondition != nil && readyCondition.Status == metav1.ConditionTrue

	var reason string
	if isReady {
		reason = ReasonOK
	} else {
		reason = determineReason(catalog)
	}

	// Clear reason values for this catalog
	for _, r := range []string{ReasonOK, ReasonAuthError, ReasonSecretMissing, ReasonUnknown} {
		if r != reason {
			readyGauge.DeleteLabelValues(catalog.Name, catalog.Namespace, ownedBy, r)
		}
	}

	// Set the current metric
	value := 0.0
	if isReady {
		value = 1.0
	}
	readyGauge.WithLabelValues(catalog.Name, catalog.Namespace, ownedBy, reason).Set(value)
}

// deleteCatalogReadyMetric removes all metric series for a catalog when it's deleted
func deleteCatalogReadyMetric(catalog *greenhousev1alpha1.Catalog) {
	ownedBy := ""
	if catalog.Labels != nil {
		ownedBy = catalog.Labels[greenhouseapis.LabelKeyOwnedBy]
	}

	// Delete all possible reason values for this catalog
	for _, r := range []string{ReasonOK, ReasonAuthError, ReasonSecretMissing, ReasonUnknown} {
		readyGauge.DeleteLabelValues(catalog.Name, catalog.Namespace, ownedBy, r)
	}
}

// determineReason analyzes the catalog status to determine the appropriate reason
// Checks GitRepository kind in the inventory for auth errors
// Checks Ready condition message for secret errors
func determineReason(catalog *greenhousev1alpha1.Catalog) string {
	hasAuthError := false
	hasMissingSecret := false

	// Check GitRepository sources in the inventory for authentication errors
	for _, sourceList := range catalog.Status.Inventory {
		for _, source := range sourceList {
			if source.Kind == sourcev1.GitRepositoryKind && source.Ready == metav1.ConditionFalse && source.Message != "" {
				msg := strings.ToLower(source.Message)

				// Check for authentication errors
				if strings.Contains(msg, "authentication") ||
					strings.Contains(msg, "auth") ||
					strings.Contains(msg, "unauthorized") ||
					strings.Contains(msg, "forbidden") ||
					strings.Contains(msg, "credentials") {
					hasAuthError = true
				}
			}
		}
	}

	// Check the Ready condition message for missing secret errors
	readyCondition := catalog.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
	if readyCondition != nil && readyCondition.Message != "" {
		msg := strings.ToLower(readyCondition.Message)
		if strings.Contains(msg, "secret") && strings.Contains(msg, "not found") {
			hasMissingSecret = true
		}
	}

	if hasMissingSecret {
		return ReasonSecretMissing
	}
	if hasAuthError {
		return ReasonAuthError
	}

	// If no specific error was found, return "Unknown" instead of "OK"
	// This function is only called for not-ready catalogs
	return ReasonUnknown
}
