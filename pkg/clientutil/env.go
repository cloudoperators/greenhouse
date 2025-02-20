// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package clientutil

import (
	"fmt"
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

// GetIntEnvWithDefault returns the integer value of the environment variable or the default value.
func GetIntEnvWithDefault(envKey string, def int) int {
	s := os.Getenv(envKey)
	i, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return i
}

// GetEnv returns the value of the environment variable or an error if it is not set
func GetEnv(envKey string) (string, error) {
	if v, ok := os.LookupEnv(envKey); ok {
		return v, nil
	}
	return "", fmt.Errorf("environment variable '%s' not set", envKey)
}
