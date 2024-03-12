// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clientutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

// IsSecretContainsKey checks whether the given secret contains a key.
func IsSecretContainsKey(s *corev1.Secret, key string) bool {
	if s.Data == nil {
		return false
	}
	v, ok := s.Data[key]
	return ok && v != nil
}

// GetSecretKeyFromSecretKeyReference returns the value of the secret identified by SecretKeyReference or an error.
func GetSecretKeyFromSecretKeyReference(ctx context.Context, c client.Client, namespace string, secretReference greenhousev1alpha1.SecretKeyReference) (string, error) {
	var secret = new(corev1.Secret)
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: secretReference.Name}, secret); err != nil {
		return "", err
	}
	if v, ok := secret.Data[secretReference.Key]; ok {
		// Trim newline characters from the end of the string.
		stringValue := string(v)
		stringValue = strings.TrimRight(stringValue, "\n")
		return stringValue, nil
	}
	return "", fmt.Errorf("secret %s/%s does not contain key %s", namespace, secretReference.Name, secretReference.Key)
}

// GetKubernetesVersion returns the kubernetes git version using the discovery client.
func GetKubernetesVersion(restClientGetter genericclioptions.RESTClientGetter) (*version.Info, error) {
	dc, err := restClientGetter.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	return dc.ServerVersion()
}

// Searches for a directory upwards starting from the given path.
func FindDirUpwards(path, dirName string, maxSteps int) (string, error) {
	return findRecursively(path, dirName, maxSteps, 0)
}

func findRecursively(path, dirName string, maxSteps, steps int) (string, error) {
	if path == "/" {
		return "", fmt.Errorf("root reached. directory not found: %s", dirName)
	}
	if maxSteps == steps {
		return "", fmt.Errorf("max steps reached. directory not found: %s", dirName)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	dirPath := filepath.Join(absPath, dirName)
	if _, err = os.Stat(dirPath); err != nil {
		if os.IsNotExist(err) {
			steps++
			return findRecursively(filepath.Join(absPath, ".."), dirName, maxSteps, steps)
		}
		return "", err
	}
	return dirPath, nil
}
