// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"strings"
	"testing"
)

func TestRender_BasicShape(t *testing.T) {
	rules := []Rule{
		{
			// Greenhouse admin: literal greenhouse-admin support_group.
			Alert: "GreenhouseProxyRequestErrorsHigh",
			For:   "15m",
			Labels: map[string]string{
				"severity":      "warning",
				"support_group": "greenhouse-admin",
				"playbook":      "https://example.com/proxy-errors",
			},
			Annotations: map[string]string{
				"summary": "HTTP 5xx errors high for proxy {{ $labels.service }}",
			},
			Group: "proxies",
		},
		{
			// Greenhouse admin: namespace-admin template = control-plane namespace.
			Alert: "GreenhouseOperatorReconcileErrorsHigh",
			For:   "15m",
			Labels: map[string]string{
				"severity":      "warning",
				"support_group": "{{ $labels.namespace }}-admin",
				"playbook":      "https://example.com/operator-errors",
			},
			Annotations: map[string]string{
				"summary": "Errors while reconciling {{$labels.controller}}",
			},
			Group: "operator",
		},
		{
			// Org admin: organization-admin template.
			Alert: "GreenhouseOrganizationNotReady",
			For:   "15m",
			Labels: map[string]string{
				"severity":      "warning",
				"support_group": "{{ $labels.organization }}-admin",
				"playbook":      "https://example.com/org-not-ready",
			},
			Annotations: map[string]string{
				"summary": "Organization is not ready",
			},
			Group: "organization",
		},
		{
			// Support groups: pure template, owned_by routing.
			Alert: "GreenhousePluginNotReady",
			For:   "15m",
			Labels: map[string]string{
				"severity":      "warning",
				"support_group": "{{ $labels.owned_by }}",
				"playbook":      "https://example.com/plugin-not-ready",
			},
			Annotations: map[string]string{
				"summary": "Plugin not ready for over 15 minutes",
			},
			Group: "plugin",
		},
		{
			// Support groups, no playbook: should still render but link
			// target falls back to the em-dash placeholder.
			Alert: "GreenhouseClusterTokenExpiry",
			Labels: map[string]string{
				"severity":      "warning",
				"support_group": "{{ $labels.owned_by }}",
			},
			Annotations: map[string]string{
				"summary": "Cluster token expires soon",
			},
			Group: "cluster",
		},
	}

	out, err := render(rules)
	if err != nil {
		t.Fatalf("render returned error: %v", err)
	}
	s := string(out)

	for _, want := range []string{
		`title: "Playbooks"`,
		`linkTitle: "Playbooks"`,
		"weight: 2",
		"This page is auto-generated",
		"## Greenhouse admin team",
		"## Organization admin team",
		"## Support groups",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("rendered output missing %q", want)
		}
	}

	if strings.Contains(s, "{{") {
		t.Errorf("rendered output contains unescaped {{; want all occurrences replaced with &lcub;&lcub;")
	}
	if !strings.Contains(s, "&lcub;&lcub;") {
		t.Errorf("rendered output is missing the escaped &lcub;&lcub; (test expected at least one Go-template expression to be escaped)")
	}

	ghIdx := strings.Index(s, "## Greenhouse admin team")
	orgIdx := strings.Index(s, "## Organization admin team")
	supIdx := strings.Index(s, "## Support groups")
	if ghIdx >= orgIdx || orgIdx >= supIdx {
		t.Errorf("section ordering broken: gh=%d org=%d sup=%d", ghIdx, orgIdx, supIdx)
	}

	ghSection := s[ghIdx:orgIdx]
	orgSection := s[orgIdx:supIdx]
	supSection := s[supIdx:]

	mustBeIn := func(sectionName, section string, alerts ...string) {
		t.Helper()
		for _, a := range alerts {
			if !strings.Contains(section, a) {
				t.Errorf("%s section missing alert %q", sectionName, a)
			}
		}
	}
	mustNotBeIn := func(sectionName, section string, alerts ...string) {
		t.Helper()
		for _, a := range alerts {
			if strings.Contains(section, a) {
				t.Errorf("%s section unexpectedly contains alert %q", sectionName, a)
			}
		}
	}

	mustBeIn("greenhouse-admin", ghSection, "GreenhouseProxyRequestErrorsHigh", "GreenhouseOperatorReconcileErrorsHigh")
	mustNotBeIn("greenhouse-admin", ghSection, "GreenhouseOrganizationNotReady", "GreenhousePluginNotReady")

	mustBeIn("org-admin", orgSection, "GreenhouseOrganizationNotReady")
	mustNotBeIn("org-admin", orgSection, "GreenhouseProxyRequestErrorsHigh", "GreenhouseOperatorReconcileErrorsHigh", "GreenhousePluginNotReady")

	mustBeIn("support-groups", supSection, "GreenhousePluginNotReady", "GreenhouseClusterTokenExpiry")
	mustNotBeIn("support-groups", supSection, "GreenhouseProxyRequestErrorsHigh", "GreenhouseOrganizationNotReady")

	// Each alert is rendered as a list item whose label is the inline-code
	// alert name wrapped in a Markdown link. The proxy alert has a real
	// playbook URL in our fixture; the cluster alert does not.
	if !strings.Contains(s, "- [`GreenhouseProxyRequestErrorsHigh`](https://example.com/proxy-errors)") {
		t.Errorf("expected list item with linked alert name and playbook URL for the proxy rule; got:\n%s", s)
	}
	// When the source has no playbook URL, the alert name renders as plain
	// inline-code (no link target) rather than a link pointing at "—".
	if !strings.Contains(s, "- `GreenhouseClusterTokenExpiry`") {
		t.Errorf("expected plain inline-code alert name when playbook is absent; got:\n%s", s)
	}
	if strings.Contains(s, "[`GreenhouseClusterTokenExpiry`](—)") {
		t.Errorf("alert without a playbook should not render with an em-dash link target")
	}
}

