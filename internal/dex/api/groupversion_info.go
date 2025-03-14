// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	apis "github.com/cloudoperators/greenhouse/api"
)

/*
The following resources were created to enable integration with the kubebuilder framework without having to use the GRPC client or a dynamic kubernetes client.
Source:
1) github.com/dexidp/dex/storage
2) github.com/dexidp/dex/storage/kubernetes
*/

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "dex.coreos.com", Version: "v1"}
	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &apis.Builder{GroupVersion: GroupVersion}
	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
