// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package clientutil

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetOwnerReference returns the OwnerReference if found else nil.
func GetOwnerReference(obj metav1.Object, kind string) *metav1.OwnerReference {
	for _, ref := range obj.GetOwnerReferences() {
		if ref.Kind == kind {
			return &ref
		}
	}
	return nil
}
