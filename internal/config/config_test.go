// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"reflect"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/cloudoperators/greenhouse/internal/config"
)

func TestOrganizationCleanupConfig(t *testing.T) {
	const fallback = 42 * time.Second

	tests := []struct {
		name         string
		configMap    *corev1.ConfigMap
		wantGroups   []string
		wantInterval time.Duration
	}{
		{
			name:         "falls back when the ConfigMap is missing",
			wantInterval: fallback,
		},
		{
			name:         "falls back when the keys are not set",
			configMap:    newConfigMap(map[string]string{"controller.plugin.maxconcurrentreconciles": "3"}),
			wantInterval: fallback,
		},
		{
			name: "reads the groups and the interval",
			configMap: newConfigMap(map[string]string{
				"organizationcleanup.groups":          "a.example.com,b.example.com",
				"organizationcleanup.requeueinterval": "90s",
			}),
			wantGroups:   []string{"a.example.com", "b.example.com"},
			wantInterval: 90 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := fake.NewClientBuilder()
			if tt.configMap != nil {
				builder = builder.WithObjects(tt.configMap)
			}

			cfg, err := config.NewManagerConfig(t.Context(), builder.Build(), "greenhouse-cm", "greenhouse")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got := cfg.GetOrganizationCleanupGroups(); !reflect.DeepEqual(got, tt.wantGroups) {
				t.Errorf("groups = %v, want %v", got, tt.wantGroups)
			}
			if got := cfg.GetOrganizationCleanupRequeueInterval(fallback); got != tt.wantInterval {
				t.Errorf("interval = %v, want %v", got, tt.wantInterval)
			}
		})
	}
}

func newConfigMap(data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "greenhouse-cm", Namespace: "greenhouse"},
		Data:       data,
	}
}
