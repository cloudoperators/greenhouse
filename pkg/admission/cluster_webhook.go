// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package admission

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudoperators/greenhouse/pkg/apis"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

// Webhook for the Cluster custom resource.

func SetupClusterWebhookWithManager(mgr ctrl.Manager) error {
	return setupWebhook(mgr,
		&greenhousev1alpha1.Cluster{},
		webhookFuncs{
			defaultFunc:        DefaultCluster,
			validateCreateFunc: ValidateCreateCluster,
			validateUpdateFunc: ValidateUpdateCluster,
			validateDeleteFunc: ValidateDeleteCluster,
		},
	)
}

//+kubebuilder:webhook:path=/mutate-greenhouse-sap-v1alpha1-cluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=clusters,verbs=create;update,versions=v1alpha1,name=mcluster.kb.io,admissionReviewVersions=v1

func DefaultCluster(ctx context.Context, _ client.Client, obj runtime.Object) error {
	logger := ctrl.LoggerFrom(ctx)
	cluster, ok := obj.(*greenhousev1alpha1.Cluster)
	if !ok {
		return nil
	}
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

func ValidateCreateCluster(ctx context.Context, _ client.Client, obj runtime.Object) (admission.Warnings, error) {
	logger := ctrl.LoggerFrom(ctx)
	cluster, ok := obj.(*greenhousev1alpha1.Cluster)
	if !ok {
		return nil, nil
	}
	annotations := cluster.GetAnnotations()
	_, deletionMarked := annotations[apis.MarkClusterDeletionAnnotation]
	_, scheduleExists := annotations[apis.ScheduleClusterDeletionAnnotation]
	if deletionMarked || scheduleExists {
		err := apierrors.NewInvalid(cluster.GroupVersionKind().GroupKind(), cluster.GetName(), nil)
		logger.Error(err, "found deletion annotation on cluster creation, admission will be denied")
		return admission.Warnings{"you cannot create a cluster with deletion annotation"}, err
	}
	return nil, nil
}

func ValidateUpdateCluster(_ context.Context, _ client.Client, _, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func ValidateDeleteCluster(ctx context.Context, _ client.Client, obj runtime.Object) (admission.Warnings, error) {
	now := time.Now()
	logger := ctrl.LoggerFrom(ctx)
	cluster, ok := obj.(*greenhousev1alpha1.Cluster)
	if !ok {
		return nil, nil
	}
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
