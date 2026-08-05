// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"

	apis "github.com/cloudoperators/greenhouse/api"
	"github.com/cloudoperators/greenhouse/internal/webhook"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

// Webhook for the Cluster custom resource.

func SetupClusterWebhookWithManager(mgr ctrl.Manager) error {
	return webhook.SetupWebhook(mgr,
		&greenhousev1alpha1.Cluster{},
		webhook.WebhookFuncs[*greenhousev1alpha1.Cluster]{
			DefaultFunc:        DefaultCluster,
			ValidateCreateFunc: ValidateCreateCluster,
			ValidateUpdateFunc: ValidateUpdateCluster,
		},
	)
}

//+kubebuilder:webhook:path=/mutate-greenhouse-sap-v1alpha1-cluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=clusters,verbs=create;update,versions=v1alpha1,name=mcluster.kb.io,admissionReviewVersions=v1

func DefaultCluster(ctx context.Context, _ client.Client, cluster *greenhousev1alpha1.Cluster) error {
	// default the "greenhouse.sap/cluster" label to be able to select the cluster by label
	labels := cluster.GetLabels()
	if labels == nil {
		labels = make(map[string]string, 1)
		cluster.SetLabels(labels)
	}
	labels[apis.LabelKeyCluster] = cluster.GetName()

	return nil
}

//+kubebuilder:webhook:path=/validate-greenhouse-sap-v1alpha1-cluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=clusters,verbs=create;update,versions=v1alpha1,name=vcluster.kb.io,admissionReviewVersions=v1

// ValidateCreateCluster validates the cluster name and its support-group ownership label on create
func ValidateCreateCluster(ctx context.Context, c client.Client, cluster *greenhousev1alpha1.Cluster) (admission.Warnings, error) {
	logger := ctrl.LoggerFrom(ctx)
	if err := webhook.InvalidateDoubleDashesInName(cluster, logger); err != nil {
		return nil, err
	}
	// capping the name at 40 chars, so we ensure to get unique urls for exposed services per cluster. service-name/namespace hash needs to fit (max 63 chars)
	if err := webhook.CapName(cluster, logger, 40); err != nil {
		return nil, err
	}

	labelValidationWarning := webhook.ValidateLabelOwnedBy(ctx, c, cluster)
	if labelValidationWarning != "" {
		return admission.Warnings{"Cluster should have a support-group Team set as its owner", labelValidationWarning}, nil
	}

	return nil, nil
}

// ValidateUpdateCluster disallows cluster updates with invalid ownership labels
func ValidateUpdateCluster(ctx context.Context, c client.Client, _, cluster *greenhousev1alpha1.Cluster) (admission.Warnings, error) {
	labelValidationWarning := webhook.ValidateLabelOwnedBy(ctx, c, cluster)
	if labelValidationWarning != "" {
		return admission.Warnings{"Cluster should have a support-group Team set as its owner", labelValidationWarning}, nil
	}
	return nil, nil
}
