// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package flux

import (
	"context"
	"strings"

	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	HelmRepositoryDefaultNamespace = "greenhouse" // TODO: make this configurable via args or env var
)

func GetSourceRepositoryType(repositoryURL string) string {
	if strings.HasPrefix(repositoryURL, "oci://") {
		return sourcev1.HelmRepositoryTypeOCI
	}
	return sourcev1.HelmRepositoryTypeDefault
}

func ChartURLToName(repositoryURL string) (repositoryName string) {
	// remove prefixes
	var prefixes = []string{
		"oci://",
		"https://",
		"http://",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(repositoryURL, prefix) {
			repositoryName = strings.TrimPrefix(repositoryURL, prefix)
			break
		}
	}

	repositoryName = strings.ReplaceAll(repositoryName, ".", "-")
	repositoryName = strings.ReplaceAll(repositoryName, "/", "-")
	return
}

func FindHelmRepositoryByURL(ctx context.Context, k8sClient client.Client, url, namespace string) (*sourcev1.HelmRepository, error) {
	repositoryName := ChartURLToName(url)
	helmRepository := &sourcev1.HelmRepository{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: repositoryName, Namespace: namespace}, helmRepository); err != nil {
		return nil, err
	}
	return helmRepository, nil
}
