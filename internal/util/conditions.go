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

func ComputeOwnerLabelCondition(ctx context.Context, c client.Client, resourceObj metav1.Object) greenhousemetav1alpha1.Condition {
	namespace := resourceObj.GetNamespace()
	if namespace == "" {
		return greenhousemetav1alpha1.FalseCondition(greenhousemetav1alpha1.OwnerLabelSetCondition,
			"", "Resource namespace is required to validate the owner")
	}

	ownerName, ok := resourceObj.GetLabels()[greenhouseapis.LabelKeyOwnedBy]
	if !ok || ownerName == "" {
		return greenhousemetav1alpha1.FalseCondition(greenhousemetav1alpha1.OwnerLabelSetCondition,
			greenhousemetav1alpha1.OwnerLabelMissingReason,
			fmt.Sprintf("Label %s is missing", greenhouseapis.LabelKeyOwnedBy))
	}

	team := new(greenhousev1alpha1.Team)
	err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: ownerName}, team)
	switch {
	case apierrors.IsNotFound(err):
		return greenhousemetav1alpha1.FalseCondition(greenhousemetav1alpha1.OwnerLabelSetCondition,
			greenhousemetav1alpha1.OwnerLabelSetToNotExistingTeamReason,
			fmt.Sprintf("team %s does not exist in resource namespace", ownerName))
	case err != nil:
		return greenhousemetav1alpha1.FalseCondition(greenhousemetav1alpha1.OwnerLabelSetCondition,
			greenhousemetav1alpha1.OwnerLabelSetToNotExistingTeamReason,
			fmt.Sprintf("team %s could not be retrieved", ownerName))
	}
	supportGroup, ok := team.Labels[greenhouseapis.LabelKeySupportGroup]
	if !ok || supportGroup != "true" {
		return greenhousemetav1alpha1.FalseCondition(greenhousemetav1alpha1.OwnerLabelSetCondition,
			greenhousemetav1alpha1.OwnerLabelSetToNonSupportGroupTeamReason,
			fmt.Sprintf("owner team %s should be a support group", ownerName))
	}

	return greenhousemetav1alpha1.TrueCondition(greenhousemetav1alpha1.OwnerLabelSetCondition, "", "")
}
