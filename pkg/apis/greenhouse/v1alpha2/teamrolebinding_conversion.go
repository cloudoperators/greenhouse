// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha2

import (
	"github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts this TeamRoleBinding to the Hub version (v1alpha1). See: https://book.kubebuilder.io/multiversion-tutorial/conversion
func (src *TeamRoleBinding) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha1.TeamRoleBinding)

	// Convert new ClusterSelector to the old selectors.
	dst.Spec.ClusterName = src.Spec.ClusterSelector.Name
	dst.Spec.ClusterSelector = src.Spec.ClusterSelector.LabelSelector

	// Rote conversion.

	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	// Spec
	dst.Spec.TeamRoleRef = src.Spec.TeamRoleRef
	dst.Spec.TeamRef = src.Spec.TeamRef
	dst.Spec.Namespaces = src.Spec.Namespaces

	// Status
	dst.Status.StatusConditions = convertStatusConditionsTo(src.Status.StatusConditions)
	dstPropagationStatus := make([]v1alpha1.PropagationStatus, 0, len(src.Status.PropagationStatus))
	for i, v := range src.Status.PropagationStatus {
		dstPropagationStatus[i] = v1alpha1.PropagationStatus{
			ClusterName: v.ClusterName,
			Condition:   convertConditionTo(v.Condition),
		}
	}
	dst.Status.PropagationStatus = dstPropagationStatus

	return nil
}

// ConvertFrom converts from the Hub version (v1alpha1) to this version.
func (dst *TeamRoleBinding) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha1.TeamRoleBinding)

	// Convert old selectors to the new ClusterSelector.
	dst.Spec.ClusterSelector = ClusterSelector{
		Name:          src.Spec.ClusterName,
		LabelSelector: src.Spec.ClusterSelector,
	}

	// Rote conversion.

	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	// Spec
	dst.Spec.TeamRoleRef = src.Spec.TeamRoleRef
	dst.Spec.TeamRef = src.Spec.TeamRef
	dst.Spec.Namespaces = src.Spec.Namespaces

	// Status
	dst.Status.StatusConditions = convertStatusConditionsFrom(src.Status.StatusConditions)
	dstPropagationStatus := make([]PropagationStatus, 0, len(src.Status.PropagationStatus))
	for i, v := range src.Status.PropagationStatus {
		dstPropagationStatus[i] = PropagationStatus{
			ClusterName: v.ClusterName,
			Condition:   convertConditionFrom(v.Condition),
		}
	}
	dst.Status.PropagationStatus = dstPropagationStatus

	return nil
}

func convertStatusConditionsTo(srcStatusConditions StatusConditions) v1alpha1.StatusConditions {
	dstConditions := make([]v1alpha1.Condition, 0, len(srcStatusConditions.Conditions))
	for i, v := range srcStatusConditions.Conditions {
		dstConditions[i] = convertConditionTo(v)
	}
	dstStatusConditions := v1alpha1.StatusConditions{
		Conditions: dstConditions,
	}
	return dstStatusConditions
}

func convertStatusConditionsFrom(srcStatusConditions v1alpha1.StatusConditions) StatusConditions {
	dstConditions := make([]Condition, 0, len(srcStatusConditions.Conditions))
	for i, v := range srcStatusConditions.Conditions {
		dstConditions[i] = convertConditionFrom(v)
	}
	dstStatusConditions := StatusConditions{
		Conditions: dstConditions,
	}
	return dstStatusConditions
}

func convertConditionTo(srcCondition Condition) v1alpha1.Condition {
	dstCondition := v1alpha1.Condition{
		Type:               v1alpha1.ConditionType(srcCondition.Type),
		Status:             srcCondition.Status,
		Reason:             v1alpha1.ConditionReason(srcCondition.Reason),
		LastTransitionTime: srcCondition.LastTransitionTime,
		Message:            srcCondition.Message,
	}
	return dstCondition
}

func convertConditionFrom(srcCondition v1alpha1.Condition) Condition {
	dstCondition := Condition{
		Type:               ConditionType(srcCondition.Type),
		Status:             srcCondition.Status,
		Reason:             ConditionReason(srcCondition.Reason),
		LastTransitionTime: srcCondition.LastTransitionTime,
		Message:            srcCondition.Message,
	}
	return dstCondition
}
