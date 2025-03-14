// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// Package scim implements a very basic scim client with utility needed to extract users from IdP Groups
// A scim client is initialized via it's base url and authentication method.
// As for now only basic auth is implemented
package scim

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-logr/logr"
)

type scimClient struct {
	log        logr.Logger
	baseURL    *url.URL
	httpClient http.Client
}

type basicAuthTransport struct {
	Username string
	Password string
	Next     http.RoundTripper
}

type Config struct {
	URL         string
	AuthType    AuthType
	BasicAuth   *BasicAuthConfig
	BearerToken *BearerTokenConfig
}

type BasicAuthConfig struct {
	Username string
	Password string
}

type BearerTokenConfig struct {
	Token  string
	Header string
	Prefix string
}

type bearerTokenTransport struct {
	Token  string
	Header string
	Prefix string
	Next   http.RoundTripper
}

const (
	groupPath       = "Groups"
	userPath        = "Users"
	paginationEndID = "end"
)

func (t *basicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(t.Username, t.Password)
	return t.Next.RoundTrip(req)
}

func (t *bearerTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	header := "Authorization"
	prefix := "Bearer"

	if t.Header != "" {
		header = t.Header
	}
	if t.Prefix != "" {
		prefix = t.Prefix
	}

	req.Header.Set(header, fmt.Sprintf("%s %s", prefix, t.Token))
	return t.Next.RoundTrip(req)
}

type ISCIMClient interface {
	GetUsers(ctx context.Context, options *QueryOptions) ([]Resource, error)
	GetGroups(ctx context.Context, options *QueryOptions) ([]Resource, error)
}

// NewSCIMClient - creates a new SCIM client with an auth transport
func NewSCIMClient(logger logr.Logger, config *Config) (ISCIMClient, error) {
	var authTransport http.RoundTripper
	baseURL, err := url.Parse(config.URL)
	if err != nil {
		return nil, err
	}

	switch config.AuthType {
	case Basic:
		if config.BasicAuth == nil {
			return nil, errors.New("could not create http scim client, Basic Auth Config missing")
		}

		if strings.TrimSpace(config.BasicAuth.Username) == "" || strings.TrimSpace(config.BasicAuth.Password) == "" {
			return nil, errors.New("could not create SCIM Client, BasicAuthConfig missing username or password")
		}
		authTransport = &basicAuthTransport{
			Username: config.BasicAuth.Username,
			Password: config.BasicAuth.Password,
			Next:     http.DefaultTransport,
		}
	case BearerToken:
		if config.BearerToken == nil {
			return nil, errors.New("could not create http scim client, BearerToken Config missing")
		}
		if config.BearerToken.Token == "" {
			return nil, errors.New("could not create http scim client, BearerToken Token missing")
		}

		authTransport = &bearerTokenTransport{
			Token:  config.BearerToken.Token,
			Header: config.BearerToken.Header,
			Prefix: config.BearerToken.Prefix,
			Next:   http.DefaultTransport,
		}
	default:
		return nil, fmt.Errorf("unknown auth type: %s", config.AuthType)
	}

	return &scimClient{
		log:     logger,
		baseURL: baseURL,
		httpClient: http.Client{
			Transport: authTransport,
		},
	}, nil
}

func (c *scimClient) GetUsers(ctx context.Context, options *QueryOptions) ([]Resource, error) {
	return c.fetchAllResources(ctx, userPath, options)
}

// GetGroups - fetches all groups (optionally if StartID is provided then it does pagination)
func (c *scimClient) GetGroups(ctx context.Context, options *QueryOptions) ([]Resource, error) {
	return c.fetchAllResources(ctx, groupPath, options)
}

// fetchAllResources handles the logic for making a single or multiple requests depending on whether StartID is set.
func (c *scimClient) fetchAllResources(ctx context.Context, path string, options *QueryOptions) ([]Resource, error) {
	if options == nil {
		options = &QueryOptions{}
	}

	var allResources []Resource
	for {
		resources, nextID, err := c.fetchPage(ctx, path, options)
		if err != nil {
			return nil, err
		}

		allResources = append(allResources, resources...)

		// If StartID is not provided or nextID is empty or nextID is the end of pagination, then break
		if options.StartID == "" || nextID == "" || nextID == paginationEndID {
			break
		}

		options.StartID = nextID
	}

	return allResources, nil
}

// fetchPage fetches a single page of results for the given path and startID
func (c *scimClient) fetchPage(ctx context.Context, path string, options *QueryOptions) ([]Resource, string, error) {
	u := c.baseURL.JoinPath(path)
	if options != nil {
		u.RawQuery = options.toQuery()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return nil, "", err
	}
	resp, body, err := c.doRequest(req) //nolint:bodyclose
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}
	return body.Resources, body.NextID, nil
}

// doRequest performs the http request and returns the scim response body struct
func (c *scimClient) doRequest(req *http.Request) (*http.Response, *ResponseBody, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			c.log.Error(err, "error closing scim response body")
		}
	}(resp.Body)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	respBody := &ResponseBody{}
	err = json.Unmarshal(body, respBody)
	if err != nil {
		return nil, nil, err
	}
	return resp, respBody, nil
}
