// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"crypto/sha256"
	"fmt"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

// DNSDomain is the DNS domain under which all services shall be exposed.
var DNSDomain string

// URLForExposedServiceInPlugin returns the URL that shall be used to expose a service centrally via Greenhouse.
// The pattern shall be $https://$service--$cluster--$namespace.$organisation.$basedomain .
// If the first subdomain exceeds 63 characters, it will be shortened to 63 characters by appending a hash.
func URLForExposedServiceInPlugin(serviceName string, plugin *greenhousev1alpha1.Plugin) string {
	subdomain := fmt.Sprintf("%s--%s--%s", serviceName, plugin.Spec.ClusterName, plugin.Spec.ReleaseNamespace)
	if len(subdomain) > 63 {
		hashedSubdomain := sha256.Sum256([]byte(subdomain))
		subdomain = fmt.Sprintf("%s-%x", subdomain[:54], hashedSubdomain[:4])
	}
	return fmt.Sprintf(
		"https://%s.%s.%s",
		subdomain, plugin.GetNamespace(), DNSDomain,
	)
}
