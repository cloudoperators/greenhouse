// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"regexp"
	"strings"
)

type Bucket int

const (
	// AdminBucket: alerts routed to the Greenhouse platform admin team.
	// The support_group is either the literal "greenhouse-admin" or a
	// `{{ $labels.namespace }}-admin` template — namespace is the
	// Greenhouse-managed control-plane namespace, so the resulting team
	// name is still a platform-team destination.
	AdminBucket Bucket = iota
	// OrgAdminBucket: alerts routed to a tenant organization's admin team.
	// The support_group renders to "<org>-admin" via a
	// `{{ $labels.organization }}-admin` template, where organization is
	// the tenant org label.
	OrgAdminBucket
	// SupportGroupsBucket: alerts routed to the team owning the affected
	// resource via its `owned_by` label, or any other non-admin destination.
	SupportGroupsBucket
)

func (b Bucket) String() string {
	switch b {
	case AdminBucket:
		return "admin"
	case OrgAdminBucket:
		return "org-admin"
	case SupportGroupsBucket:
		return "support-groups"
	default:
		return "unknown"
	}
}

var (
	templateExprRE = regexp.MustCompile(`\{\{[^{}]*\}\}`)
	// orgLabelRE matches a Go-template expression that resolves the
	// `organization` label, with or without a leading `$` and arbitrary
	// surrounding whitespace: `{{ $labels.organization }}`, `{{.Labels.organization}}`,
	// etc. Presence of this expression in the support_group is what
	// distinguishes a tenant-org admin alert from a Greenhouse-admin alert.
	orgLabelRE = regexp.MustCompile(`\{\{[^{}]*\b[lL]abels\.organization\b[^{}]*\}\}`)
)

// classify decides which bucket a rule belongs to based on its support_group label.
//
// Bucketing rule (the dispatcher comments in source files are intentionally
// not consulted — only the actual template contents):
//
//   - Missing or empty support_group → AdminBucket
//   - Literal "greenhouse-admin" → AdminBucket.
//   - support_group references the `organization` label inside a Go template
//     expression (e.g. `{{ $labels.organization }}-admin`) → OrgAdminBucket.
//     The rendered value is a tenant-org admin team name.
//   - support_group ends with "-admin" via any other template (typically
//     `{{ $labels.namespace }}-admin`) → AdminBucket. Namespace here is the
//     Greenhouse control-plane namespace, so the team is still platform.
//   - Anything else (including pure templates that strip to empty, like
//     `{{ $labels.owned_by }}`) → SupportGroupsBucket.
func classify(r Rule) Bucket {
	sg, ok := r.Labels["support_group"]
	if !ok || sg == "" {
		return AdminBucket
	}
	if sg == "greenhouse-admin" {
		return AdminBucket
	}
	if orgLabelRE.MatchString(sg) {
		return OrgAdminBucket
	}
	stripped := templateExprRE.ReplaceAllString(sg, "")
	if stripped == "" {
		return SupportGroupsBucket
	}
	if strings.HasSuffix(stripped, "-admin") {
		return AdminBucket
	}
	return SupportGroupsBucket
}
