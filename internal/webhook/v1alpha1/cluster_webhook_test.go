// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"time"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/test"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var fixedTime = time.Now()

var now = func() time.Time {
	return fixedTime
}

func fortyEight() string {
	return now().Add(48 * time.Hour).Format(time.DateTime)
}

var _ = Describe("Cluster Webhook", Ordered, func() {
	const (
		teamWithSupportGroupName      = "team-support-true"
		teamWithoutSupportGroupName   = "team-no-support"
		teamWithFalseSupportGroupName = "team-support-false"
	)

	var (
		setup                     *test.TestSetup
		teamWithSupportGroupTrue  *greenhousev1alpha1.Team
		teamWithoutSupportGroup   *greenhousev1alpha1.Team
		teamWithSupportGroupFalse *greenhousev1alpha1.Team
	)

	BeforeAll(func() {
		By("creating a new test setup")
		setup = test.NewTestSetup(test.Ctx, test.K8sClient, "cluster-webhook-test")
		By("creating a support-group:true Team")
		teamWithSupportGroupTrue = test.NewTeam(test.Ctx, teamWithSupportGroupName, test.TestNamespace, test.WithSupportGroupLabel("true"))
		Expect(setup.Create(test.Ctx, teamWithSupportGroupTrue)).To(Succeed(), "there should be no error creating the Team")
		By("creating a Team without support-group label")
		teamWithoutSupportGroup = test.NewTeam(test.Ctx, teamWithoutSupportGroupName, test.TestNamespace)
		Expect(setup.Create(test.Ctx, teamWithoutSupportGroup)).To(Succeed(), "there should be no error creating the Team")
		By("creating a Team with support-group:false")
		teamWithSupportGroupFalse = test.NewTeam(test.Ctx, teamWithFalseSupportGroupName, test.TestNamespace, test.WithSupportGroupLabel("false"))
		Expect(setup.Create(test.Ctx, teamWithSupportGroupFalse)).To(Succeed(), "there should be no error creating the Team")
	})
	AfterAll(func() {
		By("deleting the test Teams")
		test.EventuallyDeleted(test.Ctx, test.K8sClient, teamWithSupportGroupTrue)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, teamWithoutSupportGroup)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, teamWithSupportGroupFalse)
	})

	DescribeTable("Defaulting",
		func(cluster *greenhousev1alpha1.Cluster, withError bool, deletionScheduleExists bool, expectedSchedule string) {
			err := DefaultCluster(test.Ctx, nil, cluster)
			if withError {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}

			if deletionScheduleExists {
				Expect(cluster.Annotations).To(HaveKey(greenhouseapis.MarkClusterDeletionAnnotation))
				Expect(cluster.Annotations).To(HaveKey(greenhouseapis.ScheduleClusterDeletionAnnotation))
				if expectedSchedule != "" {
					Expect(cluster.Annotations[greenhouseapis.ScheduleClusterDeletionAnnotation]).To(Equal(expectedSchedule))
				}
			} else {
				Expect(cluster.Annotations).To(BeEmpty())
			}
		},
		Entry("it should not add deletion schedule if cluster is not marked for deletion",
			test.NewCluster(test.Ctx, "test-cluster", "test-namespace"),
			false, false, "",
		),
		Entry("it should add deletion schedule if cluster is marked for deletion",
			test.NewCluster(test.Ctx, "test-cluster", "test-namespace",
				test.WithClusterAnnotations(map[string]string{
					greenhouseapis.MarkClusterDeletionAnnotation: "true",
				}),
			),
			false, true, "",
		),
		Entry("it should remove deletion schedule if cluster is not marked for deletion but schedule exists",
			test.NewCluster(test.Ctx, "test-cluster", "test-namespace",
				test.WithClusterAnnotations(map[string]string{
					greenhouseapis.ScheduleClusterDeletionAnnotation: now().Format(time.DateTime),
				}),
			),
			false, false, "",
		),
		Entry("it should remove deletion schedule if cluster deletion marker has empty value",
			test.NewCluster(test.Ctx, "test-cluster", "test-namespace",
				test.WithClusterAnnotations(map[string]string{
					greenhouseapis.MarkClusterDeletionAnnotation:     "",
					greenhouseapis.ScheduleClusterDeletionAnnotation: now().Format(time.DateTime),
				}),
			),
			false, false, "",
		),
		Entry("it should not reset if deletion marker is not empty and schedule exists",
			test.NewCluster(test.Ctx, "test-cluster", "test-namespace",
				test.WithClusterAnnotations(map[string]string{
					greenhouseapis.MarkClusterDeletionAnnotation:     "true",
					greenhouseapis.ScheduleClusterDeletionAnnotation: fortyEight(),
				}),
			),
			false, true, fortyEight(),
		),
	)

	DescribeTable("Validate Create Cluster",
		func(cluster *greenhousev1alpha1.Cluster, withError bool) {
			err := DefaultCluster(test.Ctx, nil, cluster)
			Expect(err).NotTo(HaveOccurred())

			warnings, err := ValidateCreateCluster(test.Ctx, setup.Client, cluster)
			if withError {
				Expect(err).To(HaveOccurred())
				Expect(warnings).NotTo(BeEmpty())
			} else {
				Expect(err).NotTo(HaveOccurred())
				Expect(warnings).To(BeEmpty())
			}
		},
		Entry("it should allow creation of cluster without deletion annotation",
			test.NewCluster(test.Ctx, "test-cluster", test.TestNamespace,
				test.WithLabel(greenhouseapis.LabelKeyOwnedBy, teamWithSupportGroupName),
			),
			false,
		),
		Entry("it should deny creation of cluster with deletion marker annotation",
			test.NewCluster(test.Ctx, "test-cluster", test.TestNamespace,
				test.WithClusterAnnotations(map[string]string{
					greenhouseapis.MarkClusterDeletionAnnotation: "true",
				}),
				test.WithLabel(greenhouseapis.LabelKeyOwnedBy, teamWithSupportGroupName),
			),
			true,
		),
		Entry("it should allow creation of cluster with not too long token validity",
			test.NewCluster(test.Ctx, "test-cluster", test.TestNamespace,
				test.WithMaxTokenValidity(72),
				test.WithLabel(greenhouseapis.LabelKeyOwnedBy, teamWithSupportGroupName),
			),
			false,
		),
		Entry("it should deny creation of cluster without owned-by label",
			test.NewCluster(test.Ctx, "test-cluster", test.TestNamespace),
			true,
		),
		Entry("it should deny creation of cluster with owned-by label pointing to a non-existent Team",
			test.NewCluster(test.Ctx, "test-cluster", test.TestNamespace,
				test.WithLabel(greenhouseapis.LabelKeyOwnedBy, "invalid-team"),
			),
			true,
		),
		Entry("it should deny creation of cluster with owned-by label pointing to a Team without support-group label",
			test.NewCluster(test.Ctx, "test-cluster", test.TestNamespace,
				test.WithLabel(greenhouseapis.LabelKeyOwnedBy, teamWithoutSupportGroupName),
			),
			true,
		),
		Entry("it should deny creation of cluster with owned-by label pointing to a Team with support-group:false label",
			test.NewCluster(test.Ctx, "test-cluster", test.TestNamespace,
				test.WithLabel(greenhouseapis.LabelKeyOwnedBy, teamWithFalseSupportGroupName),
			),
			true,
		),
	)

	DescribeTable("Validate Update Cluster",
		func(cluster *greenhousev1alpha1.Cluster, withError bool) {
			_, err := ValidateUpdateCluster(test.Ctx, setup.Client, nil, cluster)
			if withError {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		},
		Entry("it should allow update without deletion markers",
			test.NewCluster(test.Ctx, "test-cluster", test.TestNamespace,
				test.WithClusterAnnotations(map[string]string{
					"custom-annotation": "custom-value",
				}),
				test.WithLabel(greenhouseapis.LabelKeyOwnedBy, teamWithSupportGroupName),
			),
			false,
		),
		Entry("it should allow update with valid deletion schedule",
			test.NewCluster(test.Ctx, "test-cluster", test.TestNamespace,
				test.WithClusterAnnotations(map[string]string{
					greenhouseapis.MarkClusterDeletionAnnotation:     "true",
					greenhouseapis.ScheduleClusterDeletionAnnotation: fortyEight(),
				}),
				test.WithLabel(greenhouseapis.LabelKeyOwnedBy, teamWithSupportGroupName),
			),
			false,
		),
		Entry("it should deny update with invalid deletion schedule",
			test.NewCluster(test.Ctx, "test-cluster", test.TestNamespace,
				test.WithClusterAnnotations(map[string]string{
					greenhouseapis.MarkClusterDeletionAnnotation:     "true",
					greenhouseapis.ScheduleClusterDeletionAnnotation: time.DateOnly,
				}),
				test.WithLabel(greenhouseapis.LabelKeyOwnedBy, teamWithSupportGroupName),
			),
			true,
		),
	)

	DescribeTable("Validate Delete Cluster",
		func(cluster *greenhousev1alpha1.Cluster, withError bool) {
			err := DefaultCluster(test.Ctx, nil, cluster)
			Expect(err).NotTo(HaveOccurred())

			warnings, err := ValidateDeleteCluster(test.Ctx, nil, cluster)
			if withError {
				Expect(err).To(HaveOccurred())
				Expect(warnings).NotTo(BeEmpty())
			} else {
				Expect(err).NotTo(HaveOccurred())
				Expect(warnings).To(BeEmpty())
			}
		},
		Entry("it should deny deletion of cluster without deletion annotation",
			test.NewCluster(test.Ctx, "test-cluster", "test-namespace"),
			true,
		),
		Entry("it should deny deletion of cluster with deletion marker annotation",
			test.NewCluster(test.Ctx, "test-cluster", "test-namespace",
				test.WithClusterAnnotations(map[string]string{greenhouseapis.MarkClusterDeletionAnnotation: "true"}),
			),
			true,
		),
	)
})
