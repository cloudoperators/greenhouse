// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

var (
	// ReadyGauge indicates whether the catalog is ready
	// A catalog is considered ready when all its sources and their resources are ready
	ReadyGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "greenhouse_catalog_ready",
			Help: "Indicates whether the catalog is ready (1 = ready, 0 = not ready)",
		},
		[]string{"catalog", "namespace", "owned_by"})

	// MissingSecretGauge indicates whether a specific catalog source is missing its secret
	// This metric is per-source, allowing you to identify which repository is affected
	MissingSecretGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "greenhouse_catalog_missing_secret",
			Help: "Indicates whether a catalog source is missing its secret (1 = missing, 0 = present)",
		},
		[]string{"catalog", "namespace", "owned_by", "repository_url", "git_repo_name", "ref", "secret_name"})

	// AuthErrorGauge indicates whether a catalog source has an authentication error
	// This metric is per-source, allowing you to identify which repository has auth issues
	AuthErrorGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "greenhouse_catalog_auth_error",
			Help: "Indicates whether a catalog source has an authentication error (1 = error, 0 = no error)",
		},
		[]string{"catalog", "namespace", "owned_by", "repository_url", "git_repo_name", "ref"})
)

func init() {
	crmetrics.Registry.MustRegister(ReadyGauge)
	crmetrics.Registry.MustRegister(MissingSecretGauge)
	crmetrics.Registry.MustRegister(AuthErrorGauge)
}

// UpdateCatalogReadyMetric updates the catalog ready metric based on the catalog's ready condition
func UpdateCatalogReadyMetric(catalog *greenhousev1alpha1.Catalog) {
	catalogReadyLabels := prometheus.Labels{
		"catalog":   catalog.Name,
		"namespace": catalog.Namespace,
		"owned_by":  catalog.Labels[greenhouseapis.LabelKeyOwnedBy],
	}

	// Check if the catalog has a Ready condition set to True
	readyCondition := catalog.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
	if readyCondition != nil && readyCondition.Status == metav1.ConditionTrue {
		ReadyGauge.With(catalogReadyLabels).Set(1)
	} else {
		ReadyGauge.With(catalogReadyLabels).Set(0)
	}
}

// UpdateCatalogMissingSecretMetric updates the missing secret metric for a specific catalog source
func UpdateCatalogMissingSecretMetric(catalog *greenhousev1alpha1.Catalog, repositoryURL, gitRepoName, ref, secretName string, isMissing bool) {
	catalogLabels := prometheus.Labels{
		"catalog":        catalog.Name,
		"namespace":      catalog.Namespace,
		"owned_by":       catalog.Labels[greenhouseapis.LabelKeyOwnedBy],
		"repository_url": repositoryURL,
		"git_repo_name":  gitRepoName,
		"ref":            ref,
		"secret_name":    secretName,
	}
	if isMissing {
		MissingSecretGauge.With(catalogLabels).Set(1)
	} else {
		MissingSecretGauge.With(catalogLabels).Set(0)
	}
}

// UpdateCatalogAuthErrorMetric updates the authentication error metric for a specific catalog source
func UpdateCatalogAuthErrorMetric(catalog *greenhousev1alpha1.Catalog, repositoryURL, gitRepoName, ref string, hasError bool) {
	catalogLabels := prometheus.Labels{
		"catalog":        catalog.Name,
		"namespace":      catalog.Namespace,
		"owned_by":       catalog.Labels[greenhouseapis.LabelKeyOwnedBy],
		"repository_url": repositoryURL,
		"git_repo_name":  gitRepoName,
		"ref":            ref,
	}
	if hasError {
		AuthErrorGauge.With(catalogLabels).Set(1)
	} else {
		AuthErrorGauge.With(catalogLabels).Set(0)
	}
}