func TestFormatSummary_EscapesHugoTemplateAndNewlines(t *testing.T) {
	cases := []struct {
		name      string
		in        string
		wantSubs  []string
		notWantSs []string
	}{
		{
			name:     "go-template delimiters are HTML-escaped so Hugo does not interpret them",
			in:       "value {{ $labels.service }} is high",
			wantSubs: []string{"&lcub;&lcub;"},
		},
		{
			name:     "LF newline is collapsed to a space so the list-item continuation stays on one line",
			in:       "first line\nsecond line",
			wantSubs: []string{"first line second line"},
		},
		{
			name:     "CRLF newline is collapsed to a single space",
			in:       "first line\r\nsecond line",
			wantSubs: []string{"first line second line"},
		},
		{
			name:     "empty input renders as em-dash",
			in:       "",
			wantSubs: []string{emDash},
		},
		{
			// Pipes and backticks pass through. The list-layout output does
			// not put summaries inside table cells or inline-code wrappers,
			// so neither character can break the surrounding Markdown.
			name:      "pipe and backtick are not escaped in the list layout",
			in:        "errors | warnings with `code`",
			wantSubs:  []string{"errors | warnings with `code`"},
			notWantSs: []string{`\|`, "&#96;"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatSummary(tc.in)
			for _, w := range tc.wantSubs {
				if !strings.Contains(got, w) {
					t.Errorf("formatSummary(%q) = %q, want it to contain %q", tc.in, got, w)
				}
			}
			for _, nw := range tc.notWantSs {
				if strings.Contains(got, nw) {
					t.Errorf("formatSummary(%q) = %q, want it NOT to contain %q", tc.in, got, nw)
				}
			}
			if strings.ContainsAny(got, "\n\r") {
				t.Errorf("formatSummary(%q) = %q, must not contain raw newlines", tc.in, got)
			}
		})
	}
}
