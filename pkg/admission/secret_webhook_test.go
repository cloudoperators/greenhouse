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

package admission

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	"github.com/cloudoperators/greenhouse/pkg/test"
)

var _ = Describe("Validate Secret Creation based on type", func() {
	DescribeTable("Validate secret creation with different secret types", func(secretType corev1.SecretType, dataKey string, expErr bool) {
		var secret *corev1.Secret
		if dataKey != "" {
			secret = &corev1.Secret{
				Type: secretType,
				Data: map[string][]byte{
					dataKey: []byte("something"),
				},
			}
		} else {
			secret = &corev1.Secret{
				Type: secretType,
			}
		}
		err := validateSecretGreenHouseType(secret)
		switch expErr {
		case true:
			Expect(err).To(HaveOccurred(), "expected an error, got nil")
		default:
			Expect(err).ToNot(HaveOccurred(), "expected no error, got %v", err)
		}
	},
		Entry("Secret.type is not greenhouse.sap/kubeconfig no data set", corev1.SecretType("not/greenhouse.sap/kubeconfig"), "", false),
		Entry("Secret.type is not greenhouse.sap/kubeconfig with data.kubeconfig", corev1.SecretType("not/greenhouse.sap/kubeconfig"), "kubeconfig", false),
		Entry("Secret.type is not greenhouse.sap/kubeconfig with data.greenhousekubeconfig", corev1.SecretType("not/greenhouse.sap/kubeconfig"), "greenhousekubeconfig", false),
		Entry("Secret.type is greenhouse.sap/kubeconfig with data.kubeconfig", corev1.SecretType("greenhouse.sap/kubeconfig"), "kubeconfig", false),
		Entry("Secret.type is greenhouse.sap/kubeconfig with data.greenhousekubeconfig", corev1.SecretType("greenhouse.sap/kubeconfig"), "greenhousekubeconfig", true),
		Entry("Secret.type is greenhouse.sap/kubeconfig no data set", corev1.SecretType("greenhouse.sap/kubeconfig"), "", true),
	)

	DescribeTable("Validate secret creation with different secret types and kubeconfig", func(secretType corev1.SecretType, dataKey string, dataValue []byte, expErr bool) {
		var secret *corev1.Secret
		if dataKey != "" {
			secret = &corev1.Secret{
				Type: secretType,
				Data: map[string][]byte{
					dataKey: dataValue,
				},
			}
		} else {
			secret = &corev1.Secret{
				Type: secretType,
			}
		}

		err := validateKubeconfigInSecret(secret)
		switch expErr {
		case true:
			Expect(err).To(HaveOccurred(), "expected an error, got nil")
		default:
			Expect(err).ToNot(HaveOccurred(), "expected no error, got %v", err)
		}
	},
		Entry("Secret.type is greenhouse.sap/kubeconfig but data.kubeconfig is empty", corev1.SecretType("greenhouse.sap/kubeconfig"), "kubeconfig", []byte(""), true),
		Entry("Secret.type is greenhouse.sap/kubeconfig but data.kubeconfig is invalid", corev1.SecretType("greenhouse.sap/kubeconfig"), "kubeconfig", []byte("invalid"), true),
		Entry("Secret.type is greenhouse.sap/kubeconfig and data.kubeconfig is valid kubeconfig", corev1.SecretType("greenhouse.sap/kubeconfig"), "kubeconfig", test.KubeConfig, false),
		Entry("Secret.type is greenhouse.sap/kubeconfig and data.greenhousekubeconfig is empty", corev1.SecretType("greenhouse.sap/kubeconfig"), "greenhousekubeconfig", []byte(""), true),
		Entry("Secret.type is greenhouse.sap/kubeconfig and data.greenhousekubeconfig is not a valid kubeconfig", corev1.SecretType("greenhouse.sap/kubeconfig"), "greenhousekubeconfig", []byte("invalid"), true),
		Entry("Secret.type is greenhouse.sap/kubeconfig and data.greenhousekubeconfig is a valid kubeconfig", corev1.SecretType("greenhouse.sap/kubeconfig"), "greenhousekubeconfig", test.KubeConfig, false),
	)

})
