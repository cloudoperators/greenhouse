// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package changemanagement

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
)

// Auth holds the basic auth credentials for the change management endpoint.
type Auth struct {
	Username string
	Password string
}

// Client sends changes to a change management endpoint.
type Client struct {
	endpoint   string
	httpClient *http.Client
}

// New returns a Client that adds the auth to every request it sends to the endpoint.
func New(endpoint string, auth Auth) *Client {
	return &Client{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout:   10 * time.Second,
			Transport: &authTransport{auth: auth, next: http.DefaultTransport},
		},
	}
}

// Send POSTs the payload and fails unless the endpoint answers with 2xx.
func (c *Client) Send(ctx context.Context, payload []byte) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("change management endpoint responded with %s", response.Status)
	}
	return nil
}

// authTransport sets basic auth on every request.
type authTransport struct {
	auth Auth
	next http.RoundTripper
}

// RoundTrip works on a copy of the request, because a RoundTripper must not modify the original.
func (t *authTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	request = request.Clone(request.Context())
	if t.auth.Username != "" {
		request.SetBasicAuth(t.auth.Username, t.auth.Password)
	}
	return t.next.RoundTrip(request)
}

// Render renders the payload template with sprig without env and expandenv, like Helm.
// It returns nil when the template renders nothing, which skips the change, e.g. a failed upgrade.
func Render(text string, data any) ([]byte, error) {
	funcs := sprig.TxtFuncMap()
	delete(funcs, "env")
	delete(funcs, "expandenv")
	tmpl, err := template.New("payloadTemplate").Funcs(funcs).Parse(text)
	if err != nil {
		return nil, err
	}
	var payload bytes.Buffer
	if err := tmpl.Execute(&payload, data); err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(payload.Bytes())) == 0 {
		return nil, nil
	}
	if !json.Valid(payload.Bytes()) {
		return nil, errors.New("payloadTemplate did not render valid JSON")
	}
	return payload.Bytes(), nil
}
