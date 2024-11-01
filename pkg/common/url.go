// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

// DNSDomain is the DNS domain under which all services shall be exposed.
var DNSDomain string

// URLForExposedServiceInPlugin returns the URL that shall be used to expose a service centrally via Greenhouse.
// The pattern shall be $https://$cluster--$hash.$organisation.$basedomain, where $hash = $service--$namespace
// We know $cluster is no longer than 40 characters and does not contain "--"
func URLForExposedServiceInPlugin(serviceName string, plugin *greenhousev1alpha1.Plugin) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s--%s", serviceName, plugin.Spec.ReleaseNamespace)))
	hashString := hex.EncodeToString(hash[:])
	subdomain := fmt.Sprintf("%s--%s", plugin.Spec.ClusterName, hashString[:7])
	return fmt.Sprintf(
		"https://%s.%s.%s",
		subdomain, plugin.GetNamespace(), DNSDomain,
	)
}

// ExtractCluster extracts the cluster name from the host.
// The pattern shall be $cluster--$hash, where $hash = service--$namespace
func ExtractCluster(host string) (cluster string, err error) {
	if strings.HasPrefix(host, "https://") {
		return "", fmt.Errorf("invalid host: %s, no protocol expected", host)
	}
	parts := strings.SplitN(host, ".", 2)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid host: %s", host)
	}
	parts = strings.SplitN(parts[0], "--", 2)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid host: %s", host)
	}
	return parts[0], nil
}
