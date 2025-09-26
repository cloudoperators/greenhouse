// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package flux

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

var _ = Describe("Registry Mirror Configuration", func() {
	var (
		ctx        context.Context
		fakeClient client.Client
		scheme     *runtime.Scheme
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		Expect(greenhousev1alpha1.AddToScheme(scheme)).To(Succeed())
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
	})

	Describe("GetRegistryMirrorConfig", func() {
		var (
			plugin *greenhousev1alpha1.Plugin
		)

		BeforeEach(func() {
			plugin = &greenhousev1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-plugin",
					Namespace: "test-org",
				},
			}
		})

		Context("when organization and configmap exist with valid registry mirror config", func() {
			BeforeEach(func() {
				org := &greenhousev1alpha1.Organization{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-org",
					},
					Spec: greenhousev1alpha1.OrganizationSpec{
						ConfigMapRef: "test-configmap",
					},
				}

				configMap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-configmap",
						Namespace: "test-org",
					},
					Data: map[string]string{
						registryMirrorConfigKey: `primaryMirror: "primary.registry.com"
registryMirrors:
  ghcr.io:
    basedomain: "europe.registry.com"
    subPath: "ghcr-mirror"
  docker.io:
    basedomain: "mirror.registry.com"
    subPath: "dockerhub-mirror"`,
					},
				}

				fakeClient = fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(org, configMap).
					Build()
			})

			It("should successfully parse registry mirror configuration", func() {
				config, err := GetRegistryMirrorConfig(ctx, fakeClient, plugin)

				Expect(err).NotTo(HaveOccurred())
				Expect(config).NotTo(BeNil())
				Expect(config.PrimaryMirror).To(Equal("primary.registry.com"))
				Expect(config.RegistryMirrors).To(HaveLen(2))
				Expect(config.RegistryMirrors["ghcr.io"]).To(Equal(RegistryMirror{
					BaseDomain: "europe.registry.com",
					SubPath:    "ghcr-mirror",
				}))
				Expect(config.RegistryMirrors["docker.io"]).To(Equal(RegistryMirror{
					BaseDomain: "mirror.registry.com",
					SubPath:    "dockerhub-mirror",
				}))
			})
		})

		Context("when organization does not exist", func() {
			BeforeEach(func() {
				fakeClient = fake.NewClientBuilder().
					WithScheme(scheme).
					Build()
			})

			It("should return an error", func() {
				config, err := GetRegistryMirrorConfig(ctx, fakeClient, plugin)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("organization test-org not found"))
				Expect(config).To(BeNil())
			})
		})

		Context("when organization has no configmap reference", func() {
			BeforeEach(func() {
				org := &greenhousev1alpha1.Organization{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-org",
					},
					Spec: greenhousev1alpha1.OrganizationSpec{
						ConfigMapRef: "",
					},
				}

				fakeClient = fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(org).
					Build()
			})

			It("should return nil config without error", func() {
				config, err := GetRegistryMirrorConfig(ctx, fakeClient, plugin)

				Expect(err).NotTo(HaveOccurred())
				Expect(config).To(BeNil())
			})
		})

		Context("when referenced configmap does not exist", func() {
			BeforeEach(func() {
				org := &greenhousev1alpha1.Organization{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-org",
					},
					Spec: greenhousev1alpha1.OrganizationSpec{
						ConfigMapRef: "nonexistent-configmap",
					},
				}

				fakeClient = fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(org).
					Build()
			})

			It("should return an error", func() {
				config, err := GetRegistryMirrorConfig(ctx, fakeClient, plugin)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("organization ConfigMap nonexistent-configmap not found in namespace test-org"))
				Expect(config).To(BeNil())
			})
		})

		Context("when configmap exists but has no registry mirror configuration", func() {
			BeforeEach(func() {
				org := &greenhousev1alpha1.Organization{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-org",
					},
					Spec: greenhousev1alpha1.OrganizationSpec{
						ConfigMapRef: "test-configmap",
					},
				}

				configMap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-configmap",
						Namespace: "test-org",
					},
					Data: map[string]string{
						"other-config": "some-value",
					},
				}

				fakeClient = fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(org, configMap).
					Build()
			})

			It("should return nil config without error", func() {
				config, err := GetRegistryMirrorConfig(ctx, fakeClient, plugin)

				Expect(err).NotTo(HaveOccurred())
				Expect(config).To(BeNil())
			})
		})

		Context("when configmap has invalid yaml", func() {
			BeforeEach(func() {
				org := &greenhousev1alpha1.Organization{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-org",
					},
					Spec: greenhousev1alpha1.OrganizationSpec{
						ConfigMapRef: "test-configmap",
					},
				}

				configMap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-configmap",
						Namespace: "test-org",
					},
					Data: map[string]string{
						registryMirrorConfigKey: `invalid: yaml: content: [`,
					},
				}

				fakeClient = fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(org, configMap).
					Build()
			})

			It("should return a parsing error", func() {
				config, err := GetRegistryMirrorConfig(ctx, fakeClient, plugin)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to parse registry mirror configuration"))
				Expect(config).To(BeNil())
			})
		})
	})

	Describe("validateRegistryMirrorConfig", func() {
		DescribeTable("validation scenarios",
			func(config *RegistryMirrorConfig, shouldPass bool, expectedErrorSubstring string) {
				err := validateRegistryMirrorConfig(config)

				if shouldPass {
					Expect(err).NotTo(HaveOccurred())
				} else {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(expectedErrorSubstring))
				}
			},
			Entry("valid configuration with subPath",
				&RegistryMirrorConfig{
					PrimaryMirror: "primary.registry.com",
					RegistryMirrors: map[string]RegistryMirror{
						"ghcr.io": {
							BaseDomain: "europe.registry.com",
							SubPath:    "ghcr-mirror",
						},
					},
				}, true, ""),
			Entry("valid configuration with empty primary mirror",
				&RegistryMirrorConfig{
					PrimaryMirror: "",
					RegistryMirrors: map[string]RegistryMirror{
						"docker.io": {
							BaseDomain: "mirror.registry.com",
							SubPath:    "dockerhub-mirror",
						},
					},
				}, true, ""),
			Entry("empty subPath should fail",
				&RegistryMirrorConfig{
					PrimaryMirror: "primary.registry.com",
					RegistryMirrors: map[string]RegistryMirror{
						"ghcr.io": {
							BaseDomain: "europe.registry.com",
							SubPath:    "",
						},
					},
				}, false, "subPath cannot be empty for registry ghcr.io"),
			Entry("empty registry mirrors map",
				&RegistryMirrorConfig{
					PrimaryMirror:   "primary.registry.com",
					RegistryMirrors: map[string]RegistryMirror{},
				}, false, "registryMirrors cannot be empty"),
			Entry("nil registry mirrors",
				&RegistryMirrorConfig{
					PrimaryMirror:   "primary.registry.com",
					RegistryMirrors: nil,
				}, false, "registryMirrors cannot be empty"),
			Entry("empty basedomain",
				&RegistryMirrorConfig{
					PrimaryMirror: "primary.registry.com",
					RegistryMirrors: map[string]RegistryMirror{
						"ghcr.io": {
							BaseDomain: "",
							SubPath:    "ghcr-mirror",
						},
					},
				}, false, "basedomain cannot be empty for registry ghcr.io"),
		)
	})
})
