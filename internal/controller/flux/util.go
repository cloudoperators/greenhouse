// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package flux

import (
	"context"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	sourcecontroller "github.com/fluxcd/source-controller/api/v1"
)

const (
	deliveryToolLabel = "greenhouse.sap/deployment-tool"
	deliveryToolFlux  = "flux"
)

func convertName(repoName string) (convertedName, repoType string) {
	repoType = sourcecontroller.HelmRepositoryTypeDefault
	// set the helm repository type to OCI if the repo name starts with oci://
	if strings.HasPrefix(repoName, "oci://") {
		repoType = sourcecontroller.HelmRepositoryTypeOCI
	}
	// remove prefixes
	var prefixes = []string{
		"oci://",
		"https://",
		"http://",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(repoName, prefix) {
			convertedName = strings.TrimPrefix(repoName, prefix)
			break
		}
	}

	convertedName = strings.ReplaceAll(convertedName, ".", "-")
	convertedName = strings.ReplaceAll(convertedName, "/", "-")
	return convertedName, repoType
}

func findHelmRepositoryByUrl(ctx context.Context, k8sClient client.Client, nS, url string) *sourcecontroller.HelmRepository {
	helmRepositoryList := new(sourcecontroller.HelmRepositoryList)
	if err := k8sClient.List(ctx, helmRepositoryList); err != nil {
		return nil
	}
	for _, helmRepository := range helmRepositoryList.Items {
		if helmRepository.Namespace == nS && helmRepository.Spec.URL == url {
			return &helmRepository
		}
	}
	return nil
}
