// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package scim

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	greenhousesapv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

type ScimClientTests struct{}

var firstTeamMemberRef = "https://some-user-location"

func TestScim(t *testing.T) {
	scTest := ScimClientTests{}
	t.Run("NewScimClient", scTest.TestNewScimClient)
	t.Run("GetMembers:Non existing group", scTest.TestGetTeamMembersFromNonExistingGroup)
	t.Run("GetMembers:Existing group", scTest.TestGetTeamMembersFromGroup)
	t.Run("GetMembers:Error from upstream", scTest.TestGetTeamMembersErrorResponse)
	t.Run("GetMembers:Malformed response", scTest.TestGetTeamMembersFromMalformedGroupResponse)
	t.Run("GetUser:Valid user", scTest.TestGetUser)
	t.Run("GetUsers:Error from upstream", scTest.TestGetUserErrorResponse)
	t.Run("GetUsers:Malformed response", scTest.TestGetUserMalformedUserResponse)
	t.Run("GetUsers:2 valid users, 1 malformed, 1 inactive", scTest.TestGetUsers)
}

func (sc *ScimClientTests) TestNewScimClient(t *testing.T) {
	scimConfig := Config{"some-url.com", Basic, &BasicAuthConfig{"user", "pw"}}
	_, err := NewScimClient(scimConfig)
	assert.NilError(t, err, "Should create client with basic auth credentials from env vars")
}

func (sc *ScimClientTests) TestGetTeamMembersFromNonExistingGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/Groups", "Should correctly call groups path")
		assert.Equal(t, r.URL.RawQuery, "filter=displayName+eq+%22NON_EXISTING_IDP_GROUP_NAME%22", "Should correctly call filter parameters")

		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "application/scim+json")
		_, err := w.Write([]byte(EmptyGroupResponseBodyMock))
		if err != nil {
			log.Printf("error creating mock server: %s", err)
		}
	}))
	defer server.Close()
	scimConfig := Config{server.URL, Basic, &BasicAuthConfig{"user", "pw"}}
	scimClient, err := NewScimClient(scimConfig)
	assert.NilError(t, err)

	teamMembers, err := scimClient.GetTeamMembers("NON_EXISTING_IDP_GROUP_NAME")

	assert.Error(t, err, "no mapped group found for NON_EXISTING_IDP_GROUP_NAME", "Should correctly error false group name")
	assert.Equal(t, len(teamMembers), 0, "Shouldn't return teamMembers from not existing group")
}

func (sc *ScimClientTests) TestGetTeamMembersFromGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/Groups", "Should correctly call groups path")
		assert.Equal(t, r.URL.RawQuery, "filter=displayName+eq+%22SOME_IDP_GROUP_NAME%22", "Should correctly call filter parameters")

		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "application/scim+json")
		_, err := w.Write([]byte(GroupResponseBodyWithMembersMock))
		if err != nil {
			log.Printf("error creating mock server: %s", err)
		}
	}))
	defer server.Close()
	scimConfig := Config{server.URL, Basic, &BasicAuthConfig{"user", "pw"}}
	scimClient, err := NewScimClient(scimConfig)
	assert.NilError(t, err)

	teamMembers, err := scimClient.GetTeamMembers("SOME_IDP_GROUP_NAME")
	assert.NilError(t, err, "Should not error on existing group with team members")
	assert.Equal(t, len(teamMembers), 3, "Should return all existing members")
	assert.Equal(t, teamMembers[0].Ref, firstTeamMemberRef, "Should match user reference")
}

func (sc *ScimClientTests) TestGetTeamMembersErrorResponse(t *testing.T) {
	type testCase struct {
		statusCode int
	}

	for _, testCase := range []testCase{{500}, {404}, {403}} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, r.URL.Path, "/Groups", "Should correctly call groups path")
			assert.Equal(t, r.URL.RawQuery, "filter=displayName+eq+%22SOME_IDP_GROUP_NAME%22", "Should correctly call filter parameters")

			w.WriteHeader(testCase.statusCode)
			w.Header().Add("Content-Type", "application/scim+json")
			_, err := w.Write([]byte(`{"error":"internal"}`))
			if err != nil {
				log.Printf("error creating mock server: %s", err)
			}
		}))
		defer server.Close()
		scimConfig := Config{server.URL, Basic, &BasicAuthConfig{"user", "pw"}}
		scimClient, err := NewScimClient(scimConfig)
		assert.NilError(t, err)

		teamMembers, err := scimClient.GetTeamMembers("SOME_IDP_GROUP_NAME")
		assert.ErrorContains(t, err, "could not retrieve TeamMembers from", "Should correctly error on non 200 Status codes")
		assert.Equal(t, len(teamMembers), 0, "Shouldn't return teamMembers if request errored")
	}
}

