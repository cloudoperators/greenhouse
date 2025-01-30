// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package features

import (
	"context"
	"log"
	"time"

	ofinprocess "github.com/open-feature/go-sdk-contrib/providers/go-feature-flag-in-process/pkg"
	"github.com/open-feature/go-sdk/openfeature"
	goffclient "github.com/thomaspoignant/go-feature-flag"
	"github.com/thomaspoignant/go-feature-flag/retriever/k8sretriever"
	"k8s.io/client-go/rest"

	"github.com/cloudoperators/greenhouse/pkg/clientutil"
)

func GetOFClient(appName string) *openfeature.Client {
	k8sInClusterConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to create in-cluster config: %s", err)
	}

	goFeatureFlagConfig := &goffclient.Config{
		PollingInterval: 30 * time.Minute,
		Context:         context.Background(),
		Retriever: &k8sretriever.Retriever{
			Namespace:     clientutil.GetEnvOrDefault("FEATURE_FLAG_NAMESPACE", "greenhouse"),
			ConfigMapName: clientutil.GetEnvOrDefault("FEATURE_FLAG_CONFIG_MAP_NAME", "greenhouse-feature-flags"),
			Key:           clientutil.GetEnvOrDefault("FEATURE_FLAG_CONFIG_MAP_KEY", "config.yaml"),
			ClientConfig:  *k8sInClusterConfig,
		},
	}

	options := ofinprocess.ProviderOptions{
		GOFeatureFlagConfig: goFeatureFlagConfig,
	}

	provider, err := ofinprocess.NewProviderWithContext(context.Background(), options)
	if err != nil {
		log.Fatalf("Failed to create provider: %s", err)
	}
	if err = openfeature.SetNamedProvider(appName, provider); err != nil {
		log.Fatalf("Failed to set provider: %s", err)
	}
	if err = goffclient.Init(*goFeatureFlagConfig); err != nil {
		log.Fatalf("Failed to init provider: %s", err)
	}

	return openfeature.NewClient(appName)
}
