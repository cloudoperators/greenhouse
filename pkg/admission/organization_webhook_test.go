// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package admission

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

var _ = Describe("Organization Webhook", func() {
	Context("Validate Organization Defaulting Webhook", func() {
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

	Context("Validate Delete Organization", func() {
		It("should deny deletion of an organization", func() {
			org := &greenhousev1alpha1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-organization",
				},
			}
			_, err := ValidateDeleteOrganization(context.TODO(), nil, org)
			Expect(err).To(HaveOccurred(), "webhook should deny organization deletion")
			Expect(err.Error()).To(Equal("Organization cannot be deleted"), "webhook should deny organization deletion with proper error message")
		})
	})
})
