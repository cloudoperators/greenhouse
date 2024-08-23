// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
)

func getPortForExposedService(o runtime.Object) (*corev1.ServicePort, error) {
	svc, err := convertRuntimeObjectToCoreV1Service(o)
	if err != nil {
		return nil, err
	}

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

func convertRuntimeObjectToCoreV1Service(o interface{}) (*corev1.Service, error) {
	switch obj := o.(type) {
	case *corev1.Service:
		// If it's already a corev1.Service, no conversion needed
		return obj, nil
	case *unstructured.Unstructured:
		// If it's an unstructured object, convert it to corev1.Service
		var service corev1.Service
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &service)
		return &service, errors.Wrap(err, "failed to convert to corev1.Service from unstructured object")
	default:
		return nil, fmt.Errorf("unsupported runtime.Object type: %T", obj)
	}
}
