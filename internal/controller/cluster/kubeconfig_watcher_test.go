// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
)

func TestClusterSecretToCluster(t *testing.T) {
	tests := []struct {
		name        string
		secretType  corev1.SecretType
		wantEnqueue bool
	}{
		{"kubeconfig secret enqueues", greenhouseapis.SecretTypeKubeConfig, true},
		{"oidc secret enqueues", greenhouseapis.SecretTypeOIDCConfig, true},
		{"other secret type does not enqueue", corev1.SecretTypeOpaque, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test-ns"},
				Type:       tc.secretType,
			}
			requests := clusterSecretToCluster(context.Background(), secret)
			if tc.wantEnqueue && len(requests) == 0 {
				t.Errorf("expected enqueue for secret type %q, got none", tc.secretType)
			}
			if !tc.wantEnqueue && len(requests) > 0 {
				t.Errorf("expected no enqueue for secret type %q, got %d requests", tc.secretType, len(requests))
			}
		})
	}
}
