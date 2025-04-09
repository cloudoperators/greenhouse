// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
)

var _ = Describe("Test conditions util functions", func() {

	var (
		timeNow = metav1.NewTime(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
	)

	DescribeTable("should correctly identify conditions",
		func(condition1 greenhouseapis.Condition, condition2 greenhouseapis.Condition, expected bool) {
			Expect(condition1.Equal(condition2)).To(Equal(expected))
		},
		Entry("should correctly identify equal conditions", greenhouseapis.Condition{
			Type:               greenhouseapis.ReadyCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: timeNow,
			Message:            "test",
		}, greenhouseapis.Condition{
			Type:               greenhouseapis.ReadyCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: timeNow,
			Message:            "test",
		}, true),
		Entry("should correctly identify conditions differing in the message", greenhouseapis.Condition{
			Type:               greenhouseapis.ReadyCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: timeNow,
			Message:            "test",
		}, greenhouseapis.Condition{
			Type:               greenhouseapis.ReadyCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: timeNow,
			Message:            "test2",
		}, false),
		Entry("should correctly identify conditions differing in the status", greenhouseapis.Condition{
			Type:               greenhouseapis.ReadyCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: timeNow,
			Message:            "test",
		}, greenhouseapis.Condition{
			Type:               greenhouseapis.ReadyCondition,
			Status:             metav1.ConditionFalse,
			LastTransitionTime: timeNow,
			Message:            "test",
		}, false),
		Entry("should correctly identify conditions differing in the type", greenhouseapis.Condition{
			Type:               greenhouseapis.ReadyCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: timeNow,
			Message:            "test",
		}, greenhouseapis.Condition{
			Type:               greenhouseapis.ReadyCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: timeNow,
			Message:            "test",
		}, true),
		Entry("should correctly ingore differing in the last transition time", greenhouseapis.Condition{
			Type:               greenhouseapis.ReadyCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: timeNow,
			Message:            "test",
		}, greenhouseapis.Condition{
			Type:               greenhouseapis.ReadyCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.NewTime(metav1.Now().AddDate(0, 0, -1)),
			Message:            "test",
		}, true),
	)

	DescribeTable("should correctly get the condition Status",
		func(condition greenhouseapis.Condition, expected bool) {
			Expect(condition.IsTrue()).To(Equal(expected))
		},
		Entry("should correctly identify a true condition", greenhouseapis.Condition{
			Type:               greenhouseapis.ReadyCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: timeNow,
			Message:            "test",
		}, true),
		Entry("should correctly identify a false condition", greenhouseapis.Condition{
			Type:               greenhouseapis.ReadyCondition,
			Status:             metav1.ConditionFalse,
			LastTransitionTime: timeNow,
			Message:            "test",
		}, false),
	)

	DescribeTable("should correctly calculate the Ready condition",
		func(condition greenhouseapis.Condition, expected bool) {
			statusConditions := greenhouseapis.StatusConditions{
				Conditions: []greenhouseapis.Condition{condition},
			}
			Expect(statusConditions.IsReadyTrue()).To(Equal(expected))
		},
		Entry("should return true if Ready condition is true", greenhouseapis.Condition{
			Type:               greenhouseapis.ReadyCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: timeNow,
			Message:            "test",
		}, true),
		Entry("should return false if Ready condition is false", greenhouseapis.Condition{
			Type:               greenhouseapis.ReadyCondition,
			Status:             metav1.ConditionFalse,
			LastTransitionTime: timeNow,
			Message:            "test",
		}, false),
		Entry("should return false if no conditions are set", nil, false),
	)

	DescribeTable("should correctly use SetCondition on StatusConditions",
		func(
			initialStatusConditions greenhouseapis.StatusConditions,
			expected greenhouseapis.StatusConditions,
			conditions ...greenhouseapis.Condition,

		) {
			initialStatusConditions.SetConditions(conditions...)
			Expect(initialStatusConditions).To(Equal(expected))
		},
		Entry(
			"should correctly add a condition to empty StatusConditions",
			greenhouseapis.StatusConditions{},
			greenhouseapis.StatusConditions{
				Conditions: []greenhouseapis.Condition{
					{
						Type:               greenhouseapis.ReadyCondition,
						Status:             metav1.ConditionTrue,
						LastTransitionTime: timeNow,
						Message:            "test",
					},
				},
			},
			greenhouseapis.Condition{
				Type:               greenhouseapis.ReadyCondition,
				Status:             metav1.ConditionTrue,
				Message:            "test",
				LastTransitionTime: timeNow,
			}),
		Entry(
			"should correctly add a condition to StatusConditions with an existing condition",
			greenhouseapis.StatusConditions{
				Conditions: []greenhouseapis.Condition{
					{
						Type:               greenhouseapis.RBACReady,
						Status:             metav1.ConditionFalse,
						LastTransitionTime: timeNow,
						Message:            "test",
					},
				},
			},
			greenhouseapis.StatusConditions{
				Conditions: []greenhouseapis.Condition{
					{
						Type:               greenhouseapis.RBACReady,
						Status:             metav1.ConditionFalse,
						LastTransitionTime: timeNow,
						Message:            "test",
					},
					{
						Type:               greenhouseapis.ReadyCondition,
						Status:             metav1.ConditionTrue,
						LastTransitionTime: timeNow,
						Message:            "test",
					},
				},
			},
			greenhouseapis.Condition{
				Type:               greenhouseapis.ReadyCondition,
				Status:             metav1.ConditionTrue,
				LastTransitionTime: timeNow,
				Message:            "test",
			}),
		Entry(
			"should correctly update a condition with matching Type in StatusConditions with a different condition",
			greenhouseapis.StatusConditions{
				Conditions: []greenhouseapis.Condition{
					{
						Type:               greenhouseapis.RBACReady,
						Status:             metav1.ConditionFalse,
						LastTransitionTime: timeNow,
						Message:            "test",
					},
					{
						Type:               greenhouseapis.ReadyCondition,
						Status:             metav1.ConditionFalse,
						LastTransitionTime: timeNow,
						Message:            "test",
					},
				},
			},
			greenhouseapis.StatusConditions{
				Conditions: []greenhouseapis.Condition{
					{
						Type:               greenhouseapis.RBACReady,
						Status:             metav1.ConditionFalse,
						LastTransitionTime: timeNow,
						Message:            "test",
					},
					{
						Type:               greenhouseapis.ReadyCondition,
						Status:             metav1.ConditionTrue,
						LastTransitionTime: timeNow,
						Message:            "test2",
					},
				},
			},
			greenhouseapis.Condition{
				Type:               greenhouseapis.ReadyCondition,
				Status:             metav1.ConditionTrue,
				LastTransitionTime: timeNow,
				Message:            "test2",
			}),
		Entry(
			"should ignore updating a condition with matching Type but differing LastTransitionTime in StatusConditions with a different condition",
			greenhouseapis.StatusConditions{
				Conditions: []greenhouseapis.Condition{
					{
						Type:               greenhouseapis.RBACReady,
						Status:             metav1.ConditionFalse,
						LastTransitionTime: timeNow,
						Message:            "test",
					},
					{
						Type:               greenhouseapis.ReadyCondition,
						Status:             metav1.ConditionFalse,
						LastTransitionTime: timeNow,
						Message:            "test",
					},
				},
			},
			greenhouseapis.StatusConditions{
				Conditions: []greenhouseapis.Condition{
					{
						Type:               greenhouseapis.RBACReady,
						Status:             metav1.ConditionFalse,
						LastTransitionTime: timeNow,
						Message:            "test",
					},
					{
						Type:               greenhouseapis.ReadyCondition,
						Status:             metav1.ConditionFalse,
						LastTransitionTime: timeNow,
						Message:            "test",
					},
				},
			},
			greenhouseapis.Condition{
				Type:               greenhouseapis.ReadyCondition,
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.NewTime(metav1.Now().AddDate(0, 0, -1)),
				Message:            "test",
			}),
		Entry(
			"should not update a conditions LastTransitionTime if only the message changes",
			greenhouseapis.StatusConditions{
				Conditions: []greenhouseapis.Condition{
					{
						Type:               greenhouseapis.RBACReady,
						Status:             metav1.ConditionFalse,
						LastTransitionTime: timeNow,
						Message:            "test",
					},
				},
			},
			greenhouseapis.StatusConditions{
				Conditions: []greenhouseapis.Condition{
					{
						Type:               greenhouseapis.RBACReady,
						Status:             metav1.ConditionFalse,
						LastTransitionTime: timeNow,
						Message:            "test2",
					},
				},
			},
			greenhouseapis.Condition{
				Type:               greenhouseapis.RBACReady,
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.NewTime(metav1.Now().AddDate(0, 0, -1)),
				Message:            "test2",
			},
		),
		Entry(
			"should set and update multiple conditions",
			greenhouseapis.StatusConditions{
				Conditions: []greenhouseapis.Condition{
					{
						Type:               greenhouseapis.RBACReady,
						Status:             metav1.ConditionFalse,
						LastTransitionTime: timeNow,
						Message:            "test",
					},
				},
			},
			greenhouseapis.StatusConditions{
				Conditions: []greenhouseapis.Condition{
					{
						Type:               greenhouseapis.RBACReady,
						Status:             metav1.ConditionFalse,
						LastTransitionTime: timeNow,
						Message:            "test2",
					},
					{
						Type:               greenhouseapis.ReadyCondition,
						Status:             metav1.ConditionTrue,
						LastTransitionTime: timeNow,
						Message:            "test",
					},
				},
			},
			greenhouseapis.Condition{

				Type:               greenhouseapis.RBACReady,
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.NewTime(metav1.Now().AddDate(0, 0, -1)),
				Message:            "test2",
			},
			greenhouseapis.Condition{
				Type:               greenhouseapis.ReadyCondition,
				Status:             metav1.ConditionTrue,
				LastTransitionTime: timeNow,
				Message:            "test",
			},
		),
	)

	It("should correctly identify equal conditions", func() {
		By("identifying equal conditions")
		condition1 := greenhouseapis.Condition{
			Type:               greenhouseapis.ReadyCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: timeNow,
			Message:            "test",
		}
		condition2 := greenhouseapis.Condition{
			Type:               greenhouseapis.ReadyCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.NewTime(metav1.Now().AddDate(0, 0, -1)),
			Message:            "test",
		}
		Expect(condition1.Equal(condition2)).To(BeTrue())

	})
})
