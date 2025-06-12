// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

// WithAccessMode sets the ClusterAccessMode on a Cluster
func WithAccessMode(mode greenhousev1alpha1.ClusterAccessMode) func(*greenhousev1alpha1.Cluster) {
	return func(c *greenhousev1alpha1.Cluster) {
		c.Spec.AccessMode = mode
	}
}

// WithLabel sets the label on a Cluster
func WithLabel(key, value string) func(*greenhousev1alpha1.Cluster) {
	return func(c *greenhousev1alpha1.Cluster) {
		if c.Labels == nil {
			c.Labels = make(map[string]string, 1)
		}
		c.Labels[key] = value
	}
}

// WithClusterAnnotations sets metadata annotations on a Cluster
func WithClusterAnnotations(annotations map[string]string) func(*greenhousev1alpha1.Cluster) {
	return func(c *greenhousev1alpha1.Cluster) {
		c.SetAnnotations(annotations)
	}
}

// WithKubeConfig sets the kubeconfig of a Cluster
func WithMaxTokenValidity(maxTokenValidity int32) func(*greenhousev1alpha1.Cluster) {
	return func(c *greenhousev1alpha1.Cluster) {
		c.Spec.KubeConfig.MaxTokenValidity = maxTokenValidity
	}
}

// NewCluster returns a greenhousev1alpha1.Cluster object. Opts can be used to set the desired state of the Cluster.
func NewCluster(name, namespace string, opts ...func(*greenhousev1alpha1.Cluster)) *greenhousev1alpha1.Cluster {
	cluster := &greenhousev1alpha1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: greenhousev1alpha1.ClusterSpec{
			AccessMode: greenhousev1alpha1.ClusterAccessModeDirect,
		},
	}

	for _, o := range opts {
		o(cluster)
	}
	return cluster
}
