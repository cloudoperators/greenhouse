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
)

type scimClient struct {
	baseURL    *url.URL
	httpClient http.Client
	paginator  Paginator
}

type paginator struct {
	client ISCIMClient
}

type basicAuthTransport struct {
	Username string
	Password string
	Next     http.RoundTripper
}

type Config struct {
	URL       string
	AuthType  AuthType
	BasicAuth *BasicAuthConfig
}

type BasicAuthConfig struct {
	Username string
	Password string
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

type Paginator interface {
	fetchAll(ctx context.Context, path string, options *QueryOptions) ([]Resource, error)
}

func newPaginator(c ISCIMClient) Paginator {
	return &paginator{
		client: c,
	}
}

type ISCIMClient interface {
	GetUsers(ctx context.Context, options *QueryOptions) (*ResponseBody, error)
	GetPaginatedUsers(ctx context.Context, options *QueryOptions) ([]Resource, error)
	GetGroups(ctx context.Context, options *QueryOptions) (*ResponseBody, error)
	GetPaginatedGroups(ctx context.Context, options *QueryOptions) ([]Resource, error)
	GroupExists(ctx context.Context, options *QueryOptions) (bool, error)
}

// NewSCIMClient - creates a new SCIM client with an auth transport
func NewSCIMClient(config *Config) (ISCIMClient, error) {
	var authTransport http.RoundTripper
	baseURL, err := url.Parse(config.URL)
	if err != nil {
		return nil, err
	}

	if config.AuthType == Basic {
		if config.BasicAuth == nil {
			return nil, errors.New("could not create http scimClient, BasicAuthConfig missing")
		}
		if strings.TrimSpace(config.BasicAuth.Username) == "" || strings.TrimSpace(config.BasicAuth.Password) == "" {
			return nil, errors.New("could not create SCIM Client, BasicAuthConfig missing username or password")
		}
		authTransport = &basicAuthTransport{
			Username: config.BasicAuth.Username,
			Password: config.BasicAuth.Password,
			Next:     http.DefaultTransport,
		}
	}

	c := &scimClient{
		baseURL: baseURL,
		httpClient: http.Client{
			Transport: authTransport,
		},
	}
	c.paginator = newPaginator(c)
	return c, nil
}

// GetPaginatedUsers - fetches all users using pagination
func (c *scimClient) GetPaginatedUsers(ctx context.Context, options *QueryOptions) ([]Resource, error) {
	return c.paginator.fetchAll(ctx, userPath, options)
}

// GetPaginatedGroups - fetches all groups using pagination
func (c *scimClient) GetPaginatedGroups(ctx context.Context, options *QueryOptions) ([]Resource, error) {
	return c.paginator.fetchAll(ctx, groupPath, options)
}

// fetchAll - fetches all user resources using pagination
func (p *paginator) fetchAll(ctx context.Context, path string, options *QueryOptions) ([]Resource, error) {
	var allResources []Resource

	// Ensure options is initialized
	if options == nil {
		options = &QueryOptions{}
	}

	for {
		// Fetch a single page of results
		resources, nextID, err := p.fetchPage(ctx, path, options)
		if err != nil {
			return nil, err
		}

		// Append the resources to the result set
		allResources = append(allResources, resources...)

		// Break if no more pages to fetch
		if nextID == paginationEndID || nextID == "" {
			break
		}

		// Set the NextID as StartID for the next request
		options.StartID = nextID
	}

	return allResources, nil
}

// fetchPage - fetches a single page of results and returns the resources and next ID
func (p *paginator) fetchPage(ctx context.Context, path string, options *QueryOptions) ([]Resource, string, error) {
	var responseBody *ResponseBody
	var err error
	switch path {
	case userPath:
		responseBody, err = p.client.GetUsers(ctx, options)
	case groupPath:
		responseBody, err = p.client.GetGroups(ctx, options)
	default:
		return nil, "", fmt.Errorf("unexpected path %s", path)
	}
	if err != nil {
		return nil, "", err
	}
	return responseBody.Resources, responseBody.NextID, err
}

// GetUsers - fetches users with optional query parameters
func (c *scimClient) GetUsers(ctx context.Context, options *QueryOptions) (*ResponseBody, error) {
	u := c.baseURL.JoinPath(userPath)
	if options != nil {
		u.RawQuery = options.toQuery()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return nil, err
	}
	resp, body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}
	return body, nil
}

// GroupExists - checks if a group exists
func (c *scimClient) GroupExists(ctx context.Context, options *QueryOptions) (bool, error) {
	groups, err := c.GetGroups(ctx, options)
	if err != nil {
		return false, err
	}
	if groups.TotalResults == 0 {
		return false, nil
	}
	return true, nil
}

// GetGroups - fetches groups with optional query parameters
func (c *scimClient) GetGroups(ctx context.Context, options *QueryOptions) (*ResponseBody, error) {
	u := c.baseURL.JoinPath("Groups")
	if options != nil {
		u.RawQuery = options.toQuery()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return nil, err
	}
	resp, body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}
	return body, nil
}

// doRequest performs the http request and returns the scim response body struct
func (c *scimClient) doRequest(req *http.Request) (*http.Response, *ResponseBody, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
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
