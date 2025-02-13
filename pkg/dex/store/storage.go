// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/dexidp/dex/storage"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/common"
)

const (
	Postgres                   = "postgres"
	K8s                        = "kubernetes"
	dexConnectorTypeGreenhouse = "greenhouse-oidc"
)

// Dexter - dex storage adapter interface
// Supported backends: Postgres, K8s
type Dexter interface {
	CreateUpdateConnector(ctx context.Context, k8sClient client.Client, org *greenhouseapisv1alpha1.Organization, configByte []byte) error
	CreateUpdateOauth2Client(ctx context.Context, k8sClient client.Client, org *greenhouseapisv1alpha1.Organization) error
	GetStorage() storage.Storage
	GetBackend() string
	Close() error
}

// NewDexStorageFactory - create a new dex storage adapter depending on the backend
func NewDexStorageFactory(logger *slog.Logger, backend string) (Dexter, error) {
	switch backend {
	case Postgres:
		dexStorage, err := newPostgresStore(logger)
		if err != nil {
			return nil, err
		}
		return &pgDex{storage: dexStorage, backend: backend}, nil
	case K8s:
		dexStorage, err := newKubernetesStore(logger)
		if err != nil {
			return nil, err
		}
		return &k8sDex{storage: dexStorage, backend: backend}, nil
	default:
		return nil, fmt.Errorf("unknown dexStorage backend: %s", backend)
	}
}

func getRedirects(org *greenhouseapisv1alpha1.Organization, redirectURIs []string) []string {
	redirects := []string{
		"http://localhost:8085",
		"https://dashboard." + common.DNSDomain,
		fmt.Sprintf("https://%s.dashboard.%s", org.Name, common.DNSDomain),
	}

	for _, r := range redirects {
		if !slices.Contains(redirectURIs, r) {
			redirectURIs = append(redirectURIs, r)
		}
	}

	return redirects
}
