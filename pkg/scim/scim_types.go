// Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package scim

import (
	"encoding/base64"
	"net/http"
	"net/url"
)

type GroupResponseBody struct {
	TotalResults int        `json:"totalResults"`
	Resources    []Resource `json:"resources"`
}

type Resource struct {
	ID          *string  `json:"id"`
	DisplayName *string  `json:"displayName"`
	Members     []Member `json:"members"`
}

type Member struct {
	Ref string `json:"$ref"`
}

type MemberResponseBody struct {
	ID       string  `json:"id"`
	UserName string  `json:"userName"`
	Emails   []Email `json:"emails"`
	Name     Name    `json:"name"`
	Active   bool    `json:"active"`
}

type Email struct {
	Value   string `json:"value"`
	Primary bool   `json:"primary"`
}

type Name struct {
	FamilyName string `json:"familyName"`
	GivenName  string `json:"givenname"`
}

type ScimClient struct {
	baseURL    *url.URL
	httpClient httpClient
}

type AuthType byte

const (
	Basic AuthType = iota
)

type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type basicAuthHTTPClient struct {
	basicAuthUser string
	basicAuthPw   string
	c             http.Client
}

func (c basicAuthHTTPClient) Do(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", "Basic "+basicAuth(c.basicAuthUser, c.basicAuthPw))
	return c.c.Do(req)
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
