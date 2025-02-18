// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package admission

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

var _ = Describe("Validate Organization Defaulting Webhook", func() {
	It("Should default the display name of the organization", func() {
		orgWithoutDisplayName := &greenhousev1alpha1.Organization{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-organization",
			},
			Spec: greenhousev1alpha1.OrganizationSpec{},
		}

		Expect(DefaultOrganization(context.TODO(), nil, orgWithoutDisplayName)).
			To(Succeed(), "there should be no error applying defaults to the organization")
		Expect(orgWithoutDisplayName.Spec.DisplayName).
			ToNot(BeEmpty(), "the spec.displayName should be defaulted and not be empty")
		Expect(orgWithoutDisplayName.Spec.DisplayName).
			To(Equal("test organization"), "the spec.display should be defaulted and match")
	})
})

var _ = Describe("Validate Organization Webhook", func() {
	DescribeTable("Create Organization Webhook", func(obj runtime.Object, expectedError bool) {
		warnings, err := ValidateCreateOrganization(context.Background(), nil, obj)

		Expect(warnings).To(BeEmpty())
		if expectedError {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).NotTo(HaveOccurred())
		}
	},
		Entry("with incorrect type of object", &greenhousev1alpha1.Team{}, false),
		Entry("without mapped admin group", &greenhousev1alpha1.Organization{}, true),
		Entry("with mapped admin group", &greenhousev1alpha1.Organization{
			Spec: greenhousev1alpha1.OrganizationSpec{MappedOrgAdminIDPGroup: "MAPPER_ADMIN_ID_GROUP"},
		}, false),
		Entry("with basic auth configured", &greenhousev1alpha1.Organization{
			Spec: greenhousev1alpha1.OrganizationSpec{
				MappedOrgAdminIDPGroup: "MAPPER_ADMIN_ID_GROUP",
				Authentication: &greenhousev1alpha1.Authentication{
					SCIMConfig: &greenhousev1alpha1.SCIMConfig{
						BaseURL: "https://example.org",
						BasicAuthUser: greenhousev1alpha1.ValueFromSource{
							Secret: &greenhousev1alpha1.SecretKeyReference{
								Name: "test-secret",
								Key:  "test-user",
							},
						},
						BasicAuthPw: greenhousev1alpha1.ValueFromSource{
							Secret: &greenhousev1alpha1.SecretKeyReference{
								Name: "test-secret",
								Key:  "test-password",
							},
						},
					},
				},
			},
		}, false),
		Entry("with bearer token auth configured", &greenhousev1alpha1.Organization{
			Spec: greenhousev1alpha1.OrganizationSpec{
				MappedOrgAdminIDPGroup: "MAPPER_ADMIN_ID_GROUP",
				Authentication: &greenhousev1alpha1.Authentication{
					SCIMConfig: &greenhousev1alpha1.SCIMConfig{
						BaseURL: "https://example.org",
						BearerToken: greenhousev1alpha1.ValueFromSource{
							Secret: &greenhousev1alpha1.SecretKeyReference{
								Name: "test-secret",
								Key:  "test-bearer-token",
							},
						},
					},
				},
			},
		}, false),
		Entry("with bearer token auth and basic auth configured", &greenhousev1alpha1.Organization{
			Spec: greenhousev1alpha1.OrganizationSpec{
				MappedOrgAdminIDPGroup: "MAPPER_ADMIN_ID_GROUP",
				Authentication: &greenhousev1alpha1.Authentication{
					SCIMConfig: &greenhousev1alpha1.SCIMConfig{
						BaseURL: "https://example.org",
						BearerToken: greenhousev1alpha1.ValueFromSource{
							Secret: &greenhousev1alpha1.SecretKeyReference{
								Name: "test-secret",
								Key:  "test-bearer-token",
							},
						},
						BasicAuthUser: greenhousev1alpha1.ValueFromSource{
							Secret: &greenhousev1alpha1.SecretKeyReference{
								Name: "test-secret",
								Key:  "test-user",
							},
						},
						BasicAuthPw: greenhousev1alpha1.ValueFromSource{
							Secret: &greenhousev1alpha1.SecretKeyReference{
								Name: "test-secret",
								Key:  "test-password",
							},
						},
					},
				},
			},
		}, true),
	)

	DescribeTable("Update Organization Webhook", func(obj runtime.Object, expectedError bool) {
		warnings, err := ValidateUpdateOrganization(context.Background(), nil, nil, obj)

		Expect(warnings).To(BeEmpty())
		if expectedError {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).NotTo(HaveOccurred())
		}
	},
		Entry("with incorrect type of object", &greenhousev1alpha1.Team{}, false),
		Entry("without mapped admin group", &greenhousev1alpha1.Organization{}, true),
		Entry("with mapped admin group", &greenhousev1alpha1.Organization{
			Spec: greenhousev1alpha1.OrganizationSpec{MappedOrgAdminIDPGroup: "MAPPER_ADMIN_ID_GROUP"},
		}, false),
		Entry("with basic auth configured", &greenhousev1alpha1.Organization{
			Spec: greenhousev1alpha1.OrganizationSpec{
				MappedOrgAdminIDPGroup: "MAPPER_ADMIN_ID_GROUP",
				Authentication: &greenhousev1alpha1.Authentication{
					SCIMConfig: &greenhousev1alpha1.SCIMConfig{
						BaseURL: "https://example.org",
						BasicAuthUser: greenhousev1alpha1.ValueFromSource{
							Secret: &greenhousev1alpha1.SecretKeyReference{
								Name: "test-secret",
								Key:  "test-user",
							},
						},
						BasicAuthPw: greenhousev1alpha1.ValueFromSource{
							Secret: &greenhousev1alpha1.SecretKeyReference{
								Name: "test-secret",
								Key:  "test-password",
							},
						},
					},
				},
			},
		}, false),
		Entry("with bearer token auth configured", &greenhousev1alpha1.Organization{
			Spec: greenhousev1alpha1.OrganizationSpec{
				MappedOrgAdminIDPGroup: "MAPPER_ADMIN_ID_GROUP",
				Authentication: &greenhousev1alpha1.Authentication{
					SCIMConfig: &greenhousev1alpha1.SCIMConfig{
						BaseURL: "https://example.org",
						BearerToken: greenhousev1alpha1.ValueFromSource{
							Secret: &greenhousev1alpha1.SecretKeyReference{
								Name: "test-secret",
								Key:  "test-bearer-token",
							},
						},
					},
				},
			},
		}, false),
		Entry("with bearer token auth and basic auth configured", &greenhousev1alpha1.Organization{
			Spec: greenhousev1alpha1.OrganizationSpec{
				MappedOrgAdminIDPGroup: "MAPPER_ADMIN_ID_GROUP",
				Authentication: &greenhousev1alpha1.Authentication{
					SCIMConfig: &greenhousev1alpha1.SCIMConfig{
						BaseURL: "https://example.org",
						BearerToken: greenhousev1alpha1.ValueFromSource{
							Secret: &greenhousev1alpha1.SecretKeyReference{
								Name: "test-secret",
								Key:  "test-bearer-token",
							},
						},
						BasicAuthUser: greenhousev1alpha1.ValueFromSource{
							Secret: &greenhousev1alpha1.SecretKeyReference{
								Name: "test-secret",
								Key:  "test-user",
							},
						},
						BasicAuthPw: greenhousev1alpha1.ValueFromSource{
							Secret: &greenhousev1alpha1.SecretKeyReference{
								Name: "test-secret",
								Key:  "test-password",
							},
						},
					},
				},
			},
		}, true),
	)
})
