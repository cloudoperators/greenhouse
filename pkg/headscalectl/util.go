// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package headscalectl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudoperators/greenhouse/pkg/clientutil"
)

const (
	secretKey = "HEADSCALE_CLI_API_KEY" //nolint:gosec
)

var (
	headscaleGRCPClientFunc       = clientutil.NewHeadscaleGRPCClient
	headscaleGRPCSocketClientFunc = clientutil.NewHeadscaleGRPCSocketClient
)

func validateFlags() error {
	if socketCall {
		if socketPath == "" {
			return errors.New("socket path is empty")
		}
		return nil
	}
	if headscaleGRPCURL == "" {
		return errors.New("headscale GRPC URL is empty")
	}
	if headscaleAPIKey == "" {
		return errors.New("headscale API key is empty")
	}
	return nil
}

func getKubeconfigOrDie(kubecontext string) *rest.Config {
	if kubecontext == "" {
		kubecontext = os.Getenv("KUBECONTEXT")
	}
	restConfig, err := config.GetConfigWithContext(kubecontext)
	if err != nil {
		log.FromContext(context.Background()).Error(err, "Failed to load kubeconfig")
		os.Exit(1)
	}
	return restConfig
}

/*
	func checkIfSecretExistsInCluster(secretName, secretNamespace string) (*corev1.Secret, bool) {
		var kubeClient client.Client
		restConfig := getKubeconfigOrDie("")
		kubeClient, err := clientutil.NewK8sClient(restConfig)
		if err != nil {
			log.FromContext(context.Background()).Error(err, "Failed to create Kubernetes client")
		}
		secret := new(corev1.Secret)
		secret.Name = secretName
		secret.Namespace = secretNamespace
		err = kubeClient.Get(context.Background(), client.ObjectKey{
			Name:      secret.Name,
			Namespace: secret.Namespace,
		}, secret)
		if err != nil {
			return nil, false
		}
		return secret, true
	}
*/
func createOrUpdateSecretInCluster(APIkey, secretName, secretNamespace string) {
	var kubeClient client.Client
	restConfig := getKubeconfigOrDie("")
	kubeClient, err := clientutil.NewK8sClient(restConfig)
	if err != nil {
		log.FromContext(context.Background()).Error(err, "Failed to create Kubernetes client")
	}
	secret := new(corev1.Secret)
	secret.Name = secretName
	secret.Namespace = secretNamespace
	result, err := clientutil.CreateOrPatch(context.Background(), kubeClient, secret, func() error {
		secret.StringData = map[string]string{
			secretKey: APIkey,
		}
		return nil
	})
	if err != nil {
		log.FromContext(context.Background()).Error(err, "Failed to create secret")
	}
	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(context.Background()).Info("created secret", "name", secret.Name)
	case clientutil.OperationResultUpdated:
		log.FromContext(context.Background()).Info("updated secret", "name", secret.Name)
	}
}

func Output(result interface{}, override, outputFormat string) {
	var jsonBytes []byte
	var err error
	switch outputFormat {
	case "json":
		jsonBytes, err = json.MarshalIndent(result, "", "\t")
		if err != nil {
			log.FromContext(context.Background()).Error(err, "Error marshalling JSON")
		}
	case "json-line":
		jsonBytes, err = json.Marshal(result)
		if err != nil {
			log.FromContext(context.Background()).Error(err, "Error marshalling JSON")
		}
	case "yaml":
		jsonBytes, err = yaml.Marshal(result)
		if err != nil {
			log.FromContext(context.Background()).Error(err, "Error marshalling YAML")
		}
	default:
		log.FromContext(context.Background()).Info(override)
		return
	}
	fmt.Println(string(jsonBytes))
}
