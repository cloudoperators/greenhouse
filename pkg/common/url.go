// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

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
