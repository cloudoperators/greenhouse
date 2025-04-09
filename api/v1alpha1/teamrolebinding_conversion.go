// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"github.com/cloudoperators/greenhouse/api/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts this TeamRoleBinding to the Hub version. See: https://book.kubebuilder.io/multiversion-tutorial/conversion
func (trb *TeamRoleBinding) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha2.TeamRoleBinding) //nolint:errcheck

	// Convert old selectors to the new ClusterSelector.
	dst.Spec.ClusterSelector = v1alpha2.ClusterSelector{
		Name:          trb.Spec.ClusterName,
		LabelSelector: trb.Spec.ClusterSelector,
	}

	// Rote conversion.

	// ObjectMeta
	dst.ObjectMeta = trb.ObjectMeta

	// Spec
	dst.Spec.TeamRoleRef = trb.Spec.TeamRoleRef
	dst.Spec.TeamRef = trb.Spec.TeamRef
	dst.Spec.Namespaces = trb.Spec.Namespaces

	// Status
	dst.Status.StatusConditions = trb.Status.StatusConditions

	dstPropagationStatus := make([]v1alpha2.PropagationStatus, 0, len(trb.Status.PropagationStatus))
	for i, v := range trb.Status.PropagationStatus {
		dstPropagationStatus[i] = v1alpha2.PropagationStatus{
			ClusterName: v.ClusterName,
			Condition:   v.Condition,
		}
	}
	dst.Status.PropagationStatus = dstPropagationStatus

	return nil
}

// ConvertFrom converts from the Hub version (v1alpha1) to this version.
func (trb *TeamRoleBinding) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha2.TeamRoleBinding) //nolint:errcheck

	// Convert the new ClusterSelector to the old selectors.
	trb.Spec.ClusterName = src.Spec.ClusterSelector.Name
	trb.Spec.ClusterSelector = src.Spec.ClusterSelector.LabelSelector

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
