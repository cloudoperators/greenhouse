// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"path/filepath"
	"testing"
)

func TestParseFile_PluginAlerts(t *testing.T) {
	path := filepath.Join("..", "..", "..", "charts", "manager", "alerts", "plugin.alerts")
	rules, err := parseFile(path)
	if err != nil {
		t.Fatalf("parseFile returned error: %v", err)
	}
	if len(rules) != 3 {
		t.Fatalf("expected 3 rules in plugin.alerts, got %d", len(rules))
	}
	want := map[string]bool{
		"GreenhousePluginNotReady":             false,
		"GreenhousePluginPresetNotReconciled":  false,
		"GreenhousePluginConstantlyFailing":    false,
	}
	for _, r := range rules {
		if _, ok := want[r.Alert]; !ok {
			t.Errorf("unexpected alert: %q", r.Alert)
			continue
		}
		want[r.Alert] = true
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("missing alert: %q", name)
		}
	}

	// Spot-check fields on GreenhousePluginNotReady.
	for _, r := range rules {
		if r.Alert != "GreenhousePluginNotReady" {
			continue
		}
		if r.For != "15m" {
			t.Errorf("GreenhousePluginNotReady: For = %q, want %q", r.For, "15m")
		}
		if r.Labels["severity"] != "warning" {
			t.Errorf("GreenhousePluginNotReady: severity = %q, want %q", r.Labels["severity"], "warning")
		}
		if r.Labels["support_group"] != "{{ $labels.owned_by }}" {
			t.Errorf("GreenhousePluginNotReady: support_group = %q, want %q", r.Labels["support_group"], "{{ $labels.owned_by }}")
		}
		if r.Annotations["summary"] != "Plugin not ready for over 15 minutes" {
			t.Errorf("GreenhousePluginNotReady: summary = %q", r.Annotations["summary"])
		}
		if r.Labels["playbook"] == "" {
			t.Errorf("GreenhousePluginNotReady: playbook label is empty")
		}
	}
}