func (sc *ScimClientTests) TestGetTeamMembersFromMalformedGroupResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/Groups", "Should correctly call groups path")
		assert.Equal(t, r.URL.RawQuery, "filter=displayName+eq+%22SOME_IDP_GROUP_NAME%22", "Should correctly call filter parameters")

		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "application/scim+json")
		_, err := w.Write([]byte(MalformedGroupResponseBodyMock))
		if err != nil {
			log.Printf("error creating mock server: %s", err)
		}
	}))
	defer server.Close()
	scimConfig := Config{server.URL, Basic, &BasicAuthConfig{"user", "pw"}}
	scimClient, err := NewScimClient(scimConfig)
	assert.NilError(t, err)

	teamMembers, err := scimClient.GetTeamMembers("SOME_IDP_GROUP_NAME")
	assert.ErrorContains(t, err, "could not extract members from groupResponseBody", "Should correctly error on malformed groupResponseBody")
	assert.Equal(t, len(teamMembers), 0, "Shouldn't return teamMemberson malformed groupResponseBody")
}

func (sc *ScimClientTests) TestGetUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "application/scim+json")
		_, err := w.Write([]byte(UserResponseBodyMock1))
		if err != nil {
			log.Printf("error creating mock server: %s", err)
		}
	}))
	defer server.Close()
	scimConfig := Config{server.URL, Basic, &BasicAuthConfig{"user", "pw"}}
	scimClient, err := NewScimClient(scimConfig)
	member := Member{server.URL}
	assert.NilError(t, err)
	user, err := scimClient.getUser(member)
	assert.NilError(t, err, "There should be no error getting a user of a valid member")
	assert.Equal(t, *user, greenhousesapv1alpha1.User{ID: "I12345", FirstName: "John", LastName: "Doe", Email: "john.doe@example.com"})
}

func (sc *ScimClientTests) TestGetUserErrorResponse(t *testing.T) {
	type testCase struct {
		statusCode int
	}

	for _, testCase := range []testCase{{500}, {404}, {403}} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(testCase.statusCode)
			w.Header().Add("Content-Type", "application/scim+json")
			_, err := w.Write([]byte(`{"error":"some-error"}`))
			if err != nil {
				log.Printf("error creating mock server: %s", err)
			}
		}))
		defer server.Close()
		scimConfig := Config{server.URL, Basic, &BasicAuthConfig{"user", "pw"}}
		scimClient, err := NewScimClient(scimConfig)
		assert.NilError(t, err)
		member := Member{server.URL}
		user, err := scimClient.getUser(member)
		assert.ErrorContains(t, err, "could not retrieve TeamMember from", "Should correctly error on non 200 Status codes")
		assert.Equal(t, user, (*greenhousesapv1alpha1.User)(nil), "Shouldn't return teamMembers if request errored")
	}
}

func (sc *ScimClientTests) TestGetUserMalformedUserResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "application/scim+json")
		_, err := w.Write([]byte(MalFormedUserResponseBodyMock))
		if err != nil {
			log.Printf("error creating mock server: %s", err)
		}
	}))
	defer server.Close()
	scimConfig := Config{server.URL, Basic, &BasicAuthConfig{"user", "pw"}}
	scimClient, err := NewScimClient(scimConfig)
	assert.NilError(t, err)
	member := Member{server.URL}
	user, err := scimClient.getUser(member)
	assert.ErrorContains(t, err, "could not create User", "Should correctly error on malformed user")
	assert.Equal(t, user, (*greenhousesapv1alpha1.User)(nil), "Shouldn't return teamMembers if request errored")
}

func (sc *ScimClientTests) TestGetUsers(t *testing.T) {
	// valid user 1
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "application/scim+json")
		_, err := w.Write([]byte(UserResponseBodyMock1))
		if err != nil {
			log.Printf("error creating mock server: %s", err)
		}
	}))
	defer server1.Close()

	// malformed user
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "application/scim+json")
		_, err := w.Write([]byte(MalFormedUserResponseBodyMock))
		if err != nil {
			log.Printf("error creating mock server: %s", err)
		}
	}))
	defer server2.Close()

	// valid user 2
	server3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "application/scim+json")
		_, err := w.Write([]byte(UserResponseBodyMock2))
		if err != nil {
			log.Printf("error creating mock server: %s", err)
		}
	}))
	defer server3.Close()

	// inactive user
	server4 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "application/scim+json")
		_, err := w.Write([]byte(InactiveUserResponseBodyMock))
		if err != nil {
			log.Printf("error creating mock server: %s", err)
		}
	}))
	defer server4.Close()
	scimConfig := Config{"https://some-url", Basic, &BasicAuthConfig{"user", "pw"}}
	scimClient, err := NewScimClient(scimConfig)
	assert.NilError(t, err)
	members := []Member{{server1.URL}, {server2.URL}, {server3.URL}, {server4.URL}}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	users := scimClient.GetUsers(members)
	expectedUsers := []greenhousesapv1alpha1.User{{ID: "I12345", FirstName: "John", LastName: "Doe", Email: "john.doe@example.com"}, {ID: "I23456", FirstName: "Jane", LastName: "Doe", Email: "jane.doe@example.com"}}
	assert.NilError(t, err, "Should not error on getting users")
	assert.Equal(t, strings.Contains(buf.String(), "failed getting user: could not create User from memberResponseBody:"), true, "Should log error on malformed user")
	assert.Equal(t, strings.Contains(buf.String(), "failed getting user: user is not active:"), true, "Should log error on inactive user")
	assert.Equal(t, len(users), len(expectedUsers), "Should not error and not return malformed or inactive user")
}
