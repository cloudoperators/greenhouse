// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugindefinition

import (
	"context"

	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcecontroller "github.com/fluxcd/source-controller/api/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

// setHelmRepositoryReadyCondition checks the HelmRepository status and sets the HelmRepositoryReady condition on the given object.
func setHelmRepositoryReadyCondition(ctx context.Context, k8sClient client.Client, obj lifecycle.RuntimeObject, helmRepo *sourcecontroller.HelmRepository) {
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(helmRepo), helmRepo); err != nil {
		obj.SetCondition(greenhousemetav1alpha1.UnknownCondition(
			greenhousev1alpha1.HelmRepositoryReadyCondition, "", "unable to fetch HelmRepository status"))
		return
	}

	readyCondition := meta.FindStatusCondition(helmRepo.Status.Conditions, fluxmeta.ReadyCondition)
	switch {
	case readyCondition == nil:
		obj.SetCondition(greenhousemetav1alpha1.UnknownCondition(
			greenhousev1alpha1.HelmRepositoryReadyCondition, "", "HelmRepository status pending"))
	case readyCondition.Status == metav1.ConditionTrue:
		obj.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousev1alpha1.HelmRepositoryReadyCondition, "", "HelmRepository is ready"))
	default:
		obj.SetCondition(greenhousemetav1alpha1.FalseCondition(
			greenhousev1alpha1.HelmRepositoryReadyCondition,
			greenhousemetav1alpha1.ConditionReason(readyCondition.Reason),
			readyCondition.Message))
	}
}
