// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"encoding/base64"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var _ = Describe("Validate Secret Creation based on type", func() {
	DescribeTable("Validate secret creation with different secret types", func(secretType corev1.SecretType, dataKey string, expErr bool) {
		var secret *corev1.Secret
		var ctx context.Context
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
		err := validateSecretGreenHouseType(ctx, secret)
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

	DescribeTable("Validate OIDC secret creation",
		func(annotations map[string]string, certData []byte, expErr bool) {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "oidc-secret",
					Namespace:   corev1.NamespaceDefault,
					Annotations: annotations,
				},
				Type: corev1.SecretType("greenhouse.sap/oidc-config"),
				Data: map[string][]byte{
					greenhouseapis.SecretAPIServerCAKey: certData,
				},
			}

			err := validateGreenhouseOIDCType(secret)

			if expErr {
				Expect(err).To(HaveOccurred(), "expected an error, but got nil")
			} else {
				Expect(err).ToNot(HaveOccurred(), "expected no error, but got %v", err)
			}
		},
		Entry("Valid APIServerURL but missing certificate authority key", map[string]string{greenhouseapis.SecretAPIServerURLAnnotation: "https://example.com"}, nil, true),
		Entry("Missing APIServerURL annotation", nil, []byte("validBase64EncodedCert"), true),
		Entry("Invalid APIServerURL (http scheme)", map[string]string{greenhouseapis.SecretAPIServerURLAnnotation: "http://example.com"}, []byte("validBase64EncodedCert"), true),
		Entry("Invalid APIServerURL (malformed URL)", map[string]string{greenhouseapis.SecretAPIServerURLAnnotation: "not-a-url"}, []byte("validBase64EncodedCert"), true),
		Entry("Valid APIServerURL with valid base64 certificate", map[string]string{greenhouseapis.SecretAPIServerURLAnnotation: "https://example.com"}, []byte(base64.StdEncoding.EncodeToString([]byte("valid-cert"))), false),
	)
})
