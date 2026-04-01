// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package teamrbac

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
)

var _ = Describe("resolveTeamRefs", func() {
	It("should return teamRefs when set", func() {
		trb := &greenhousev1alpha2.TeamRoleBinding{
			Spec: greenhousev1alpha2.TeamRoleBindingSpec{
				TeamRefs: []string{"team-a", "team-b"},
			},
		}
		Expect(resolveTeamRefs(trb)).To(Equal([]string{"team-a", "team-b"}))
	})

	It("should fall back to teamRef when teamRefs is empty", func() {
		trb := &greenhousev1alpha2.TeamRoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "pre-migration-trb"},
			Spec: greenhousev1alpha2.TeamRoleBindingSpec{
				TeamRef: "team-legacy",
			},
		}
		Expect(resolveTeamRefs(trb)).To(Equal([]string{"team-legacy"}))
	})

	It("should prefer teamRefs over teamRef when both are set", func() {
		trb := &greenhousev1alpha2.TeamRoleBinding{
			Spec: greenhousev1alpha2.TeamRoleBindingSpec{
				TeamRef:  "team-old",
				TeamRefs: []string{"team-new"},
			},
		}
		Expect(resolveTeamRefs(trb)).To(Equal([]string{"team-new"}))
	})

	It("should return nil when neither is set", func() {
		trb := &greenhousev1alpha2.TeamRoleBinding{
			Spec: greenhousev1alpha2.TeamRoleBindingSpec{},
		}
		Expect(resolveTeamRefs(trb)).To(BeNil())
	})
})
