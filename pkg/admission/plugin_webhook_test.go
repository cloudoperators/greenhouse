// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package admission

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

var _ = DescribeTable("Validate PluginOption Type and Value are consistent", func(expectedType greenhousev1alpha1.PluginOptionType, defaultValue any, expErr bool) {

	plugin := &greenhousev1alpha1.Plugin{
		Spec: greenhousev1alpha1.PluginSpec{
			Options: []greenhousev1alpha1.PluginOption{
				{
					Name:    "test",
					Default: test.MustReturnJSONFor(defaultValue),
					Type:    expectedType,
				},
			},
		},
	}
	actErr := validatePluginOptionValueAndType(plugin)
	switch expErr {
	case false:
		Expect(actErr).ToNot(HaveOccurred(), "unexpected error occurred")
	default:
		var err *apierrors.StatusError
		Expect(errors.As(actErr, &err)).To(BeTrue(), "expected an *apierrors.StatusError, got %T", actErr)

		Expect(err.ErrStatus.Reason).To(Equal(metav1.StatusReasonInvalid), "expected an error with reason %s, got %s", metav1.StatusReasonInvalid, err.ErrStatus)
	}
},
	Entry("PluginOptionTypeBool Consistent", greenhousev1alpha1.PluginOptionTypeBool, true, false),
	Entry("PluginOptionTypeBool Inconsistent", greenhousev1alpha1.PluginOptionTypeBool, "notabool", true),
	Entry("PluginOptionTypeString Consistent", greenhousev1alpha1.PluginOptionTypeString, "string", nil),
	Entry("PluginOptionTypeString Consistent Integer as String", greenhousev1alpha1.PluginOptionTypeString, "1", false),
	Entry("PluginOptionTypeString Inconsistent", greenhousev1alpha1.PluginOptionTypeString, 1, true),
	Entry("PluginOptionTypeInt Consistent", greenhousev1alpha1.PluginOptionTypeInt, 1, false),
	Entry("PluginOptionTypeInt Inconsistent", greenhousev1alpha1.PluginOptionTypeInt, "one", true),
	Entry("PluginOptionTypeList Consistent", greenhousev1alpha1.PluginOptionTypeList, []string{"one", "two"}, false),
	Entry("PluginOptionTypeList Inconsistent", greenhousev1alpha1.PluginOptionTypeList, "one", true),
	Entry("PluginOptionTypeMap Consistent", greenhousev1alpha1.PluginOptionTypeMap, map[string]any{"key": "value"}, false),
	Entry("PluginOptionTypeMap Inconsistent", greenhousev1alpha1.PluginOptionTypeMap, "one", true),
	Entry("PluginOptionTypeMap Consistent Nested Map", greenhousev1alpha1.PluginOptionTypeMap, map[string]any{"key": map[string]any{"nestedKey": "value"}}, false),
	Entry("PluginOptionTypeSecret Consistent", greenhousev1alpha1.PluginOptionTypeSecret, "secret", false),
	Entry("PluginOptionTypeSecret Inconsistent", greenhousev1alpha1.PluginOptionTypeSecret, []string{"one", "two"}, true),
)
