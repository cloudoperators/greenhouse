// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"strings"
	"testing"
)

func TestExtractErrorsFromTestPodLogs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"Single not ok log",
			"1..1\nnot ok 1 Test Health\n# (in test file /tests/run.sh, line 5)\n#   `[ \"$code\" == \"200\" ]' failed\n",
			"not ok 1 Test Health\n# (in test file /tests/run.sh, line 5)\n#   `[ \"$code\" == \"200\" ]' failed\n",
		},
		{
			"Mixed logs with ok and not ok",
			"1..9\nnot ok 1 Verify successful deployment\n# Failure details\nok 2 Success message\nnot ok 3 Another failure\n# More failure details\n# Additional debug info\n# Checking deployment state\nnot ok 4 Third failure\n# Additional failure details\nnot ok 5 Fourth failure\n# More deep failure details",
			"not ok 1 Verify successful deployment\n# Failure details\n\nnot ok 3 Another failure\n# More failure details\n# Additional debug info\n# Checking deployment state\n\nnot ok 4 Third failure\n# Additional failure details\n\nnot ok 5 Fourth failure\n# More deep failure details",
		},
		{
			"Don't capture logs when the test doesn't use bats framework",
			"wget: bad address 'testietest:9771'\n",
			"",
		},
		{
			"Ignore unrelated system errors",
			"[ERROR] Could not connect to database\n[INFO] Retrying in 5 seconds\n",
			"",
		},
		{
			"Ignore generic runtime logs",
			"Starting test execution...\nTest suite initialized\nAll dependencies resolved\n",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractErrorsFromTestPodLogs(tt.input)
			if strings.TrimSpace(result) != strings.TrimSpace(tt.expected) {
				t.Errorf("Test Failed: %s\nExpected:\n%q\nGot:\n%q", tt.name, tt.expected, result)
			}
		})
	}
}
