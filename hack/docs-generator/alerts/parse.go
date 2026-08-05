// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"

	"sigs.k8s.io/yaml"
)

// Rule is one alerting rule extracted from a .alerts file.
// Group is filled in by the caller (parseFile leaves it empty); see main.go.
type Rule struct {
	Alert       string            `json:"alert"`
	Expression  string            `json:"expr"`
	For         string            `json:"for"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`

	// Group is the source filename without extension (e.g. "plugin").
	// Filled in by the caller after parsing.
	Group string `json:"-"`
}

// fileSchema mirrors the top-level YAML structure of a .alerts file.
type fileSchema struct {
	Groups []struct {
		Name  string `json:"name"`
		Rules []Rule `json:"rules"`
	} `json:"groups"`
}

// parseFile reads a single .alerts file and returns its flattened rules.
// All groups inside the file are concatenated; rule order matches source order.
func parseFile(path string) ([]Rule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var f fileSchema
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	var out []Rule
	for _, g := range f.Groups {
		out = append(out, g.Rules...)
	}
	return out, nil
}
