// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterSelector specifies a selector for clusters by name or by label
type ClusterSelector struct {
	// Name of a single Cluster to select.
	Name string `json:"clusterName,omitempty"`
	// LabelSelector is a label query over a set of Clusters.
	LabelSelector metav1.LabelSelector `json:"labelSelector,omitempty"`
}
