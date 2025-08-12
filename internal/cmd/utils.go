// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/util/yaml"
)

func loadFromFile[T any](path string) (*T, error) {
	var result T
	err := loadAndUnmarshalObject(path, &result)
	return &result, err
}

func loadAndUnmarshalObject(path string, o any) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	f, err := os.ReadFile(absPath)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(f, o)
}
