// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

func ComputeOwnerLabelCondition(ctx context.Context, c client.Client, resourceObj metav1.Object, ownerLabelSetCondition greenhousemetav1alpha1.Condition) greenhousemetav1alpha1.Condition {
	namespace := resourceObj.GetNamespace()
	if namespace == "" {
		ownerLabelSetCondition.Message = "Resource namespace is required to validate the owner"
		ownerLabelSetCondition.Status = metav1.ConditionFalse
		return ownerLabelSetCondition
	}

	ownerName, ok := resourceObj.GetLabels()[greenhouseapis.LabelKeyOwnedBy]
	if !ok || ownerName == "" {
		ownerLabelSetCondition.Reason = greenhousemetav1alpha1.OwnerLabelMissingReason
		ownerLabelSetCondition.Message = fmt.Sprintf("Label %s is missing", greenhouseapis.LabelKeyOwnedBy)
		ownerLabelSetCondition.Status = metav1.ConditionFalse
		return ownerLabelSetCondition
	}

	team := new(greenhousev1alpha1.Team)
	err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: ownerName}, team)
	switch {
	case apierrors.IsNotFound(err):
		ownerLabelSetCondition.Reason = greenhousemetav1alpha1.OwnerLabelSetToNotExistingTeamReason
		ownerLabelSetCondition.Message = fmt.Sprintf("team %s does not exist in resource namespace", ownerName)
		ownerLabelSetCondition.Status = metav1.ConditionFalse
		return ownerLabelSetCondition
	case err != nil:
		ownerLabelSetCondition.Reason = greenhousemetav1alpha1.OwnerLabelSetToNotExistingTeamReason
		ownerLabelSetCondition.Message = fmt.Sprintf("team %s could not be retrieved", ownerName)
		ownerLabelSetCondition.Status = metav1.ConditionFalse
		return ownerLabelSetCondition
	}
	supportGroup, ok := team.Labels[greenhouseapis.LabelKeySupportGroup]
	if !ok || supportGroup != "true" {
		ownerLabelSetCondition.Reason = greenhousemetav1alpha1.OwnerLabelSetToNonSupportGroupTeamReason
		ownerLabelSetCondition.Message = fmt.Sprintf("owner team %s should be a support group", ownerName)
		ownerLabelSetCondition.Status = metav1.ConditionFalse
		return ownerLabelSetCondition
	}

	ownerLabelSetCondition.Status = metav1.ConditionTrue
	return ownerLabelSetCondition
}
