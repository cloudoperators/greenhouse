// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"fmt"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

// DNSDomain is the DNS domain under which all services shall be exposed.
var DNSDomain string

// URLForExposedServiceInPlugin returns the URL that shall be used to expose a service centrally via Greenhouse.
func URLForExposedServiceInPlugin(serviceName string, plugin *greenhousev1alpha1.Plugin) string {
	return fmt.Sprintf(
		// The pattern shall be $https://$service-$namespace-$cluster.$organisation.$basedomain .
		"https://%s--%s--%s.%s.%s",
		serviceName, plugin.GetNamespace(), plugin.Spec.ClusterName,
		plugin.GetNamespace(), DNSDomain,
	)
}
