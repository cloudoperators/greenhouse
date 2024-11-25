// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// Package v1alpha2 contains API Schema definitions for the greenhouse.sap v1alpha2 API group
// +kubebuilder:object:generate=true
// +groupName=greenhouse.sap
package v1alpha2

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/cloudoperators/greenhouse/pkg/apis"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: apis.GroupName, Version: "v1alpha2"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &apis.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
