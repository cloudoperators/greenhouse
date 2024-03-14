// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// Package scim implemements a very basic scim client with utility needed to extract users from IdP Groups
// A scim client is initialized via it's base url and authentication method.
// As for now only basic auth is implemented
package scim

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

const (
	groupPathName             = "Groups"
	filterQueryKey            = "filter"
	filterQueryExpressionStub = "displayName eq \"%s\""
)

type Config struct {
	RawURL          string
	AuthType        AuthType
	BasicAuthConfig *BasicAuthConfig
}

type BasicAuthConfig struct {
	BasicAuthUser string
	BasicAuthPw   string
}

// Returns a scimClient
func NewScimClient(scimConfig Config) (*ScimClient, error) {
	baseURL, err := url.Parse(scimConfig.RawURL)
	if err != nil {
		return nil, err
	}

	httpClient, err := generateHTTPClient(scimConfig)
	if err != nil {
		return nil, err
	}

	scimClient := ScimClient{baseURL, httpClient}
	return &scimClient, nil
}

func generateHTTPClient(scimConfig Config) (httpClient, error) {
	switch scimConfig.AuthType {
	case Basic:
		if scimConfig.BasicAuthConfig == nil {
			return nil, fmt.Errorf("could not create http client, BasicAuthConfig missing")
		}
		basicAuthUser := scimConfig.BasicAuthConfig.BasicAuthUser
		basicAuthPw := scimConfig.BasicAuthConfig.BasicAuthPw
		return basicAuthHTTPClient{basicAuthUser, basicAuthPw, http.Client{}}, nil
	}
	return nil, fmt.Errorf("no client available for %v", scimConfig.AuthType)
}

// Returns team members referenced by URL in a IdP group
func (s *ScimClient) GetTeamMembers(teamMappedIDPGroup string) ([]Member, error) {
	groupEndpoint := s.baseURL.JoinPath(groupPathName)
	params := s.baseURL.Query()
	params.Add(filterQueryKey, fmt.Sprintf(filterQueryExpressionStub, teamMappedIDPGroup))
	groupEndpoint.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, groupEndpoint.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("could not create request %v ", req)
	}

	response, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		dumpedResponse, err := httputil.DumpResponse(response, true)
		if err != nil {
			return nil, fmt.Errorf("failed to dump response: %w", err)
		}
		return nil, fmt.Errorf("could not retrieve TeamMembers from %s : %s", groupEndpoint.String(), string(dumpedResponse))
	}

	var groupResponseBody = new(GroupResponseBody)

	err = json.NewDecoder(response.Body).Decode(&groupResponseBody)
	if err != nil {
		return nil, err
	}

	if groupResponseBody.TotalResults == 0 {
		return nil, fmt.Errorf("no mapped group found for %s", teamMappedIDPGroup)
	}

	if len(groupResponseBody.Resources) == 0 || groupResponseBody.Resources[0].Members == nil {
		return nil, fmt.Errorf("unexpected response format, could not extract members from groupResponseBody %v", groupResponseBody)
	}

	return groupResponseBody.Resources[0].Members, nil
}

// Returns a full fledged Users array from the members array
func (s *ScimClient) GetUsers(members []Member) []greenhousev1alpha1.User {
	var wg sync.WaitGroup
	usersBuffer := make(chan *greenhousev1alpha1.User, len(members))
	wg.Add(len(members))
	for _, member := range members {
		go func(member Member) {
			defer wg.Done()
			user, err := s.getUser(member)
			if err != nil {
				log.Printf(`failed getting user: %s`, err)
			}
			usersBuffer <- user
		}(member)
	}
	wg.Wait()
	users := []greenhousev1alpha1.User{}
	for i := 0; i < cap(usersBuffer); i++ {
		user := <-usersBuffer
		if user != nil {
			users = append(users, *user)
		}
	}
	return users
}

func (s *ScimClient) getUser(member Member) (*greenhousev1alpha1.User, error) {
	req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, member.Ref, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("could not create request %v ", req)
	}

	response, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		dumpedResponse, err := httputil.DumpResponse(response, true)
		if err != nil {
			return nil, fmt.Errorf("failed to dump response: %w", err)
		}
		return nil, fmt.Errorf("could not retrieve TeamMember from %s : %s", member.Ref, string(dumpedResponse))
	}

	var memberResponseBody = new(MemberResponseBody)

	err = json.NewDecoder(response.Body).Decode(&memberResponseBody)
	if err != nil {
		return nil, err
	}

	if !memberResponseBody.Active {
		return nil, fmt.Errorf("user is not active: %v", memberResponseBody)
	}

	user := greenhousev1alpha1.User{}
	if memberResponseBody.UserName != "" && memberResponseBody.Name.GivenName != "" && memberResponseBody.Name.FamilyName != "" && len(memberResponseBody.Emails) > 0 && memberResponseBody.Emails[0].Value != "" {
		user = greenhousev1alpha1.User{ID: memberResponseBody.UserName, FirstName: memberResponseBody.Name.GivenName, LastName: memberResponseBody.Name.FamilyName, Email: memberResponseBody.Emails[0].Value}
	} else {
		return nil, fmt.Errorf("could not create User from memberResponseBody: %v", memberResponseBody)
	}

	return &user, nil
}
