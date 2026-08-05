// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	alertsDir := flag.String("alerts-dir", "charts/manager/alerts", "Directory containing .alerts files")
	outputFile := flag.String("output", "docs/operations/playbooks/_index.md", "Path to write the rendered Markdown page")
	flag.Parse()

	if err := run(*alertsDir, *outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "alerts-doc generator: %v\n", err)
		os.Exit(1)
	}
}

func run(alertsDir, outputFile string) error {
	entries, err := os.ReadDir(alertsDir)
	if err != nil {
		return fmt.Errorf("read alerts dir %s: %w", alertsDir, err)
	}
	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) != ".alerts" {
			continue
		}
		files = append(files, e.Name())
	}
	sort.Strings(files)

	var allRules []Rule
	for _, name := range files {
		rules, err := parseFile(filepath.Join(alertsDir, name))
		if err != nil {
			return err
		}
		group := strings.TrimSuffix(name, ".alerts")
		for i := range rules {
			rules[i].Group = group
			if strings.TrimSpace(rules[i].Labels["support_group"]) == "" {
				fmt.Fprintf(os.Stderr, "alerts-doc generator: warning: %s/%s has no support_group label, defaulting to admin bucket\n", group, rules[i].Alert)
			}
		}
		allRules = append(allRules, rules...)
	}

	out, err := render(allRules)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(outputFile), 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	if err := os.WriteFile(outputFile, out, 0o644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}
