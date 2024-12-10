// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package scim

import (
	"context"
	"fmt"

	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Basic_Auth_Client(t *testing.T) {
	testTable := []struct {
		name      string
		username  string
		password  string
		withError bool
	}{
		{
			name:     "it should successfully create a basic auth client",
			username: "some-username",
			password: "some-password",
		},
		{
			name:      "it should fail to create a basic auth client, when no username is provided",
			password:  "some-password",
			withError: true,
		},
		{
			name:      "it should fail to create a basic auth client, when no password is provided",
			username:  "some-username",
			withError: true,
		},
		{
			name:      "it should fail to create a basic auth client, when no username and password is provided",
			withError: true,
		},
	}
	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			server, _ := setup()
			_, err := NewSCIMClient(&Config{
				URL:      server.URL + baseURLPath,
				AuthType: Basic,
				BasicAuth: &BasicAuthConfig{
					Username: test.username,
					Password: test.password,
				},
			})
			if test.withError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			t.Cleanup(func() {
				// tear down test server
				server.Close()
			})
		})
	}
}

func TestClient_GroupExists(t *testing.T) {
	// Create a fake client to mock API calls.
	responseMap := make(map[string]mockResponse)
	ctx := context.Background()
	server, mux := setup()

	mux.HandleFunc("/Groups", func(w http.ResponseWriter, r *http.Request) {
		mockResp, ok := responseMap[r.URL.Query().Get("filter")]
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Add("Content-Type", "application/scim+json")
		w.WriteHeader(mockResp.statusCode)
		_, err := fmt.Fprint(w, mockResp.body)
		assert.NoError(t, err)
	})
	scimClient, err := NewSCIMClient(&Config{
		URL:      server.URL + baseURLPath,
		AuthType: Basic,
		BasicAuth: &BasicAuthConfig{
			Username: "some-username",
			Password: "some-password",
		},
	})
	assert.NoError(t, err)

	testTable := []struct {
		name                 string
		queryOptions         *QueryOptions
		mockResponse         mockResponse
		expectedExists       bool
		expectedTotalResults int
		withError            bool
	}{
		{
			name: "it should successfully check if group exists",
			queryOptions: &QueryOptions{
				Filter:             GroupFilterByDisplayName("SOME_IDP_GROUP_NAME"),
				ExcludedAttributes: SetAttributes(AttrMembers),
			},
			mockResponse:         existingGroupResponseBodyMockFn(),
			expectedTotalResults: 1,
			expectedExists:       true,
		},
		{
			name: "it should successfully check if group does not exist",
			queryOptions: &QueryOptions{
				Filter:             GroupFilterByDisplayName("non-existing-group"),
				ExcludedAttributes: SetAttributes(AttrMembers),
			},
			mockResponse:   emptyResponseBodyMockFn(),
			expectedExists: false,
		},
		{
			name: "it should error out when invalid request is made",
			queryOptions: &QueryOptions{
				Filter:  GroupFilterByDisplayName("error-group"),
				StartID: "1",
			},
			mockResponse: errorResponseBodyMockFn(),
			withError:    true,
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			// Update the response map for this test case
			responseMap[test.queryOptions.Filter.String()] = test.mockResponse
			exists, err := scimClient.GroupExists(ctx, test.queryOptions)
			if test.withError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedExists, exists)
			}
			group, err := scimClient.GetGroups(ctx, test.queryOptions)
			if test.withError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedTotalResults, group.TotalResults)
			}
		})
	}
	t.Cleanup(func() {
		// tear down test server
		server.Close()
	})
}

