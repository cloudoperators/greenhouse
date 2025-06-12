// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WithSecretType sets the type of the Secret
func WithSecretType(secretType corev1.SecretType) func(*corev1.Secret) {
	return func(s *corev1.Secret) {
		s.Type = secretType
	}
}

func WithSecretAnnotations(annotations map[string]string) func(*corev1.Secret) {
	return func(s *corev1.Secret) {
		s.SetAnnotations(annotations)
	}
}

func WithSecretLabels(labels map[string]string) func(*corev1.Secret) {
	return func(s *corev1.Secret) {
		s.SetLabels(labels)
	}
}

// WithSecretData sets the data of the Secret
func WithSecretData(data map[string][]byte) func(*corev1.Secret) {
	return func(s *corev1.Secret) {
		s.Data = data
	}
}

// WithSecretNamespace sets the namespace of the Secret
func WithSecretNamespace(namespace string) func(*corev1.Secret) {
	return func(s *corev1.Secret) {
		s.Namespace = namespace
	}
}

// NewSecret returns a Secret object. Opts can be used to set the desired state of the Secret.
func NewSecret(name, namespace string, opts ...func(*corev1.Secret)) *corev1.Secret {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.GroupName,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	for _, opt := range opts {
		opt(secret)
	}
	return secret
}
