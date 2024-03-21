// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

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
