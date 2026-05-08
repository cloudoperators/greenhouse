// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"

	apis "github.com/cloudoperators/greenhouse/api"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
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
			ValidateDeleteFunc: ValidateDeleteCluster,
		},
	)
}

//+kubebuilder:webhook:path=/mutate-greenhouse-sap-v1alpha1-cluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=clusters,verbs=create;update,versions=v1alpha1,name=mcluster.kb.io,admissionReviewVersions=v1

func DefaultCluster(ctx context.Context, _ client.Client, cluster *greenhousev1alpha1.Cluster) error {
	logger := ctrl.LoggerFrom(ctx)

	// default the "greenhouse.sap/cluster" label to be able to select the cluster by label
	labels := cluster.GetLabels()
	if labels == nil {
		labels = make(map[string]string, 1)
		cluster.SetLabels(labels)
	}
	labels[apis.LabelKeyCluster] = cluster.GetName()

	annotations := cluster.GetAnnotations()
	deletionVal, deletionMarked := annotations[apis.MarkClusterDeletionAnnotation]
	_, scheduleExists := annotations[apis.ScheduleClusterDeletionAnnotation]

	// if the deletion annotation is not set, but the schedule exists, remove the schedule
	// it could be that the schedule was set in the past, but the deletion annotation was removed intentionally
	if !deletionMarked && scheduleExists {
		logger.Info("found deletion schedule but no deletion annotation, schedule will be removed", "schedule", annotations[apis.ScheduleClusterDeletionAnnotation])
		delete(annotations, apis.ScheduleClusterDeletionAnnotation)
		return nil
	}
	// if the deletion annotation is set, but the schedule does not exist, set the schedule
	if deletionMarked {
		// if the deletion annotation is empty, remove the annotation and schedule
		// it could be that the deletion annotation was set in the past, but the value was removed intentionally
		if strings.TrimSpace(deletionVal) == "" {
			logger.Info("found deletion annotation with empty value, annotation and schedule will be removed")
			delete(annotations, apis.MarkClusterDeletionAnnotation)
			delete(annotations, apis.ScheduleClusterDeletionAnnotation)
			return nil
		}
		if !scheduleExists {
			// resource was marked for deletion so a schedule should be set to 48hrs from now
			annotations[apis.ScheduleClusterDeletionAnnotation] = time.Now().Add(48 * time.Hour).Format(time.DateTime)
			logger.Info("found deletion annotation, setting schedule", "schedule", annotations[apis.ScheduleClusterDeletionAnnotation])
			return nil
		}
	}
	return nil
}

//+kubebuilder:webhook:path=/validate-greenhouse-sap-v1alpha1-cluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=clusters,verbs=create;update;delete,versions=v1alpha1,name=vcluster.kb.io,admissionReviewVersions=v1

// ValidateCreateCluster disallows creating clusters with deletionMarked or deletionSchedule annotations
func ValidateCreateCluster(ctx context.Context, c client.Client, cluster *greenhousev1alpha1.Cluster) (admission.Warnings, error) {
	logger := ctrl.LoggerFrom(ctx)
	if err := webhook.InvalidateDoubleDashesInName(cluster, logger); err != nil {
		return nil, err
	}
	// capping the name at 40 chars, so we ensure to get unique urls for exposed services per cluster. service-name/namespace hash needs to fit (max 63 chars)
	if err := webhook.CapName(cluster, logger, 40); err != nil {
		return nil, err
	}
	annotations := cluster.GetAnnotations()
	_, deletionMarked := annotations[apis.MarkClusterDeletionAnnotation]
	_, scheduleExists := annotations[apis.ScheduleClusterDeletionAnnotation]
	if deletionMarked || scheduleExists {
		err := apierrors.NewInvalid(cluster.GroupVersionKind().GroupKind(), cluster.GetName(), nil)
		logger.Error(err, "found deletion annotation on cluster creation, admission will be denied")
		return admission.Warnings{"you cannot create a cluster with deletion annotation"}, err
	}

	labelValidationWarning := webhook.ValidateLabelOwnedBy(ctx, c, cluster)
	if labelValidationWarning != "" {
		return admission.Warnings{"Cluster should have a support-group Team set as its owner", labelValidationWarning}, nil
	}

	return nil, nil
}

// ValidateUpdateCluster disallows cluster updates with invalid deletion schedules
func ValidateUpdateCluster(ctx context.Context, c client.Client, _, cluster *greenhousev1alpha1.Cluster) (admission.Warnings, error) {
	logger := ctrl.LoggerFrom(ctx)
	_, _, err := clientutil.ExtractDeletionSchedule(cluster.GetAnnotations())
	if err != nil {
		err = apierrors.NewBadRequest("invalid deletion schedule provided - expected format 2006-01-02 15:04:05")
		logger.Error(err, "update request denied", "cluster", cluster.GetName())
		return admission.Warnings{"update is not allowed"}, err
	}
	labelValidationWarning := webhook.ValidateLabelOwnedBy(ctx, c, cluster)
	if labelValidationWarning != "" {
		return admission.Warnings{"Cluster should have a support-group Team set as its owner", labelValidationWarning}, nil
	}
	return nil, nil
}

// ValidateDeleteCluster only allows deletion requests for clusters with a deletion schedule timestamp past now.
func ValidateDeleteCluster(ctx context.Context, _ client.Client, cluster *greenhousev1alpha1.Cluster) (admission.Warnings, error) {
	now := time.Now()
	logger := ctrl.LoggerFrom(ctx)
	groupResource := schema.GroupResource{
		Group:    cluster.GroupVersionKind().Group,
		Resource: "clusters",
	}
	annotations := cluster.GetAnnotations()
	isScheduled, schedule, err := clientutil.ExtractDeletionSchedule(annotations)
	if err != nil {
		err = apierrors.NewForbidden(groupResource, cluster.GetName(), err)
		logger.Error(err, "deletion request denied", "cluster", cluster.GetName())
		return admission.Warnings{"deletion is not allowed"}, err
	}

	if !isScheduled {
		msg := fmt.Sprintf("schedule deletion by setting the deletion annotation - %s to true", apis.MarkClusterDeletionAnnotation)
		err = apierrors.NewForbidden(groupResource, cluster.GetName(), errors.New(msg))
		logger.Error(err, "deletion request denied", "cluster", cluster.GetName())
		return admission.Warnings{"deletion is not allowed"}, err
	}

	logger.Info("found deletion annotation, schedule", "schedule", schedule)
	canDelete, err := clientutil.ShouldProceedDeletion(now, schedule)
	// ideally we would not hit the err condition as invalid format of time.DateTime should be caught in the ExtractDeletionSchedule function
	if err != nil {
		logger.Error(err, "error while checking deletion schedule")
		err = apierrors.NewInvalid(
			cluster.GroupVersionKind().GroupKind(),
			cluster.GetName(),
			field.ErrorList{field.Invalid(field.NewPath("metadata", "annotations"), annotations, err.Error())})
		return admission.Warnings{"invalid deletion schedule"}, err
	}
	if canDelete {
		logger.Info("deletion request allowed", "cluster", cluster.GetName())
		return nil, nil
	}
	err = apierrors.NewForbidden(groupResource, cluster.GetName(), errors.New("deletion scheduled at - "+schedule.String()))
	logger.Error(err, "deletion request denied", "cluster", cluster.GetName())
	return admission.Warnings{"deletion is not allowed"}, err
}
