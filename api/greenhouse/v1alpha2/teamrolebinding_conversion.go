// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha2

import (
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	"github.com/cloudoperators/greenhouse/api/greenhouse/v1alpha1"
)

// ConvertTo converts this TeamRoleBinding to the Hub version (v1alpha1). See: https://book.kubebuilder.io/multiversion-tutorial/conversion
func (trb *TeamRoleBinding) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha1.TeamRoleBinding) //nolint:errcheck

	// Convert new ClusterSelector to the old selectors.
	dst.Spec.ClusterName = trb.Spec.ClusterSelector.Name
	dst.Spec.ClusterSelector = trb.Spec.ClusterSelector.LabelSelector

	// Rote conversion.

	// ObjectMeta
	dst.ObjectMeta = trb.ObjectMeta

	// Spec
	dst.Spec.TeamRoleRef = trb.Spec.TeamRoleRef
	dst.Spec.TeamRef = trb.Spec.TeamRef
	dst.Spec.Namespaces = trb.Spec.Namespaces

	// Status
	dst.Status.StatusConditions = trb.Status.StatusConditions

	dstPropagationStatus := make([]v1alpha1.PropagationStatus, 0, len(trb.Status.PropagationStatus))
	for i, v := range trb.Status.PropagationStatus {
		dstPropagationStatus[i] = v1alpha1.PropagationStatus{
			ClusterName: v.ClusterName,
			Condition:   v.Condition,
		}
	}
	dst.Status.PropagationStatus = dstPropagationStatus

	return nil
}

// ConvertFrom converts from the Hub version (v1alpha1) to this version.
func (trb *TeamRoleBinding) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha1.TeamRoleBinding) //nolint:errcheck

	// Convert old selectors to the new ClusterSelector.
	trb.Spec.ClusterSelector = ClusterSelector{
		Name:          src.Spec.ClusterName,
		LabelSelector: src.Spec.ClusterSelector,
	}

	// Rote conversion.

	// ObjectMeta
	trb.ObjectMeta = src.ObjectMeta

	// Spec
	trb.Spec.TeamRoleRef = src.Spec.TeamRoleRef
	trb.Spec.TeamRef = src.Spec.TeamRef
	trb.Spec.Namespaces = src.Spec.Namespaces

	// Status
	trb.Status.StatusConditions = src.Status.StatusConditions
	dstPropagationStatus := make([]PropagationStatus, 0, len(src.Status.PropagationStatus))
	for i, v := range src.Status.PropagationStatus {
		dstPropagationStatus[i] = PropagationStatus{
			ClusterName: v.ClusterName,
			Condition:   v.Condition,
		}
	}
	trb.Status.PropagationStatus = dstPropagationStatus

	return nil
}
