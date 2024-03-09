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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
