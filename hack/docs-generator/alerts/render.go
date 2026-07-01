// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"sort"
	"strings"
	"text/template"
)

//go:embed template.md
var pageTemplate string

const emDash = "—"

type row struct {
	Alert    string
	Summary  string
	Playbook string
}

type section struct {
	Heading     string
	Description string
	Rules       []row
}

type page struct {
	Sections []section
}

// render produces the Markdown page bytes from a flat list of rules.
// Each rule must have its Group field populated.
func render(rules []Rule) ([]byte, error) {
	ruleBuckets := map[Bucket][]Rule{
		AdminBucket:         {},
		OrgAdminBucket:      {},
		SupportGroupsBucket: {},
	}
	for _, r := range rules {
		b := classify(r)
		ruleBuckets[b] = append(ruleBuckets[b], r)
	}

	p := page{
		Sections: []section{
			{
				Heading:     "Greenhouse admin team",
				Description: "Alerts that page the Greenhouse platform team (`greenhouse-admin`). These cover Greenhouse itself.",
				Rules:       buildRows(ruleBuckets[AdminBucket]),
			},
			{
				Heading:     "Organization admin team",
				Description: "Alerts that page a tenant organization's admin team (`<org>-admin`). These cover tenant-organization-level resources.",
				Rules:       buildRows(ruleBuckets[OrgAdminBucket]),
			},
			{
				Heading:     "Support groups",
				Description: "Alerts that page the team owning the affected resource via its `owned_by` label. These cover tenant-managed resources — Plugins, Clusters, Catalogs, TeamRoleBindings.",
				Rules:       buildRows(ruleBuckets[SupportGroupsBucket]),
			},
		},
	}

	tmpl, err := template.New("alerts").Parse(pageTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, p); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}
	return buf.Bytes(), nil
}

// buildRows sorts rules by source group (filename) then by appearance order
// inside that group, and converts them to renderRows.
func buildRows(rules []Rule) []row {
	sort.SliceStable(rules, func(i, j int) bool {
		return rules[i].Group < rules[j].Group
	})
	out := make([]row, 0, len(rules))
	for _, r := range rules {
		out = append(out, toRow(r))
	}
	return out
}

func toRow(r Rule) row {
	return row{
		Alert:    r.Alert,
		Summary:  formatSummary(r.Annotations["summary"]),
		Playbook: orDash(r.Labels["playbook"]),
	}
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return emDash
	}
	return s
}

// formatSummary neutralises characters in the alert summary that would
// otherwise produce broken or surprising Markdown when the page is rendered:
// Hugo `{{` template delimiters, and embedded newlines that would break the
// list-item continuation.
func formatSummary(s string) string {
	if strings.TrimSpace(s) == "" {
		return emDash
	}
	escaped := strings.ReplaceAll(s, "{{", "&lcub;&lcub;")
	escaped = strings.ReplaceAll(escaped, "\r\n", " ")
	escaped = strings.ReplaceAll(escaped, "\n", " ")
	return escaped
}
