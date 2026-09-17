// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"testing"
)

func TestLabelSafeVersion(t *testing.T) {
	tests := []struct {
		input  string
		want   string
		wantOK bool
	}{
		// standard semver
		{"v1.31.4", "v1.31.4", true},
		// build metadata with '+'
		{"v1.31.4+k3s1", "v1.31.4-k3s1", true},
		// multiple '+' segments
		{"v1.31.4+build.1+meta", "v1.31.4-build.1-meta", true},
		// exactly 63 chars — no truncation needed
		{"v1." + "123456789012345678901234567890123456789012345678901234567890", "v1." + "123456789012345678901234567890123456789012345678901234567890", true},
		// 64 chars — truncated to 63, no trailing separator
		{"v1." + "1234567890123456789012345678901234567890123456789012345678901", "v1." + "123456789012345678901234567890123456789012345678901234567890", true},
		// 63 chars ending in '.' — truncated trailing '.' stripped
		{"v" + "1234567890123456789012345678901234567890123456789012345678901.", "v" + "1234567890123456789012345678901234567890123456789012345678901", true},
		// leading separator after '+' replacement stripped
		{"+v1.0", "v1.0", true},
		// only separators after sanitization → invalid
		{"+++", "", false},
		// empty string → caller never calls this but function handles it
		{"", "", false},
	}

	for _, tc := range tests {
		got, ok := labelSafeVersion(tc.input)
		if ok != tc.wantOK {
			t.Errorf("labelSafeVersion(%q): ok=%v, want %v (got %q)", tc.input, ok, tc.wantOK, got)
			continue
		}
		if ok && got != tc.want {
			t.Errorf("labelSafeVersion(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
