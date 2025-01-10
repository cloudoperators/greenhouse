// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"testing"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

// TestRewrite tests the rewrite function of the proxy manager.
// The test uses a tabular driven test to test the different cases and sets up the internal proxy manager with a cluster and a routes.
// If checks if he url is properly rewritten and the request context contains the cluster name
// and a logger with the correct values.
func TestRewrite(t *testing.T) {
	proxyURL, err := url.Parse("https://apiserver/proxy/url")
	if err != nil {
		t.Fatal("failed to parse proxy url")
	}

	tests := []struct {
		name        string
		url         string
		expectedURL string
		contextVal  any
	}{
		{
			name:        "valid host",
			url:         "https://cluster--1234567.organisation.basedomain/abcd",
			expectedURL: "https://apiserver/proxy/url/abcd",
			contextVal:  "cluster",
		},
		{
			name:        "invalid host",
			url:         "https://something.organisation.basedomain/abcd",
			expectedURL: "https://something.organisation.basedomain/abcd",
			contextVal:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputURL, err := url.Parse(tt.url)
			if err != nil {
				t.Fatal("failed to parse url")
			}
			pm := NewProxyManager()
			pm.clusters["cluster"] = clusterRoutes{
				routes: map[string]route{
					inputURL.Scheme + "://" + inputURL.Host: {
						url:         proxyURL,
						namespace:   "namespace",
						serviceName: "test",
					},
				},
			}
			r, err := http.NewRequestWithContext(context.Background(), http.MethodGet, inputURL.String(), http.NoBody)
			if err != nil {
				t.Fatal("failed to create request")
				return
			}
			req := httputil.ProxyRequest{
				In:  r,
				Out: r.Clone(r.Context()),
			}
			pm.rewrite(&req)

			if _, err := logr.FromContext(req.Out.Context()); err != nil {
				t.Error("expected logger in outgoing request context")
			}
			if req.Out.URL.String() != tt.expectedURL {
				t.Errorf("expected url %s, got %s", tt.expectedURL, req.Out.URL.String())
			}
			if req.Out.Context().Value(contextClusterKey{}) != tt.contextVal {
				t.Errorf("expected cluster %s in context, got %s", "cluster", req.Out.Context().Value(contextClusterKey{}))
			}
		})
	}
}

func TestURLGenerationWithProtocols(t *testing.T) {
	// Test cases to cover different protocol scenarios
	tests := []struct {
		name            string
		protocol        *string
		expectedURLPath string
	}{
		{
			name:            "default_no_protocol",
			protocol:        nil,
			expectedURLPath: "/api/v1/namespaces/namespace/services/test:8080/proxy",
		},
		{
			name:            "explicit_http_protocol",
			protocol:        pointer("http"),
			expectedURLPath: "/api/v1/namespaces/namespace/services/test:8080/proxy",
		},
		{
			name:            "explicit_https_protocol",
			protocol:        pointer("https"),
			expectedURLPath: "/api/v1/namespaces/namespace/services/https:test:8080/proxy",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pm := NewProxyManager()
			pm.client = fake.NewClientBuilder().WithScheme(test.GreenhouseV1Alpha1Scheme()).WithObjects(
				&greenhousev1alpha1.Plugin{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "plugin1",
						Namespace: "namespace",
					},
					Spec: greenhousev1alpha1.PluginSpec{
						ClusterName: "cluster-1",
					},
					Status: greenhousev1alpha1.PluginStatus{
						ExposedServices: map[string]greenhousev1alpha1.Service{
							"https://cluster-1--1234567.org.basedomain": {
								Namespace: "namespace",
								Name:      "test",
								Port:      8080,
								Protocol:  tc.protocol,
							},
						},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cluster-1",
						Namespace: "namespace",
					},
					Type: "greenhouse.sap/kubeconfig",
					Data: map[string][]byte{
						greenhouseapis.GreenHouseKubeConfigKey: []byte(`
kind: Config
apiVersion: v1
clusters:
- name: cluster1
  cluster:
    server: https://apiserver.test
contexts:
- context:
    cluster: cluster1
    user: user1
  name: context1
current-context: context1
users:
- name: user1
`),
					},
				}).Build()

			ctx := context.Background()
			_, err := pm.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: "cluster-1", Namespace: "namespace"}})

			if err != nil {
				t.Errorf("expected no error, got: %s", err)
			}

			route, ok := pm.clusters["cluster-1"].routes["https://cluster-1--1234567.org.basedomain"]
			if !ok {
				t.Fatal("expected route to be added")
			}

			targetURL := route.url
			expectedURL := "https://apiserver.test" + tc.expectedURLPath
			if targetURL.String() != expectedURL {
				t.Errorf("expected url %s, got %s", expectedURL, targetURL.String())
			}
		})
	}
}

// helper function to create string pointer
func pointer(s string) *string {
	return &s
}
