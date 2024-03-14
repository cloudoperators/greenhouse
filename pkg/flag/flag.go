// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package flag

import (
	"fmt"
	"os"
	"strings"

	flag "github.com/spf13/pflag"
)

var (
	//ignore a few stupid flags that come with dependencies
	flagBlacklist  = []string{"v", "version", "vmodule", "master", "alsologtostderr", "kubeconfig", "log_backtrace_at", "log_dir", "stderrthreshold", "logtostderr"}
	envVarReplacer = strings.NewReplacer(
		"-", "_",
		".", "_",
		"/", "_",
	)
)

// Parse sets flags that have not been set explicitly from environment variables
// This is a trimmed down version of https://github.com/peterbourgon/ff/blob/a2a0e274f2e9702f96865c2c31d9238129432dca/parse.go#L15
func Parse(fs *flag.FlagSet, args []string) error {
	// First priority: commandline flags (explicit user preference).
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("error parsing commandline args: %w", err)
	}

	provided := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		provided[f.Name] = true
	})

	blacklisted := map[string]bool{}
	for _, name := range flagBlacklist {
		blacklisted[name] = true
	}

	var visitErr error
	fs.VisitAll(func(f *flag.Flag) {
		if visitErr != nil {
			return
		}

		if provided[f.Name] || blacklisted[f.Name] {
			return
		}

		var key = strings.ToUpper(f.Name)
		key = envVarReplacer.Replace(key)
		value := os.Getenv(key)
		if value == "" {
			return
		}
		if err := fs.Set(f.Name, value); err != nil {
			visitErr = fmt.Errorf("error setting flag %q from env var %q: %w", f.Name, key, err)
		}
	})
	return visitErr
}
