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
			"1..1\nnot ok 1 Test Health\n# (in test file /tests/run.sh, line 5)\n#   `[ \"$code\" == \"200\" ]' failed\nstream closed EOF for kube-monitoring/plutono-test (plutono-test)",
			"not ok 1 Test Health\n# (in test file /tests/run.sh, line 5)\n#   `[ \"$code\" == \"200\" ]' failed\nstream closed EOF for kube-monitoring/plutono-test (plutono-test)",
		},
		{
			"Mixed logs with ok and not ok",
			"1..9\nnot ok 1 Verify successful deployment\n# Failure details\nok 2 Success message\nnot ok 3 Another failure\n# More failure details\n# Additional debug info\n# Checking deployment state\nnot ok 4 Third failure\n# Additional failure details\nnot ok 5 Fourth failure\n# More deep failure details",
			"not ok 1 Verify successful deployment\n# Failure details\n\nnot ok 3 Another failure\n# More failure details\n# Additional debug info\n# Checking deployment state\n\nnot ok 4 Third failure\n# Additional failure details\n\nnot ok 5 Fourth failure\n# More deep failure details",
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
