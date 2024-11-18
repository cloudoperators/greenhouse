// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// HelmChartReference references a Helm Chart in a chart repository.
type HelmChartReference struct {
	// Name of the HelmChart chart.
	Name string `json:"name"`
	// Repository of the HelmChart chart.
	Repository string `json:"repository"`
	// Version of the HelmChart chart.
	Version string `json:"version"`
}

// String returns the printable HelmChartReference.
func (h *HelmChartReference) String() string {
	return fmt.Sprintf("%s/%s:%s", h.Repository, h.Name, h.Version)
}

// ValueFromSource is a valid source for a value.
type ValueFromSource struct {
	// Secret references the secret containing the value.
	Secret *SecretKeyReference `json:"secret,omitempty"`
}

// SecretKeyReference specifies the secret and key containing the value.
type SecretKeyReference struct {
	// Name of the secret in the same namespace.
	Name string `json:"name"`
	// Key in the secret to select the value from.
	Key string `json:"key"`
}

// UIApplicationReference references the UI pluginDefinition to use.
type UIApplicationReference struct {
	// URL specifies the url to a built javascript asset.
	// By default, assets are loaded from the Juno asset server using the provided name and version.
	URL string `json:"url,omitempty"`

	// Name of the UI application.
	Name string `json:"name"`

	// Version of the frontend application.
	Version string `json:"version"`
}

type ClusterSelector struct {
	Name          string               `json:"clusterName,omitempty"`
	LabelSelector metav1.LabelSelector `json:"labelSelector,omitempty"`
	ExcludeList   []string             `json:"excludeList,omitempty"`
}

// ListClusters returns the list of Clusters that match the ClusterSelector's Name or LabelSelector with applied ExcludeList.
// If the Name or LabelSelector does not return any cluster, an empty ClusterList is returned without error.
func (cs *ClusterSelector) ListClusters(ctx context.Context, c client.Client, namespace string) (*ClusterList, error) {
	if cs.Name != "" {
		cluster := new(Cluster)
		err := c.Get(ctx, types.NamespacedName{Name: cs.Name, Namespace: namespace}, cluster)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return &ClusterList{}, nil
			}
			return nil, err
		}
		return &ClusterList{Items: []Cluster{*cluster}}, nil
	}

	labelSelector, err := metav1.LabelSelectorAsSelector(&cs.LabelSelector)
	if err != nil {
		return nil, err
	}
	var clusters = new(ClusterList)
	if err := c.List(ctx, clusters, client.InNamespace(namespace), client.MatchingLabelsSelector{Selector: labelSelector}); err != nil {
		return nil, err
	}
	if len(clusters.Items) == 0 || len(cs.ExcludeList) == 0 {
		return clusters, nil
	}

	filteredClusters := make([]Cluster, 0)
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
