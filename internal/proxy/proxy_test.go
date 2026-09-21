// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package proxy

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"testing"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/test"
)

func newTestProxyManager() *PmManager {
	return &PmManager{
		logger: logr.Discard(),
		store:  NewStore(),
	}
}

func TestRewrite(t *testing.T) {
	proxyURL, err := url.Parse("https://api.test-api-server.com/api/v1/namespaces/kube-monitoring/services/test-service:8080")
	if err != nil {
		t.Fatal("failed to parse proxy URL")
	}

	tests := []struct {
		name                            string
		url                             string
		expectedUpstreamServiceRouteURL string
		contextVal                      any
	}{
		{
			name:                            "valid host with path",
			url:                             "https://cluster--1234567.organisation.basedomain/dashboard",
			expectedUpstreamServiceRouteURL: "https://api.test-api-server.com/api/v1/namespaces/kube-monitoring/services/test-service:8080/dashboard",
			contextVal:                      "cluster",
		},
		{
			name:                            "valid host with deeper path",
			url:                             "https://cluster--1234567.organisation.basedomain/api/resource",
			expectedUpstreamServiceRouteURL: "https://api.test-api-server.com/api/v1/namespaces/kube-monitoring/services/test-service:8080/api/resource",
			contextVal:                      "cluster",
		},
		{
			name:                            "valid host with already prefixed path",
			url:                             "https://cluster--1234567.organisation.basedomain/api/v1/namespaces/kube-monitoring/services/test-service:8080/existing-path",
			expectedUpstreamServiceRouteURL: "https://api.test-api-server.com/api/v1/namespaces/kube-monitoring/services/test-service:8080/existing-path",
			contextVal:                      "cluster",
		},
		{
			name:                            "unknown cluster request",
			url:                             "https://unknown-cluster.organisation.basedomain/dashboard",
			expectedUpstreamServiceRouteURL: "https://unknown-cluster.organisation.basedomain/dashboard",
			contextVal:                      nil,
		},
		{
			name:                            "invalid host format",
			url:                             "https://something.organisation.basedomain/abcd",
			expectedUpstreamServiceRouteURL: "https://something.organisation.basedomain/abcd",
			contextVal:                      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputURL, err := url.Parse(tt.url)
			if err != nil {
				t.Fatal("failed to parse URL")
			}

			pm := newTestProxyManager()
			pm.store.SetRoutes("cluster", map[string]route{
				inputURL.Scheme + "://" + inputURL.Host: {
					url:         proxyURL,
					namespace:   "kube-monitoring",
					serviceName: "test-service",
				},
			})

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

			if req.Out.URL.String() != tt.expectedUpstreamServiceRouteURL {
				t.Errorf("expected URL %s, got %s", tt.expectedUpstreamServiceRouteURL, req.Out.URL.String())
			}

			if req.Out.Context().Value(contextClusterKey{}) != tt.contextVal {
				t.Errorf("expected cluster %v in context, got %v", tt.contextVal, req.Out.Context().Value(contextClusterKey{}))
			}
		})
	}
}

func TestURLGenerationWithProtocols(t *testing.T) {
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
			protocol:        new("http"),
			expectedURLPath: "/api/v1/namespaces/namespace/services/test:8080/proxy",
		},
		{
			name:            "explicit_https_protocol",
			protocol:        new("https"),
			expectedURLPath: "/api/v1/namespaces/namespace/services/https:test:8080/proxy",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pm := newTestProxyManager()

			fakeClient := fake.NewClientBuilder().WithScheme(test.GreenhouseV1Alpha1Scheme()).WithObjects(
				&greenhousev1alpha1.Plugin{
					ObjectMeta: metav1.ObjectMeta{Name: "plugin1", Namespace: "namespace"},
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
								Type:      greenhousev1alpha1.ServiceTypeService,
							},
						},
					},
				},
			).Build()
			pm.cache = readerCache{Cache: nil, reader: fakeClient}
			pm.store.SetTransport("cluster-1", http.DefaultTransport, "https://apiserver.test")

			pm.rebuildClusterRoutes(context.Background(), "namespace", "cluster-1")

			r, ok := pm.store.Route("cluster-1", "https://cluster-1--1234567.org.basedomain")
			if !ok {
				t.Fatal("expected route to be added")
			}

			expectedURL := "https://apiserver.test" + tc.expectedURLPath
			if r.url.String() != expectedURL {
				t.Errorf("expected url %s, got %s", expectedURL, r.url.String())
			}
		})
	}
}

// readerCache adapts a client.Reader to the cache.Cache interface so
// rebuildClusterRoutes can list plugins from a fake client in tests. Only List
// is exercised.
type readerCache struct {
	cache.Cache
	reader client.Reader
}

func (r readerCache) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return r.reader.List(ctx, list, opts...)
}

func TestModifyResponse(t *testing.T) {
	tests := []struct {
		name               string
		locationHeader     string
		expectedLocation   string
		expectHeaderChange bool
	}{
		{
			name:               "Valid input with proxy paths",
			locationHeader:     "/api/v1/namespaces/kube-monitoring/services/test-service:8080/proxy/api/main.js",
			expectedLocation:   "/api/main.js",
			expectHeaderChange: true,
		},
		{
			name:               "Single proxy path",
			locationHeader:     "/api/v1/namespaces/kube-monitoring/services/test-service:8080/proxy/",
			expectedLocation:   "/",
			expectHeaderChange: true,
		},
		{
			name:               "No match in location header",
			locationHeader:     "/other/path/that/does/not/match",
			expectedLocation:   "/other/path/that/does/not/match",
			expectHeaderChange: false,
		},
		{
			name:               "Empty location header",
			locationHeader:     "",
			expectedLocation:   "",
			expectHeaderChange: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				Header: http.Header{
					"Location": []string{tt.locationHeader},
				},
				Request: &http.Request{},
			}

			pm := newTestProxyManager()
			err := pm.modifyResponse(resp)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			location := resp.Header.Get("Location")
			if location != tt.expectedLocation {
				t.Errorf("expected location %s, got %s", tt.expectedLocation, location)
			}

			headerChanged := location != tt.locationHeader
			if headerChanged != tt.expectHeaderChange {
				t.Errorf("expected header change: %v, got: %v", tt.expectHeaderChange, headerChanged)
			}
		})
	}
}
