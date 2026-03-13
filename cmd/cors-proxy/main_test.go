// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReverseProxyRewritesHostToTarget(t *testing.T) {
	var receivedHost string
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHost = r.Host
		w.WriteHeader(http.StatusOK)
	}))
	defer targetServer.Close()

	target, err := url.Parse(targetServer.URL)
	require.NoError(t, err)

	reverseProxy := newReverseProxy(target)

	proxy := httptest.NewServer(reverseProxy)
	defer proxy.Close()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, proxy.URL+"/some/path", http.NoBody)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, target.Host, receivedHost)
}
