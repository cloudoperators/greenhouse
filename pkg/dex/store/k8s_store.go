// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"encoding/base32"
	"hash/fnv"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/kubernetes"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	dexapi "github.com/cloudoperators/greenhouse/pkg/dex/api"
)

type k8sDex struct {
	storage storage.Storage
	backend string
}

const encoding = "abcdefghijklmnopqrstuvwxyz234567"

// newKubernetesStore - creates a new kubernetes storage backend for dex
func newKubernetesStore(logger *slog.Logger) (storage.Storage, error) {
	cfg := kubernetes.Config{InCluster: true}
	cfgPath := determineKubeMode()
	if strings.TrimSpace(cfgPath) != "" {
		cfg.InCluster = false
		cfg.KubeConfigFile = cfgPath
	}
	dexStorage, err := cfg.Open(logger)
	if err != nil {
		return nil, err
	}
	return dexStorage, nil
}

func determineKubeMode() string {
	cfgPath := os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
	if strings.TrimSpace(cfgPath) != "" {
		return cfgPath
	}
	_, err := rest.InClusterConfig()
	if err == nil {
		return ""
	}
	return filepath.Join(homedir.HomeDir(), ".kube", "config")
}

func (k *k8sDex) GetBackend() string {
	return k.backend
}

// CreateUpdateConnector - creates or updates a dex connector in dex kubernetes storage backend
func (k *k8sDex) CreateUpdateConnector(ctx context.Context, k8sClient client.Client, org *greenhouseapisv1alpha1.Organization, configByte []byte) (err error) {
	var result clientutil.OperationResult
	var dexConnector = new(dexapi.Connector)
	namespaceName := org.GetName()
	dexConnector.SetName(namespaceName)
	dexConnector.SetNamespace(namespaceName)
	result, err = clientutil.CreateOrPatch(ctx, k8sClient, dexConnector, func() error {
		dexConnector.DexConnector.Type = dexConnectorTypeGreenhouse
		dexConnector.DexConnector.Name = cases.Title(language.English).String(org.Name)
		dexConnector.DexConnector.ID = org.GetName()
		dexConnector.DexConnector.Config = configByte
		return controllerutil.SetControllerReference(org, dexConnector, k8sClient.Scheme())
	})
	if err != nil {
		log.FromContext(ctx).Error(err, "failed to create/update dex connector", "namespace", dexConnector.Namespace, "name", dexConnector.GetName())
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
func (k *k8sDex) CreateUpdateOauth2Client(ctx context.Context, k8sClient client.Client, org *greenhouseapisv1alpha1.Organization) error {
	var oAuth2Client = new(dexapi.OAuth2Client)
	namespaceName := org.GetName()
	oAuth2Client.SetName(encodedOAuth2ClientName(namespaceName))
	oAuth2Client.SetNamespace(namespaceName)

	result, err := clientutil.CreateOrPatch(ctx, k8sClient, oAuth2Client, func() error {
		oAuth2Client.Client.Public = true
		oAuth2Client.Client.ID = org.Name
		oAuth2Client.Client.Name = org.Name
		oAuth2Client.RedirectURIs = getRedirects(org, oAuth2Client.RedirectURIs)
		return controllerutil.SetControllerReference(org, oAuth2Client, k8sClient.Scheme())
	})
	if err != nil {
		log.FromContext(ctx).Error(err, "failed to create/update oauth2client", "namespace", oAuth2Client.Namespace, "name", oAuth2Client.GetName())
		return err
	}
	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created oauth2client", "namespace", oAuth2Client.Namespace, "name", oAuth2Client.GetName())
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated oauth2client", "namespace", oAuth2Client.Namespace, "name", oAuth2Client.GetName())
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
