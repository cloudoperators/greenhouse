// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package clientutil

import (
	"os"
	"strconv"
)

// GetEnvOrDefault returns the value of the environment variable or the default value.
func GetEnvOrDefault(envKey, defaultValue string) string {
	if v, ok := os.LookupEnv(envKey); ok {
		return v
	}
	return defaultValue
}

func GetIntEnvWithDefault(envKey string, def int) int {
	s := os.Getenv(envKey)
	i, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return i
}
