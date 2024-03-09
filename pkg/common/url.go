// Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"fmt"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

// DNSDomain is the DNS domain under which all services shall be exposed.
var DNSDomain string

// URLForExposedServiceInPluginConfig returns the URL that shall be used to expose a service centrally via Greenhouse.
func URLForExposedServiceInPluginConfig(serviceName string, pluginConfig *greenhousev1alpha1.PluginConfig) string {
	return fmt.Sprintf(
		// The pattern shall be $https://$service-$namespace-$cluster.$organisation.$basedomain .
		"https://%s--%s--%s.%s.%s",
		serviceName, pluginConfig.GetNamespace(), pluginConfig.Spec.ClusterName,
		pluginConfig.GetNamespace(), DNSDomain,
	)
}
