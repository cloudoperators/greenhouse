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
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

const (
	Postgres                   = "postgres"
	K8s                        = "kubernetes"
	dexConnectorTypeGreenhouse = "greenhouse-oidc"
	clientIDKey                = "clientID"
	clientSecretKey            = "clientSecret"
)

// Dexter - dex storage adapter interface
// Supported backends: Postgres, K8s
type Dexter interface {
	CreateUpdateConnector(ctx context.Context, k8sClient client.Client, org *greenhouseapisv1alpha1.Organization, configByte []byte, namespace string) error
	CreateUpdateOauth2Client(ctx context.Context, k8sClient client.Client, org *greenhouseapisv1alpha1.Organization, namespace string) error
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

// getDeterministicSecret - generate a deterministic secret based on the clientID and secretKey
func getDeterministicSecret(clientID string, secretKey types.UID) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(clientID))
	return hex.EncodeToString(h.Sum(nil))[:32]
}

// prepareClientSecret - create a coreV1 secret with the clientID and secret
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

// writeCredentialsToNamespace - write the client credentials to the organization namespace
// if the secret already exists, update the secret
// set organization as the owner reference for the secret
func writeCredentialsToNamespace(ctx context.Context, cl client.Client, org *greenhouseapisv1alpha1.Organization, secret *corev1.Secret) error {
	existing := &corev1.Secret{}
	err := cl.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, existing)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.FromContext(ctx).Error(err, "unable to get dex client credentials", "name", secret.Name, "namespace", secret.Namespace)
			return err
		}
		// if not found, create the secret with owner reference
		// set the owner reference
		err = controllerruntime.SetControllerReference(org, secret, cl.Scheme())
		if err != nil {
			log.FromContext(ctx).Error(err, "unable to set controller reference for dex client credentials", "name", secret.Name, "namespace", secret.Namespace)
			return err
		}
		err = cl.Create(ctx, secret)
		if err != nil {
			log.FromContext(ctx).Error(err, "unable to create dex client credentials", "name", secret.Name, "namespace", secret.Namespace)
			return err
		}
		return nil
	}

	existing.Data[clientIDKey] = secret.Data[clientIDKey]
	existing.Data[clientSecretKey] = secret.Data[clientSecretKey]
	err = controllerruntime.SetControllerReference(org, existing, cl.Scheme())
	if err != nil {
		log.FromContext(ctx).Error(err, "unable to create / patch dex client credentials", "name", secret.Name, "namespace", secret.Namespace)
		return err
	}
	err = cl.Update(ctx, existing)
	if err != nil {
		log.FromContext(ctx).Error(err, "unable to update dex client credentials", "name", secret.Name, "namespace", secret.Namespace)
		return err
	}
	return nil
}
