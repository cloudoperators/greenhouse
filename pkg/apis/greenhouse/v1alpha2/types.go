// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha2

import (
	"context"

	"github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterSelector specifies a selector for clusters by name or by label with the option to exclude specific clusters.
type ClusterSelector struct {
	// Name of a single Cluster to select.
	Name string `json:"name,omitempty"`
	// LabelSelector is a label query over a set of Clusters.
	LabelSelector metav1.LabelSelector `json:"labelSelector,omitempty"`
	// ExcludeList is a list of Cluster names to exclude from LabelSelector query.
	ExcludeList []string `json:"excludeList,omitempty"`
}

// ListClusters returns the list of Clusters that match the ClusterSelector's Name or LabelSelector with applied ExcludeList.
// If the Name or LabelSelector does not return any cluster, an empty ClusterList is returned without error.
func (cs *ClusterSelector) ListClusters(ctx context.Context, c client.Client, namespace string) (*v1alpha1.ClusterList, error) {
	if cs.Name != "" {
		cluster := new(v1alpha1.Cluster)
		err := c.Get(ctx, types.NamespacedName{Name: cs.Name, Namespace: namespace}, cluster)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return &v1alpha1.ClusterList{}, nil
			}
			return nil, err
		}
		return &v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil
	}

	labelSelector, err := metav1.LabelSelectorAsSelector(&cs.LabelSelector)
	if err != nil {
		return nil, err
	}
	var clusters = new(v1alpha1.ClusterList)
	if err := c.List(ctx, clusters, client.InNamespace(namespace), client.MatchingLabelsSelector{Selector: labelSelector}); err != nil {
		return nil, err
	}
	if len(clusters.Items) == 0 || len(cs.ExcludeList) == 0 {
		return clusters, nil
	}

	filteredClusters := make([]v1alpha1.Cluster, 0)
	for _, cluster := range clusters.Items {
		shouldExclude := false
		for _, excludedClusterName := range cs.ExcludeList {
			if cluster.Name == excludedClusterName {
				shouldExclude = true
				break
			}
		}
		if shouldExclude {
			continue
		}
		filteredClusters = append(filteredClusters, cluster)
	}
	clusters.Items = filteredClusters

	return clusters, nil
}
