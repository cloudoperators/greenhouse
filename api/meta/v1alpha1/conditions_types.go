// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConditionType is a valid condition of a resource.
type ConditionType string

// ConditionReason is a valid reason for a condition of a resource.
type ConditionReason string

// Condition contains additional information on the state of a resource.
type Condition struct {
	// Type of the condition.
	Type ConditionType `json:"type"`

	// Status of the condition.
	Status metav1.ConditionStatus `json:"status"`

	// Reason is a one-word, CamelCase reason for the condition's last transition.
	Reason ConditionReason `json:"reason,omitempty"`

	// LastTransitionTime is the last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`

	// Message is an optional human-readable message indicating details about the last transition.
	Message string `json:"message,omitempty"`
}

// NewCondition returns a Condition with the given type, status, reason and message. LastTransitionTime is set to now.
func NewCondition(condition ConditionType, status metav1.ConditionStatus, reason ConditionReason, message string) Condition {
	return Condition{
		Type:               condition,
		Status:             status,
		Reason:             reason,
		LastTransitionTime: metav1.Now(),
		Message:            message,
	}
}

// TrueCondition returns a Condition with ConditionTrue and the given type, reason and message. LastTransitionTime is set to now.
func TrueCondition(t ConditionType, reason ConditionReason, message string) Condition {
	return NewCondition(t, metav1.ConditionTrue, reason, message)
}

// FalseCondition returns a Condition with ConditionFalse and the given type, reason and message. LastTransitionTime is set to now.
func FalseCondition(t ConditionType, reason ConditionReason, message string) Condition {
	return NewCondition(t, metav1.ConditionFalse, reason, message)
}

// UnknownCondition returns a Condition with ConditionUnknown and the given type, reason and message. LastTransitionTime is set to now.
func UnknownCondition(t ConditionType, reason ConditionReason, message string) Condition {
	return NewCondition(t, metav1.ConditionUnknown, reason, message)
}

// Equal returns true if the condition is identical to the supplied condition,
// ignoring the LastTransitionTime.
func (c *Condition) Equal(other Condition) bool {
	return c.Type == other.Type &&
		c.Status == other.Status &&
		c.Reason == other.Reason &&
		c.Message == other.Message
}

// IsTrue returns true if the condition is true.
func (c *Condition) IsTrue() bool {
	return c.Status == metav1.ConditionTrue
}

func (c *Condition) IsUnknown() bool {
	return c.Status == metav1.ConditionUnknown
}

// IsFalse returns true if the condition is false.
func (c *Condition) IsFalse() bool {
	return c.Status == metav1.ConditionFalse
}

// +kubebuilder:object:generate=true

// A StatusConditions contains a list of conditions.
// Only one condition of a given type may exist in the list.
type StatusConditions struct {
	// +listType="map"
	// +listMapKey=type
	Conditions []Condition `json:"conditions,omitempty"`
}

// SetConditions sets the corresponding Conditions in StatusConditions to newConditions.
// If the LastTransitionTime of a new Condition is not set, it is set to the current time.
// If a Condition of the same ConditionType exists, it is updated. The LastTransitionTime is only updated if the Status changes.
// If a condition of the same ConditionType does not exist, it is appended.
func (sc *StatusConditions) SetConditions(conditionsToSet ...Condition) {
	for _, conditionToSet := range conditionsToSet {
		if conditionToSet.LastTransitionTime.IsZero() {
			conditionToSet.LastTransitionTime = metav1.Now()
		}
		exists := false
		for idx, currentCondition := range sc.Conditions {
			// if the condition already exists, update it if necessary
			if currentCondition.Type == conditionToSet.Type {
				exists = true
				if !currentCondition.Equal(conditionToSet) {
					// do not update LastTransitionTime if status does not change
					if currentCondition.Status == conditionToSet.Status {
						conditionToSet.LastTransitionTime = currentCondition.LastTransitionTime
					}
					(sc.Conditions)[idx] = conditionToSet
				}
				break
			}
		}
		// if the condition does not exist, append it
		if !exists {
			sc.Conditions = append(sc.Conditions, conditionToSet)
		}
	}
}

// GetConditionByType returns the condition of the given type, if it exists.
func (sc *StatusConditions) GetConditionByType(conditionType ConditionType) *Condition {
	if len(sc.Conditions) == 0 {
		return nil
	}
	for _, condition := range sc.Conditions {
		if condition.Type == conditionType {
			return &condition
		}
	}
	return nil
}

// IsReadyTrue returns true if the Ready condition is true.
func (sc *StatusConditions) IsReadyTrue() bool {
	c := sc.GetConditionByType(ReadyCondition)
	if c == nil {
		return false
	}
	return c.IsTrue()
}
