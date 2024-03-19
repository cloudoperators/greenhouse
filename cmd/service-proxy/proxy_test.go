// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"testing"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
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
			url:         "https://name--namespace--cluster.organisation.basedomain/abcd",
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
				routes: map[string]*url.URL{
					inputURL.Scheme + "://" + inputURL.Host: proxyURL,
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

// TestReconcile tests the reconcile function of the proxy manager.
// It injects a client from  sigs.k8s.io/controller-runtime/pkg/client/fake into the proxy manager and
// sets up a cluster and a pluginconfig with an exposed service in the fake client.
// The test checks if the route is properly added to the cluster.
func TestReconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(greenhousev1alpha1.AddToScheme(scheme))

	pm := NewProxyManager()
	pm.client = fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&greenhousev1alpha1.PluginConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "plugin1",
				Namespace: "namespace",
			},
			Spec: greenhousev1alpha1.PluginConfigSpec{
				ClusterName: "cluster",
			},
			Status: greenhousev1alpha1.PluginConfigStatus{
				ExposedServices: map[string]greenhousev1alpha1.Service{
					"https://service--namespace--cluster.org.basedomain": {
						Namespace: "namespace",
						Name:      "test",
						Port:      8080,
					},
				},
			},
		},
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster",
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
	pm.clusters["cluster"] = clusterRoutes{
		routes: map[string]*url.URL{},
	}
	ctx := context.Background()
	_, err := pm.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: "cluster", Namespace: "namespace"}})

	if err != nil {
		t.Errorf("expected no error, got: %s", err)
	}

	targetURL, ok := pm.clusters["cluster"].routes["https://service--namespace--cluster.org.basedomain"]
	if !ok {
		t.Fatal("expected route to be added")
	}
	expectedURL := fmt.Sprintf("%s/api/v1/namespaces/%s/services/%s:%s:%s/proxy", "https://apiserver.test", "namespace", "http", "test", "8080")
	if targetURL.String() != expectedURL {
		t.Errorf("expected url %s, got %s", expectedURL, targetURL.String())
	}
}
