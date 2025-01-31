// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/dexidp/dex/storage"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
)

const (
	Postgres                   = "postgres"
	K8s                        = "kubernetes"
	dexConnectorTypeGreenhouse = "greenhouse-oidc"
	clientIDKey                = "clientID"
	clientSecretKey            = "clientSecret"
)

type Dexter interface {
	CreateUpdateConnector(ctx context.Context, k8sClient client.Client, org *greenhouseapisv1alpha1.Organization, configByte []byte, namespace string) error
	CreateUpdateOauth2Client(ctx context.Context, k8sClient client.Client, org *greenhouseapisv1alpha1.Organization, namespace string) error
	GetStorage() storage.Storage
	GetBackend() string
	Close() error
}

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

func getDeterministicSecret(clientID, version string, secretKey types.UID) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(version + clientID))
	return hex.EncodeToString(h.Sum(nil))[:32]
}

func prepareClientSecret(namespace, clientID, clientSecret string) *corev1.Secret {
	secret := new(corev1.Secret)
	secret.SetName(clientID + "-dex-secrets")
	secret.SetNamespace(namespace)
	secret.Data = map[string][]byte{
		clientIDKey:     []byte(clientID),
		clientSecretKey: []byte(clientSecret),
	}
	return secret
}

func writeCredentialsToNamespace(ctx context.Context, cl client.Client, secret *corev1.Secret) error {
	result, err := clientutil.CreateOrPatch(ctx, cl, secret, func() error {
		return nil
	})
	if err != nil {
		log.FromContext(ctx).Error(err, "unable to create dex client credentials", "name", secret.Name, "namespace", secret.Namespace)
	}
	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created oauth2client secrets", "name", secret.Name, "namespace", secret.Namespace)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated oauth2client secrets", "name", secret.Name, "namespace", secret.Namespace)
	}
	return nil
}
