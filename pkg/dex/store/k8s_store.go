// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"encoding/base32"
	"fmt"
	"hash/fnv"
	"log/slog"
	"strings"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/kubernetes"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/common"
	dexapi "github.com/cloudoperators/greenhouse/pkg/dex/api"
	"github.com/cloudoperators/greenhouse/pkg/util"
)

type k8sDex struct {
	storage storage.Storage
	backend string
}

const encoding = "abcdefghijklmnopqrstuvwxyz234567"

// newKubernetesStore - creates a new kubernetes storage backend for dex
func newKubernetesStore(logger *slog.Logger) (storage.Storage, error) {
	cfg := kubernetes.Config{InCluster: true}
	kEnv := clientutil.GetEnvOrDefault("KUBECONFIG", "")
	if strings.TrimSpace(kEnv) != "" {
		cfg.InCluster = false
		cfg.KubeConfigFile = kEnv
	}
	dexStorage, err := cfg.Open(logger)
	if err != nil {
		return nil, err
	}
	return dexStorage, nil
}

func (k *k8sDex) GetBackend() string {
	return k.backend
}

// CreateUpdateConnector - creates or updates a dex connector in dex kubernetes storage backend
func (k *k8sDex) CreateUpdateConnector(ctx context.Context, k8sClient client.Client, org *greenhouseapisv1alpha1.Organization, configByte []byte, namespace string) (err error) {
	var result clientutil.OperationResult
	var dexConnector = new(dexapi.Connector)
	dexConnector.Namespace = namespace
	dexConnector.ObjectMeta.Name = org.GetName()
	result, err = clientutil.CreateOrPatch(ctx, k8sClient, dexConnector, func() error {
		dexConnector.DexConnector.Type = dexConnectorTypeGreenhouse
		dexConnector.DexConnector.Name = cases.Title(language.English).String(org.Name)
		dexConnector.DexConnector.ID = org.GetName()
		dexConnector.DexConnector.Config = configByte
		return controllerutil.SetControllerReference(org, dexConnector, k8sClient.Scheme())
	})
	if err != nil {
		return
	}
	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created dex connector", "namespace", dexConnector.Namespace, "name", dexConnector.GetName())
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated dex connector", "namespace", dexConnector.Namespace, "name", dexConnector.GetName())
	}
	return
}

// CreateUpdateOauth2Client - creates or updates an oauth2 client in dex kubernetes storage backend
func (k *k8sDex) CreateUpdateOauth2Client(ctx context.Context, k8sClient client.Client, org *greenhouseapisv1alpha1.Organization, namespace string) error {
	var oAuth2Client = new(dexapi.OAuth2Client)
	oAuth2Client.ObjectMeta.Name = encodedOAuth2ClientName(org.Name)
	oAuth2Client.ObjectMeta.Namespace = namespace
	generatedClientSecret := getDeterministicSecret(org.Name, org.GetUID())

	result, err := clientutil.CreateOrPatch(ctx, k8sClient, oAuth2Client, func() error {
		oAuth2Client.Client.Public = true
		oAuth2Client.Client.ID = org.Name
		oAuth2Client.Secret = generatedClientSecret
		oAuth2Client.Client.Name = org.Name
		for _, requiredRedirectURL := range []string{
			"http://localhost:8085",
			"http://localhost:8000",
			"https://dashboard." + common.DNSDomain,
			fmt.Sprintf("https://%s.dashboard.%s", org.Name, common.DNSDomain),
		} {
			oAuth2Client.Client.RedirectURIs = util.AppendStringToSliceIfNotContains(requiredRedirectURL, oAuth2Client.RedirectURIs)
		}
		return controllerutil.SetControllerReference(org, oAuth2Client, k8sClient.Scheme())
	})
	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created oauth2client", "namespace", oAuth2Client.Namespace, "name", oAuth2Client.GetName())
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated oauth2client", "namespace", oAuth2Client.Namespace, "name", oAuth2Client.GetName())
	}

	secret := prepareClientSecret(namespace, org.Name, generatedClientSecret)
	err = writeCredentialsToNamespace(ctx, k8sClient, secret)
	if err != nil {
		return err
	}
	return nil
}

// encodedOAuth2ClientName - encodes the org name to a base32 string
// for kubernetes backend storage we need to encode the name OAuth2Client CR name
// See https://github.com/dexidp/dex/issues/1606 for encoding
func encodedOAuth2ClientName(orgName string) string {
	return strings.TrimRight(base32.
		NewEncoding(encoding).
		EncodeToString(fnv.New64().Sum([]byte(orgName))), "=",
	)
}

// GetStorage - returns the underlying dex storage interface
func (k *k8sDex) GetStorage() storage.Storage {
	return k.storage
}

func (k *k8sDex) Close() error {
	return k.storage.Close()
}