func TestClient_GetUsers(t *testing.T) {
	// Create a mock response map and test server
	responseMap := make(map[string]mockResponse)
	ctx := context.Background()
	server, mux := setup()

	mux.HandleFunc("/Users", func(w http.ResponseWriter, r *http.Request) {
		mockResp, ok := responseMap[r.URL.Query().Get("filter")]
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Add("Content-Type", "application/scim+json")
		w.WriteHeader(mockResp.statusCode)
		_, err := fmt.Fprint(w, mockResp.body)
		assert.NoError(t, err)
	})

	// Create the SCIM client
	scimClient, err := NewSCIMClient(&Config{
		URL:      server.URL + baseURLPath,
		AuthType: Basic,
		BasicAuth: &BasicAuthConfig{
			Username: "some-username",
			Password: "some-password",
		},
	})
	assert.NoError(t, err)

	// Define test cases
	testTable := []struct {
		name          string
		queryOptions  *QueryOptions
		mockResponse  mockResponse
		expectedUsers int
		withError     bool
	}{
		{
			name: "it should successfully fetch users",
			queryOptions: &QueryOptions{
				Filter:     UserFilterByGroupDisplayName("SOME_IDP_GROUP_NAME"),
				Attributes: SetAttributes(AttrName, AttrEmails, AttrDisplayName, AttrActive),
			},
			mockResponse:  userResponseBodyMockFn(),
			expectedUsers: 2,
		},
		{
			name: "it should return an empty result when no users exist",
			queryOptions: &QueryOptions{
				Filter:     UserFilterByGroupDisplayName("non-existing-group"),
				Attributes: SetAttributes(AttrName, AttrEmails, AttrDisplayName, AttrActive),
			},
			mockResponse:  emptyResponseBodyMockFn(),
			expectedUsers: 0,
		},
		{
			name: "it should handle an error response",
			queryOptions: &QueryOptions{
				Filter:  UserFilterByGroupDisplayName("some-group"),
				StartID: "invalid",
			},
			mockResponse: errorResponseBodyMockFn(),
			withError:    true,
		},
	}

	// Run the test cases
	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			// Set up the mock response for this test case
			responseMap[test.queryOptions.Filter.String()] = test.mockResponse

			// Make the GetUsers call
			users, err := scimClient.GetUsers(ctx, test.queryOptions)
			if test.withError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedUsers, users.TotalResults)
			}
		})
	}
	t.Cleanup(func() {
		server.Close()
	})
}

func TestClient_GetPaginatedUsers(t *testing.T) {
	// Create a mock response map and test server
	responseMap := make(map[string]mockResponse)
	ctx := context.Background()
	server, mux := setup()

	// This counter will track how many times "/Users" has been called.
	callCount := 0

	mux.HandleFunc("/Users", func(w http.ResponseWriter, r *http.Request) {
		callCount++

		startID := r.URL.Query().Get("startId") // use startId to distinguish pages

		mockResp, ok := responseMap[startID]
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Add("Content-Type", "application/scim+json")
		w.WriteHeader(mockResp.statusCode)
		_, err := fmt.Fprint(w, mockResp.body)
		assert.NoError(t, err)
	})

	// Create the SCIM client
	scimClient, err := NewSCIMClient(&Config{
		URL:      server.URL + baseURLPath,
		AuthType: Basic,
		BasicAuth: &BasicAuthConfig{
			Username: "some-username",
			Password: "some-password",
		},
	})
	assert.NoError(t, err)

	// simulate two pages:
	// First page: returns 2 users and a nextId
	// Second page: returns the last user, no nextId

	firstPageResponse := firstUserPaginatedResponseFn()
	secondPageResponse := secondUserPaginatedResponseFn()

	// The key "" represents the first request (no startId),
	// "second-page" represents the second request.
	responseMap["initial"] = firstPageResponse
	responseMap["second-page"] = secondPageResponse

	// Call GetPaginatedUsers
	users, err := scimClient.GetPaginatedUsers(ctx, &QueryOptions{
		Filter:  UserFilterByGroupDisplayName("some-group"),
		StartID: "initial",
	})

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 3, len(users))
	// Ensure we got the combined results from both pages
	assert.Equal(t, "user1", users[0].ID)
	assert.Equal(t, "user2", users[1].ID)
	assert.Equal(t, "user3", users[2].ID)

	// Assert that the "/Users" endpoint was hit twice:
	// once for the first page and once for the second page.
	assert.Equal(t, 2, callCount)

	t.Cleanup(func() {
		server.Close()
	})
}
