// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRelative returns paths relative to the repository root from this test file.
// The test file lives at hack/docs-generator/alerts/, so the repo root is three levels up.
func repoRelative(t *testing.T, parts ...string) string {
	t.Helper()
	all := append([]string{"..", "..", ".."}, parts...)
	return filepath.Join(all...)
}

func TestRunAgainstRealSource(t *testing.T) {
	alertsDir := repoRelative(t, "charts", "manager", "alerts")
	outDir := t.TempDir()
	outFile := filepath.Join(outDir, "_index.md")

	if err := run(alertsDir, outFile); err != nil {
		t.Fatalf("run: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	out := string(data)
	if len(strings.TrimSpace(out)) == 0 {
		t.Fatal("output is empty")
	}

	for _, want := range []string{
		"## Greenhouse admin team",
		"## Organization admin team",
		"## Support groups",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing section heading %q", want)
		}
	}

	alertNames := collectAlertNamesFromSource(t, alertsDir)
	if len(alertNames) == 0 {
		t.Fatal("no alert names collected from source — test setup is wrong")
	}
	for _, name := range alertNames {
		if !strings.Contains(out, name) {
			t.Errorf("output missing alert %q", name)
		}
	}

	if strings.Contains(out, "{{") {
		t.Errorf("output contains unescaped {{ — Hugo escape is broken")
	}

	ghIdx := strings.Index(out, "## Greenhouse admin team")
	orgIdx := strings.Index(out, "## Organization admin team")
	supIdx := strings.Index(out, "## Support groups")
	if ghIdx < 0 || orgIdx < 0 || supIdx < 0 || !(ghIdx < orgIdx && orgIdx < supIdx) {
		t.Fatalf("section ordering broken: gh=%d org=%d sup=%d", ghIdx, orgIdx, supIdx)
	}
	ghSection := out[ghIdx:orgIdx]
	orgSection := out[orgIdx:supIdx]
	supSection := out[supIdx:]

	mustBeIn := func(sectionName, section string, alerts ...string) {
		t.Helper()
		for _, a := range alerts {
			if !strings.Contains(section, a) {
				t.Errorf("%s section missing alert %q", sectionName, a)
			}
		}
	}

	// Proxies use the literal `greenhouse-admin`; operator uses
	// `{{ $labels.namespace }}-admin` which is the Greenhouse control-plane
	// namespace, so it bucketed as platform admin.
	mustBeIn("greenhouse-admin",
		ghSection,
		"GreenhouseProxyRequestErrorsHigh",
		"GreenhouseOperatorReconcileErrorsHigh",
	)
	// Organization, resource, and the team-membership-count-drop rule use
	// the `{{ $labels.organization }}-admin` template — these page the
	// tenant org admin team.
	mustBeIn("org-admin",
		orgSection,
		"GreenhouseOrganizationNotReady",
		"GreenhouseResourceOwnedByLabelMissing",
		"GreenhouseTeamMembershipCountDrop",
	)
	// Cluster, plugin, catalog, and the team-role-binding rule are routed
	// via `owned_by`.
	mustBeIn("support-groups",
		supSection,
		"GreenhouseClusterNotReady",
		"GreenhousePluginNotReady",
		"GreenhouseTeamRoleBindingNotReady",
	)

	// team.alerts contributes one rule to org-admin and one to support-groups
	// — the file split itself does not determine bucketing.
	if !(strings.Contains(orgSection, "GreenhouseTeamMembershipCountDrop") &&
		strings.Contains(supSection, "GreenhouseTeamRoleBindingNotReady")) {
		t.Error("team.alerts must contribute rules to both org-admin and support-groups buckets")
	}

	// Spot-check that alert names render as Markdown links pointing at the
	// playbook URL from the source.
	const wantLink = "[`GreenhouseProxyRequestErrorsHigh`](https://cloudoperators.github.io/greenhouse/docs/operations/playbooks/proxy/proxy-request-errors-high/)"
	if !strings.Contains(out, wantLink) {
		t.Errorf("expected linked-alert-name list item not found:\n  want substring: %s", wantLink)
	}
}

// collectAlertNamesFromSource greps the source files for `alert:` lines so the
// test stays in sync with the source rather than hard-coding the full list.
func collectAlertNamesFromSource(t *testing.T, alertsDir string) []string {
	t.Helper()
	entries, err := os.ReadDir(alertsDir)
	if err != nil {
		t.Fatalf("read alerts dir: %v", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".alerts" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(alertsDir, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			const prefix = "- alert:"
			if !strings.HasPrefix(line, prefix) {
				continue
			}
			name := strings.TrimSpace(strings.TrimPrefix(line, prefix))
			if name != "" {
				names = append(names, name)
			}
		}
	}
	return names
}
