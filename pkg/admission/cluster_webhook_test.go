// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package admission

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"

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
			&greenhousev1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
				},
				Spec: greenhousev1alpha1.ClusterSpec{
					AccessMode: greenhousev1alpha1.ClusterAccessModeDirect,
				},
			},
			false, false, "",
		),
		Entry("it should add deletion schedule if cluster is marked for deletion",
			&greenhousev1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
					Annotations: map[string]string{
						greenhouseapis.MarkClusterDeletionAnnotation: "true",
					},
				},
				Spec: greenhousev1alpha1.ClusterSpec{
					AccessMode: greenhousev1alpha1.ClusterAccessModeDirect,
				},
			},
			false, true, "",
		),
		Entry("it should remove deletion schedule if cluster is not marked for deletion but schedule exists",
			&greenhousev1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
					Annotations: map[string]string{
						greenhouseapis.ScheduleClusterDeletionAnnotation: now().Format(time.DateTime),
					},
				},
				Spec: greenhousev1alpha1.ClusterSpec{
					AccessMode: greenhousev1alpha1.ClusterAccessModeDirect,
				},
			},
			false, false, "",
		),
		Entry("it should remove deletion schedule if cluster deletion marker has empty value",
			&greenhousev1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
					Annotations: map[string]string{
						greenhouseapis.MarkClusterDeletionAnnotation:     "",
						greenhouseapis.ScheduleClusterDeletionAnnotation: now().Format(time.DateTime),
					},
				},
				Spec: greenhousev1alpha1.ClusterSpec{
					AccessMode: greenhousev1alpha1.ClusterAccessModeDirect,
				},
			},
			false, false, "",
		),
		Entry("it should not reset if deletion marker is not empty and schedule exists",
			&greenhousev1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
					Annotations: map[string]string{
						greenhouseapis.MarkClusterDeletionAnnotation:     "true",
						greenhouseapis.ScheduleClusterDeletionAnnotation: fortyEight(),
					},
				},
				Spec: greenhousev1alpha1.ClusterSpec{
					AccessMode: greenhousev1alpha1.ClusterAccessModeDirect,
				},
			},
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
			&greenhousev1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
				},
				Spec: greenhousev1alpha1.ClusterSpec{
					AccessMode: greenhousev1alpha1.ClusterAccessModeDirect,
				},
			},
			false,
		),
		Entry("it should deny creation of cluster with deletion marker annotation",
			&greenhousev1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
					Annotations: map[string]string{
						greenhouseapis.MarkClusterDeletionAnnotation: "true",
					},
				},
				Spec: greenhousev1alpha1.ClusterSpec{
					AccessMode: greenhousev1alpha1.ClusterAccessModeDirect,
				},
			},
			true,
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
			&greenhousev1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
					Annotations: map[string]string{
						"custom-annotation": "custom-value",
					},
				},
				Spec: greenhousev1alpha1.ClusterSpec{
					AccessMode: greenhousev1alpha1.ClusterAccessModeDirect,
				},
			},
			false,
		),
		Entry("it should allow update with valid deletion schedule",
			&greenhousev1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
					Annotations: map[string]string{
						greenhouseapis.MarkClusterDeletionAnnotation:     "true",
						greenhouseapis.ScheduleClusterDeletionAnnotation: fortyEight(),
					},
				},
				Spec: greenhousev1alpha1.ClusterSpec{
					AccessMode: greenhousev1alpha1.ClusterAccessModeDirect,
				},
			},
			false,
		),
		Entry("it should deny update with invalid deletion schedule",
			&greenhousev1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
					Annotations: map[string]string{
						greenhouseapis.MarkClusterDeletionAnnotation:     "true",
						greenhouseapis.ScheduleClusterDeletionAnnotation: "2006-01-02",
					},
				},
				Spec: greenhousev1alpha1.ClusterSpec{
					AccessMode: greenhousev1alpha1.ClusterAccessModeDirect,
				},
			},
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
		Entry("it should not allow deletion of cluster without deletion annotation",
			&greenhousev1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
				},
				Spec: greenhousev1alpha1.ClusterSpec{
					AccessMode: greenhousev1alpha1.ClusterAccessModeDirect,
				},
			},
			true,
		),
		Entry("it should deny deletion of cluster with deletion marker annotation",
			&greenhousev1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
					Annotations: map[string]string{
						greenhouseapis.MarkClusterDeletionAnnotation: "true",
					},
				},
				Spec: greenhousev1alpha1.ClusterSpec{
					AccessMode: greenhousev1alpha1.ClusterAccessModeDirect,
				},
			},
			true,
		),
	)
})
