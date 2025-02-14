// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package scim

import (
	"fmt"
	"net/url"
	"strings"
)

type AuthType string

const (
	Basic       AuthType = "basic"
	BearerToken AuthType = "token"
)

const (
	InitialStartID = "initial"
)

type Filter struct {
	expression string
}

func newFilter(attribute, operator, value string) Filter {
	return Filter{
		expression: fmt.Sprintf("%s %s \"%s\"", attribute, operator, value),
	}
}

func (f Filter) String() string {
	return f.expression
}

func GroupFilterByDisplayName(displayName string) Filter {
	return newFilter("displayName", "eq", displayName)
}

func UserFilterByGroupDisplayName(displayName string) Filter {
	return newFilter("groups.display", "eq", displayName)
}

// Attribute represents a SCIM attribute
type Attribute struct {
	attribute string
}

// Predefined SCIM attributes
var (
	AttrName        = Attribute{attribute: "name"}
	AttrEmails      = Attribute{attribute: "emails"}
	AttrDisplayName = Attribute{attribute: "displayName"}
	AttrActive      = Attribute{attribute: "active"}
	AttrMembers     = Attribute{attribute: "members"}
)

func AttrCustom(name string) Attribute {
	return Attribute{attribute: name}
}

// String returns the attribute string
func (a Attribute) String() string {
	return a.attribute
}

// SetAttributes sets a comma-separated string for multiple attributes
func SetAttributes(attributes ...Attribute) string {
	var attrStrings []string
	for _, attr := range attributes {
		attrStrings = append(attrStrings, attr.String())
	}
	return strings.Join(attrStrings, ",")
}

type QueryOptions struct {
	Filter             Filter
	Attributes         string
	ExcludedAttributes string
	StartID            string
}

func (q *QueryOptions) toQuery() string {
	values := url.Values{}
	if strings.TrimSpace(q.Filter.String()) != "" {
		values.Add("filter", q.Filter.String())
	}
	if strings.TrimSpace(q.Attributes) != "" {
		values.Add("attributes", q.Attributes)
	}
	if strings.TrimSpace(q.ExcludedAttributes) != "" {
		values.Add("excludedAttributes", q.ExcludedAttributes)
	}
	if strings.TrimSpace(q.StartID) != "" {
		values.Add("startId", q.StartID)
	}
	return values.Encode()
}

type Resource struct {
	ID          string        `json:"id"`
	Meta        ResourceMeta  `json:"meta"`
	UserName    string        `json:"userName"`
	Name        UserNameField `json:"name"`
	DisplayName string        `json:"displayName"`
	Active      bool          `json:"active"`
	Emails      []UserEmails  `json:"emails"`
}

type UserEmails struct {
	Value   string `json:"value"`
	Primary bool   `json:"primary"`
}

type ResourceMeta struct {
	Location     string `json:"location"`
	Version      string `json:"version"`
	ResourceType string `json:"resourceType"`
}

type UserNameField struct {
	FamilyName string `json:"familyName"`
	GivenName  string `json:"givenName"`
}

type ResponseBody struct {
	TotalResults int        `json:"totalResults"`
	ItemsPerPage int        `json:"itemsPerPage"`
	Resources    []Resource `json:"Resources"`
	StartID      string     `json:"startId"`
	NextID       string     `json:"nextId"`
}

func (r Resource) FirstName() string {
	return r.Name.GivenName
}

func (r Resource) LastName() string {
	return r.Name.FamilyName
}

func (r Resource) ActiveUser() bool {
	return r.Active
}

func (r Resource) PrimaryEmail() string {
	for _, email := range r.Emails {
		if email.Primary {
			return email.Value
		}
	}
	return ""
}
