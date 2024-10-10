// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/cloudoperators/greenhouse/pkg/controllers/fixtures"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestIgnoreStatusUpdatePredicate(t *testing.T) {
	pred := IgnoreStatusUpdatePredicate()

	tests := []struct {
		name     string
		oldObj   client.Object
		newObj   client.Object
		expected bool
	}{
		{
			name: "it should allow reconciliation when annotations change",
			oldObj: &fixtures.Dummy{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"key": "value1"},
				},
			},
			newObj: &fixtures.Dummy{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"key": "value2"},
				},
			},
			expected: true,
		},
		{
			name: "it should allow reconciliation when labels change",
			oldObj: &fixtures.Dummy{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"key": "value1"},
				},
			},
			newObj: &fixtures.Dummy{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"key": "value2"},
				},
			},
			expected: true,
		},
		{
			name: "it should allow reconciliation when observed generation has changed",
			oldObj: &fixtures.Dummy{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
			},
			newObj: &fixtures.Dummy{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 2,
				},
			},
			expected: true,
		},
		{
			name: "it should allow reconciliation when finalizer is changed",
			oldObj: &fixtures.Dummy{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{"finalizer1"},
				},
			},
			newObj: &fixtures.Dummy{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{""},
				},
			},
			expected: true,
		},
		{
			name: "it should allow reconciliation when deletion timestamp set",
			oldObj: &fixtures.Dummy{
				ObjectMeta: metav1.ObjectMeta{},
			},
			newObj: &fixtures.Dummy{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
			},
			expected: true,
		},
		{
			name: "it should not allow reconciliation when there is no change",
			oldObj: &fixtures.Dummy{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"key": "value"},
					Labels:      map[string]string{"key": "value"},
					Generation:  1,
					Finalizers:  []string{"finalizer"},
				},
			},
			newObj: &fixtures.Dummy{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"key": "value"},
					Labels:      map[string]string{"key": "value"},
					Generation:  1,
					Finalizers:  []string{"finalizer"},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateEvent := event.UpdateEvent{
				ObjectOld: tt.oldObj,
				ObjectNew: tt.newObj,
			}
			result := pred.Update(updateEvent)
			assert.Equal(t, tt.expected, result)
		})
	}
}
