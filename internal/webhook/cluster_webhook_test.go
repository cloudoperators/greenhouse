// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package admission

import (
	"context"
	"time"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/test/mocks"

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

var _ = Describe("Cluster Webhook", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	DescribeTable("Defaulting",
		func(cluster *greenhousev1alpha1.Cluster, withError bool, deletionScheduleExists bool, expectedSchedule string) {
			err := DefaultCluster(ctx, nil, cluster)
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
			mocks.NewCluster("test-cluster", "test-namespace"),
			false, false, "",
		),
		Entry("it should add deletion schedule if cluster is marked for deletion",
			mocks.NewCluster("test-cluster", "test-namespace",
				mocks.WithClusterAnnotations(map[string]string{
					greenhouseapis.MarkClusterDeletionAnnotation: "true",
				}),
			),
			false, true, "",
		),
		Entry("it should remove deletion schedule if cluster is not marked for deletion but schedule exists",
			mocks.NewCluster("test-cluster", "test-namespace",
				mocks.WithClusterAnnotations(map[string]string{
					greenhouseapis.ScheduleClusterDeletionAnnotation: now().Format(time.DateTime),
				}),
			),
			false, false, "",
		),
		Entry("it should remove deletion schedule if cluster deletion marker has empty value",
			mocks.NewCluster("test-cluster", "test-namespace",
				mocks.WithClusterAnnotations(map[string]string{
					greenhouseapis.MarkClusterDeletionAnnotation:     "",
					greenhouseapis.ScheduleClusterDeletionAnnotation: now().Format(time.DateTime),
				}),
			),
			false, false, "",
		),
		Entry("it should not reset if deletion marker is not empty and schedule exists",
			mocks.NewCluster("test-cluster", "test-namespace",
				mocks.WithClusterAnnotations(map[string]string{
					greenhouseapis.MarkClusterDeletionAnnotation:     "true",
					greenhouseapis.ScheduleClusterDeletionAnnotation: fortyEight(),
				}),
			),
			false, true, fortyEight(),
		),
	)

	DescribeTable("Validate Create Cluster",
		func(cluster *greenhousev1alpha1.Cluster, withError bool) {
			err := DefaultCluster(ctx, nil, cluster)
			Expect(err).NotTo(HaveOccurred())

			warnings, err := ValidateCreateCluster(ctx, nil, cluster)
			if withError {
				Expect(err).To(HaveOccurred())
				Expect(warnings).NotTo(BeEmpty())
			} else {
				Expect(err).NotTo(HaveOccurred())
				Expect(warnings).To(BeEmpty())
			}
		},
		Entry("it should allow creation of cluster without deletion annotation",
			mocks.NewCluster("test-cluster", "test-namespace"),
			false,
		),
		Entry("it should deny creation of cluster with deletion marker annotation",
			mocks.NewCluster("test-cluster", "test-namespace",
				mocks.WithClusterAnnotations(map[string]string{
					greenhouseapis.MarkClusterDeletionAnnotation: "true",
				}),
			),
			true,
		),
		Entry("it should allow creation of cluster with not too long token validity",
			mocks.NewCluster("test-cluster", "test-namespace",
				mocks.WithMaxTokenValidity(72),
			),
			false,
		),
	)

	DescribeTable("Validate Update Cluster",
		func(cluster *greenhousev1alpha1.Cluster, withError bool) {
			_, err := ValidateUpdateCluster(ctx, nil, nil, cluster)
			if withError {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		},
		Entry("it should allow update without deletion markers",
			mocks.NewCluster("test-cluster", "test-namespace",
				mocks.WithClusterAnnotations(map[string]string{
					"custom-annotation": "custom-value",
				}),
			),
			false,
		),
		Entry("it should allow update with valid deletion schedule",
			mocks.NewCluster("test-cluster", "test-namespace",
				mocks.WithClusterAnnotations(map[string]string{
					greenhouseapis.MarkClusterDeletionAnnotation:     "true",
					greenhouseapis.ScheduleClusterDeletionAnnotation: fortyEight(),
				}),
			),
			false,
		),
		Entry("it should deny update with invalid deletion schedule",
			mocks.NewCluster("test-cluster", "test-namespace",
				mocks.WithClusterAnnotations(map[string]string{
					greenhouseapis.MarkClusterDeletionAnnotation:     "true",
					greenhouseapis.ScheduleClusterDeletionAnnotation: time.DateOnly,
				}),
			),
			true,
		),
	)

	DescribeTable("Validate Delete Cluster",
		func(cluster *greenhousev1alpha1.Cluster, withError bool) {
			err := DefaultCluster(ctx, nil, cluster)
			Expect(err).NotTo(HaveOccurred())

			warnings, err := ValidateDeleteCluster(ctx, nil, cluster)
			if withError {
				Expect(err).To(HaveOccurred())
				Expect(warnings).NotTo(BeEmpty())
			} else {
				Expect(err).NotTo(HaveOccurred())
				Expect(warnings).To(BeEmpty())
			}
		},
		Entry("it should deny deletion of cluster without deletion annotation",
			mocks.NewCluster("test-cluster", "test-namespace"),
			true,
		),
		Entry("it should deny deletion of cluster with deletion marker annotation",
			mocks.NewCluster("test-cluster", "test-namespace",
				mocks.WithClusterAnnotations(map[string]string{greenhouseapis.MarkClusterDeletionAnnotation: "true"}),
			),
			true,
		),
	)
})
