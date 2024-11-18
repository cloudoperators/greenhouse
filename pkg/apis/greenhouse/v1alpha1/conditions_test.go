// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

var _ = Describe("Test conditions util functions", func() {

	var (
		timeNow = metav1.NewTime(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
	)

	DescribeTable("should correctly identify conditions",
		func(condition1 greenhousev1alpha1.Condition, condition2 greenhousev1alpha1.Condition, expected bool) {
			Expect(condition1.Equal(condition2)).To(Equal(expected))
		},
		Entry("should correctly identify equal conditions", greenhousev1alpha1.Condition{
			Type:               greenhousev1alpha1.ReadyCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: timeNow,
			Message:            "test",
		}, greenhousev1alpha1.Condition{
			Type:               greenhousev1alpha1.ReadyCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: timeNow,
			Message:            "test",
		}, true),
		Entry("should correctly identify conditions differing in the message", greenhousev1alpha1.Condition{
			Type:               greenhousev1alpha1.ReadyCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: timeNow,
			Message:            "test",
		}, greenhousev1alpha1.Condition{
			Type:               greenhousev1alpha1.ReadyCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: timeNow,
			Message:            "test2",
		}, false),
		Entry("should correctly identify conditions differing in the status", greenhousev1alpha1.Condition{
			Type:               greenhousev1alpha1.ReadyCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: timeNow,
			Message:            "test",
		}, greenhousev1alpha1.Condition{
			Type:               greenhousev1alpha1.ReadyCondition,
			Status:             metav1.ConditionFalse,
			LastTransitionTime: timeNow,
			Message:            "test",
		}, false),
		Entry("should correctly identify conditions differing in the type", greenhousev1alpha1.Condition{
			Type:               greenhousev1alpha1.ReadyCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: timeNow,
			Message:            "test",
		}, greenhousev1alpha1.Condition{
			Type:               greenhousev1alpha1.ReadyCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: timeNow,
			Message:            "test",
		}, true),
		Entry("should correctly ingore differing in the last transition time", greenhousev1alpha1.Condition{
			Type:               greenhousev1alpha1.ReadyCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: timeNow,
			Message:            "test",
		}, greenhousev1alpha1.Condition{
			Type:               greenhousev1alpha1.ReadyCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.NewTime(metav1.Now().AddDate(0, 0, -1)),
			Message:            "test",
		}, true),
	)

	DescribeTable("should correctly get the condition Status",
		func(condition greenhousev1alpha1.Condition, expected bool) {
			Expect(condition.IsTrue()).To(Equal(expected))
		},
		Entry("should correctly identify a true condition", greenhousev1alpha1.Condition{
			Type:               greenhousev1alpha1.ReadyCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: timeNow,
			Message:            "test",
		}, true),
		Entry("should correctly identify a false condition", greenhousev1alpha1.Condition{
			Type:               greenhousev1alpha1.ReadyCondition,
			Status:             metav1.ConditionFalse,
			LastTransitionTime: timeNow,
			Message:            "test",
		}, false),
	)

	DescribeTable("should correctly calculate the Ready condition",
		func(condition greenhousev1alpha1.Condition, expected bool) {
			statusConditions := greenhousev1alpha1.StatusConditions{
				Conditions: []greenhousev1alpha1.Condition{condition},
			}
			Expect(statusConditions.IsReadyTrue()).To(Equal(expected))
		},
		Entry("should return true if Ready condition is true", greenhousev1alpha1.Condition{
			Type:               greenhousev1alpha1.ReadyCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: timeNow,
			Message:            "test",
		}, true),
		Entry("should return false if Ready condition is false", greenhousev1alpha1.Condition{
			Type:               greenhousev1alpha1.ReadyCondition,
			Status:             metav1.ConditionFalse,
			LastTransitionTime: timeNow,
			Message:            "test",
		}, false),
		Entry("should return false if no conditions are set", nil, false),
	)

	DescribeTable("should correctly use SetCondition on StatusConditions",
		func(
			initialStatusConditions greenhousev1alpha1.StatusConditions,
			expected greenhousev1alpha1.StatusConditions,
			conditions ...greenhousev1alpha1.Condition,

		) {
			initialStatusConditions.SetConditions(conditions...)
			Expect(initialStatusConditions).To(Equal(expected))
		},
		Entry(
			"should correctly add a condition to empty StatusConditions",
			greenhousev1alpha1.StatusConditions{},
			greenhousev1alpha1.StatusConditions{
				Conditions: []greenhousev1alpha1.Condition{
					{
						Type:               greenhousev1alpha1.ReadyCondition,
						Status:             metav1.ConditionTrue,
						LastTransitionTime: timeNow,
						Message:            "test",
					},
				},
			},
			greenhousev1alpha1.Condition{
				Type:               greenhousev1alpha1.ReadyCondition,
				Status:             metav1.ConditionTrue,
				Message:            "test",
				LastTransitionTime: timeNow,
			}),
		Entry(
			"should correctly add a condition to StatusConditions with an existing condition",
			greenhousev1alpha1.StatusConditions{
				Conditions: []greenhousev1alpha1.Condition{
					{
						Type:               greenhousev1alpha1.KubeConfigValid,
						Status:             metav1.ConditionFalse,
						LastTransitionTime: timeNow,
						Message:            "test",
					},
				},
			},
			greenhousev1alpha1.StatusConditions{
				Conditions: []greenhousev1alpha1.Condition{
					{
						Type:               greenhousev1alpha1.KubeConfigValid,
						Status:             metav1.ConditionFalse,
						LastTransitionTime: timeNow,
						Message:            "test",
					},
					{
						Type:               greenhousev1alpha1.ReadyCondition,
						Status:             metav1.ConditionTrue,
						LastTransitionTime: timeNow,
						Message:            "test",
					},
				},
			},
			greenhousev1alpha1.Condition{
				Type:               greenhousev1alpha1.ReadyCondition,
				Status:             metav1.ConditionTrue,
				LastTransitionTime: timeNow,
				Message:            "test",
			}),
		Entry(
			"should correctly update a condition with matching Type in StatusConditions with a different condition",
			greenhousev1alpha1.StatusConditions{
				Conditions: []greenhousev1alpha1.Condition{
					{
						Type:               greenhousev1alpha1.KubeConfigValid,
						Status:             metav1.ConditionFalse,
						LastTransitionTime: timeNow,
						Message:            "test",
					},
					{
						Type:               greenhousev1alpha1.ReadyCondition,
						Status:             metav1.ConditionFalse,
						LastTransitionTime: timeNow,
						Message:            "test",
					},
				},
			},
			greenhousev1alpha1.StatusConditions{
				Conditions: []greenhousev1alpha1.Condition{
					{
						Type:               greenhousev1alpha1.KubeConfigValid,
						Status:             metav1.ConditionFalse,
						LastTransitionTime: timeNow,
						Message:            "test",
					},
					{
						Type:               greenhousev1alpha1.ReadyCondition,
						Status:             metav1.ConditionTrue,
						LastTransitionTime: timeNow,
						Message:            "test2",
					},
				},
			},
			greenhousev1alpha1.Condition{
				Type:               greenhousev1alpha1.ReadyCondition,
				Status:             metav1.ConditionTrue,
				LastTransitionTime: timeNow,
				Message:            "test2",
			}),
		Entry(
			"should ignore updating a condition with matching Type but differing LastTransitionTime in StatusConditions with a different condition",
			greenhousev1alpha1.StatusConditions{
				Conditions: []greenhousev1alpha1.Condition{
					{
						Type:               greenhousev1alpha1.KubeConfigValid,
						Status:             metav1.ConditionFalse,
						LastTransitionTime: timeNow,
						Message:            "test",
					},
					{
						Type:               greenhousev1alpha1.ReadyCondition,
						Status:             metav1.ConditionFalse,
						LastTransitionTime: timeNow,
						Message:            "test",
					},
				},
			},
			greenhousev1alpha1.StatusConditions{
				Conditions: []greenhousev1alpha1.Condition{
					{
						Type:               greenhousev1alpha1.KubeConfigValid,
						Status:             metav1.ConditionFalse,
						LastTransitionTime: timeNow,
						Message:            "test",
					},
					{
						Type:               greenhousev1alpha1.ReadyCondition,
						Status:             metav1.ConditionFalse,
						LastTransitionTime: timeNow,
						Message:            "test",
					},
				},
			},
			greenhousev1alpha1.Condition{
				Type:               greenhousev1alpha1.ReadyCondition,
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.NewTime(metav1.Now().AddDate(0, 0, -1)),
				Message:            "test",
			}),
		Entry(
			"should not update a conditions LastTransitionTime if only the message changes",
			greenhousev1alpha1.StatusConditions{
				Conditions: []greenhousev1alpha1.Condition{
					{
						Type:               greenhousev1alpha1.KubeConfigValid,
						Status:             metav1.ConditionFalse,
						LastTransitionTime: timeNow,
						Message:            "test",
					},
				},
			},
			greenhousev1alpha1.StatusConditions{
				Conditions: []greenhousev1alpha1.Condition{
					{
						Type:               greenhousev1alpha1.KubeConfigValid,
						Status:             metav1.ConditionFalse,
						LastTransitionTime: timeNow,
						Message:            "test2",
					},
				},
			},
			greenhousev1alpha1.Condition{
				Type:               greenhousev1alpha1.KubeConfigValid,
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.NewTime(metav1.Now().AddDate(0, 0, -1)),
				Message:            "test2",
			},
		),
		Entry(
			"should set and update multiple conditions",
			greenhousev1alpha1.StatusConditions{
				Conditions: []greenhousev1alpha1.Condition{
					{
						Type:               greenhousev1alpha1.KubeConfigValid,
						Status:             metav1.ConditionFalse,
						LastTransitionTime: timeNow,
						Message:            "test",
					},
				},
			},
			greenhousev1alpha1.StatusConditions{
				Conditions: []greenhousev1alpha1.Condition{
					{
						Type:               greenhousev1alpha1.KubeConfigValid,
						Status:             metav1.ConditionFalse,
						LastTransitionTime: timeNow,
						Message:            "test2",
					},
					{
						Type:               greenhousev1alpha1.ReadyCondition,
						Status:             metav1.ConditionTrue,
						LastTransitionTime: timeNow,
						Message:            "test",
					},
				},
			},
			greenhousev1alpha1.Condition{

				Type:               greenhousev1alpha1.KubeConfigValid,
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.NewTime(metav1.Now().AddDate(0, 0, -1)),
				Message:            "test2",
			},
			greenhousev1alpha1.Condition{
				Type:               greenhousev1alpha1.ReadyCondition,
				Status:             metav1.ConditionTrue,
				LastTransitionTime: timeNow,
				Message:            "test",
			},
		),
	)

	It("should correctly identify equal conditions", func() {
		By("identifying equal conditions")
		condition1 := greenhousev1alpha1.Condition{
			Type:               greenhousev1alpha1.ReadyCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: timeNow,
			Message:            "test",
		}
		condition2 := greenhousev1alpha1.Condition{
			Type:               greenhousev1alpha1.ReadyCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.NewTime(metav1.Now().AddDate(0, 0, -1)),
			Message:            "test",
		}
		Expect(condition1.Equal(condition2)).To(BeTrue())

	})


    It("should remove the deprecated NoHelmChartTestFailures condition", func() {
        By("removing the deprecated NoHelmChartTestFailures condition")
        noHelmChartTestFailuresCondition := greenhousev1alpha1.Condition{
            Type:               greenhousev1alpha1.NoHelmChartTestFailuresCondition,
            Status:             metav1.ConditionTrue,
            LastTransitionTime: timeNow,
            Message:            "test",
        }

		existingCondition1 := greenhousev1alpha1.Condition{
            Type:               greenhousev1alpha1.HelmReconcileFailedCondition,
            Status:             metav1.ConditionFalse,
            LastTransitionTime: timeNow,
            Message:            "test",			
        }

		existingCondition2 := greenhousev1alpha1.Condition{
            Type:               greenhousev1alpha1.HelmDriftDetectedCondition,
            Status:             metav1.ConditionTrue,
            LastTransitionTime: timeNow,
            Message:            "test",			
        }
		
        statusConditions := greenhousev1alpha1.StatusConditions{
            Conditions: []greenhousev1alpha1.Condition{noHelmChartTestFailuresCondition, existingCondition1, existingCondition2},
        }

		newCondition1 := greenhousev1alpha1.Condition{
            Type:               greenhousev1alpha1.HelmChartTestSucceededCondition,
            Status:             metav1.ConditionTrue,
            LastTransitionTime: timeNow,
            Message:            "test",			
        }

		newCondition2 := greenhousev1alpha1.Condition{
			Type:               greenhousev1alpha1.ReadyCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.NewTime(metav1.Now().AddDate(0, 0, -1)),
			Message:            "test",
		}

        // Set new condition
        statusConditions.SetConditions(newCondition1)
        statusConditions.SetConditions(newCondition2)

        // Check if the deprecated condition is removed
        Expect(statusConditions.GetConditionByType(greenhousev1alpha1.NoHelmChartTestFailuresCondition)).To(BeNil())

        // Check if the new condition is added
        condition := statusConditions.GetConditionByType(greenhousev1alpha1.HelmChartTestSucceededCondition)
        Expect(condition).NotTo(BeNil())
        Expect(condition.Status).To(Equal(metav1.ConditionTrue))
        Expect(condition.LastTransitionTime).To(Equal(timeNow))
    })
})
