package catalog

import (
	"slices"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getReadyCondition(conditions []metav1.Condition) *metav1.Condition {
	if len(conditions) == 0 {
		return nil
	}
	readyIndex := slices.IndexFunc(conditions, func(cond metav1.Condition) bool {
		return cond.Type == "Ready"
	})
	if readyIndex < 0 {
		return nil
	}
	return &conditions[readyIndex]
}
