// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package flux

import (
	"errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

// computeReadyCondition computes the ReadyCondition for the Plugin based on various status conditions
func computeReadyCondition(
	conditions greenhousemetav1alpha1.StatusConditions,
) (readyCondition greenhousemetav1alpha1.Condition) {

	readyCondition = *conditions.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)

	// If the Cluster is not ready, the Plugin could not be ready
	if conditions.GetConditionByType(greenhousev1alpha1.ClusterAccessReadyCondition).IsFalse() {
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Message = "cluster access not ready"
		return readyCondition
	}
	// If the Helm reconcile failed, the Plugin is not up to date / ready
	if conditions.GetConditionByType(greenhousev1alpha1.HelmReconcileFailedCondition).IsTrue() {
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Message = "Helm reconcile failed"
		return readyCondition
	}

	// In other cases, the Plugin is ready
	readyCondition.Status = metav1.ConditionTrue
	readyCondition.Message = "ready"
	return readyCondition
}

func getPortForExposedService(svc corev1.Service) (*corev1.ServicePort, error) {
	if len(svc.Spec.Ports) == 0 {
		return nil, errors.New("service has no ports")
	}

	// Check for matching of named port set by label
	var namedPort = svc.Labels[greenhouseapis.LabelKeyExposeNamedPort]

	if namedPort != "" {
		for _, port := range svc.Spec.Ports {
			if port.Name == namedPort {
				return port.DeepCopy(), nil
			}
		}
	}

	// Default to first port
	return svc.Spec.Ports[0].DeepCopy(), nil
}
