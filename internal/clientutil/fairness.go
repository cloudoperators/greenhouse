// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package clientutil

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	flowcontrolapi "k8s.io/api/flowcontrol/v1beta2"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport"
)

const pingPath = "/livez/ping"

// isPriorityAndFairnessEnabled - returns true if the server has the Priority and Fairness check
// filter enabled.
func isPriorityAndFairnessEnabled(ctx context.Context, config *rest.Config) (bool, error) {
	transportConfig, err := config.TransportConfig()
	if err != nil {
		return false, fmt.Errorf("building transport config: %w", err)
	}
	trippy, err := transport.New(transportConfig)
	if err != nil {
		return false, fmt.Errorf("building round tripper: %w", err)
	}

	// Build the base api-server URL from the provided REST client config.
	serverURL, err := apiServerURL(config)
	if err != nil {
		return false, fmt.Errorf("building server URL: %w", err)
	}

	// Use the ping endpoint, because it is fast.
	// The endpoint for old clusters 1.23 range is behind a feature gate so even a 404 will still have the flow control headers.
	serverURL.Path = pingPath

	// Build HEAD request with an empty body.
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, serverURL.String(), http.NoBody)
	if err != nil {
		return false, fmt.Errorf("error building %s request: %w", pingPath, err)
	}

	if config.UserAgent != "" {
		req.Header.Set("User-Agent", config.UserAgent)
	}
	// we don't need to set "Accept" header, because we don't need to worry about the response body, only the headers matter.

	resp, err := trippy.RoundTrip(req)
	if err != nil {
		return false, fmt.Errorf("error making %s request: %w", pingPath, err)
	}
	// HEAD request should not have a body in general, but check nevertheless
	if resp.Body != nil {
		// close to free up resources
		err := resp.Body.Close()
		if err != nil {
			return false, fmt.Errorf("closing response body: %w", err)
		}
	}

	// If the response has one of the flow control headers,
	// that means the server has the Priority and Fairness filter enabled.
	// There are always two headers
	// x-kubernetes-pf-prioritylevel-uid
	// x-kubernetes-pf-flowschema-uid
	// we only need to check one of them

	// key = flowcontrolapi.ResponseHeaderMatchedPriorityLevelConfigurationUID
	key := flowcontrolapi.ResponseHeaderMatchedFlowSchemaUID
	if value := resp.Header.Get(key); value != "" {
		// the value is a UUID, nothing to do with it.
		return true, nil
	}
	return false, nil
}

// apiServerURL - returns the base URL for the cluster based on rest config.
// Host and Version are required. GroupVersion is ignored.
// Based on `defaultServerUrlFor` from k8s.io/client-go@v0.23.2/rest/url_utils.go
func apiServerURL(config *rest.Config) (*url.URL, error) {
	// config.Insecure is taken to mean "I want HTTPS but ignore check of certs against a CA."
	hasCA := config.CAFile != "" || len(config.CAData) != 0
	hasCert := config.CertFile != "" || len(config.CertData) != 0
	defaultTLS := hasCA || hasCert || config.Insecure
	host := config.Host
	if host == "" {
		host = "localhost"
	}

	hostURL, _, err := rest.DefaultServerURL(host, config.APIPath, schema.GroupVersion{}, defaultTLS)
	return hostURL, err
}
