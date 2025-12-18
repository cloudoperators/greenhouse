// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package setup

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/cenkalti/backoff/v5"
	aregv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloudoperators/greenhouse/internal/local/klient"
	"github.com/cloudoperators/greenhouse/internal/local/utils"
)

type Webhook struct {
	DockerFile string `yaml:"dockerFile" json:"dockerFile"`
}

const (
	LocalDevIMG                        = "greenhouse/manager:local"
	WebhookContainer                   = "webhook"
	AuthzContainer                     = "authz"
	DeploymentKind                     = "Deployment"
	WebhookDeploymentNameSuffix        = "-webhook"
	MutatingWebhookConfigurationKind   = "MutatingWebhookConfiguration"
	ValidatingWebhookConfigurationKind = "ValidatingWebhookConfiguration"
	webhookCertSecSuffix               = "-webhook-cert"
	webhookCertInjectionSuffix         = "-client-cert"
)

// setupWebhookManifest - sets up the webhook manifest by modifying the webhook configurations and deployment.
// If devMode is true, it modifies the webhook configurations to use host.docker.internal URL and removes service from clientConfig.
// It does not deploy the webhook deployment in devMode.
func (m *Manifest) setupWebhookManifest(resources []map[string]any, devMode bool) ([]map[string]any, error) {
	webhookManifests := make([]map[string]any, 0)
	webhookURL := getWebhookURL()
	webhookResources := extractResourcesByKinds(resources, MutatingWebhookConfigurationKind, ValidatingWebhookConfigurationKind)
	utils.Log("setting cert-manager annotation for webhook resources...")
	webhookResources = m.setCertManagerAnnotation(webhookResources)
	if devMode {
		utils.Log("enabling webhook local development...")
		webhooks, err := m.modifyWebhooks(webhookResources, webhookURL)
		if err != nil {
			return nil, err
		}
		if len(webhooks) > 0 {
			webhookManifests = append(webhookManifests, webhooks...)
		}
		return webhookManifests, nil
	}
	webhookManifests = append(webhookManifests, webhookResources...)
	releaseName := m.ReleaseName
	webhookDeployment, err := extractResourceByNameKind(resources, releaseName+WebhookDeploymentNameSuffix, DeploymentKind)
	if err != nil {
		return nil, err
	}

	utils.Log("modifying webhook deployment...")
	webhookDeployment, err = m.modifyWebhookDeployment(webhookDeployment)
	if err != nil {
		return nil, err
	}
	webhookManifests = append(webhookManifests, webhookDeployment)
	return webhookManifests, nil
}

// modifyWebhookDeployment - sets the local image of the webhook deployment
func (m *Manifest) modifyWebhookDeployment(deploymentResource map[string]any) (map[string]any, error) {
	deployment := &appsv1.Deployment{}
	deploymentStr, err := utils.Stringy(deploymentResource)
	if err != nil {
		return nil, err
	}
	// convert yaml to appsv1.Deployment
	err = utils.FromYamlToK8sObject(deploymentStr, deployment)
	if err != nil {
		return nil, err
	}
	index := getContainerIndex(deployment, WebhookContainer)
	if index == -1 {
		return nil, errors.New("manager container not found in deployment")
	}
	deployment.Spec.Template.Spec.Containers[index].Image = LocalDevIMG
	deployment.Spec.Replicas = utils.Int32P(1)
	depBytes, err := utils.FromK8sObjectToYaml(deployment, appsv1.SchemeGroupVersion)
	if err != nil {
		return nil, err
	}
	return utils.RawK8sInterface(depBytes)
}

func (m *Manifest) setupAuthzManifest(resources []map[string]any) ([]map[string]any, error) {
	manifests := make([]map[string]any, 0)
	authzDeployment, err := extractResourceByNameKind(resources, m.ReleaseName, DeploymentKind)
	if err != nil {
		return nil, err
	}

	utils.Log("modifying authorization webhook deployment...")
	authzDeployment, err = m.modifyAuthzDeployment(authzDeployment)
	if err != nil {
		return nil, err
	}
	manifests = append(manifests, authzDeployment)

	remainingResources := extractResourcesByKinds(resources, ServiceKind, "ServiceAccount", "ClusterRole", "ClusterRoleBinding")
	manifests = append(manifests, remainingResources...)
	return manifests, nil
}

func (m *Manifest) modifyAuthzDeployment(deploymentResource map[string]any) (map[string]any, error) {
	deployment := &appsv1.Deployment{}
	deploymentStr, err := utils.Stringy(deploymentResource)
	if err != nil {
		return nil, err
	}
	err = utils.FromYamlToK8sObject(deploymentStr, deployment)
	if err != nil {
		return nil, err
	}
	index := getContainerIndex(deployment, AuthzContainer)
	if index == -1 {
		return nil, errors.New("authz container not found in deployment")
	}
	deployment.Spec.Template.Spec.Containers[index].Image = LocalDevIMG
	deployment.Spec.Replicas = utils.Int32P(1)
	depBytes, err := utils.FromK8sObjectToYaml(deployment, appsv1.SchemeGroupVersion)
	if err != nil {
		return nil, err
	}
	return utils.RawK8sInterface(depBytes)
}

func (m *Manifest) setCertManagerAnnotation(resources []map[string]any) []map[string]any {
	certSecName := fmt.Sprintf("greenhouse/%s%s", m.ReleaseName, webhookCertInjectionSuffix)
	for idx, resource := range resources {
		m, ok := resource["metadata"]
		if !ok {
			continue
		}
		metadata, ok := m.(map[string]any)
		if !ok {
			continue
		}
		// Grab existing annotations (if any), else make a new map
		var annotations map[string]any
		if annAny, found := metadata["annotations"]; found {
			a := annAny.(map[string]any) //nolint:errcheck
			annotations = a
		} else {
			annotations = make(map[string]any)
		}
		// inject cert-manager annotation
		annotations["cert-manager.io/inject-ca-from"] = certSecName
		metadata["annotations"] = annotations
		resource["metadata"] = metadata
		resources[idx] = resource
	}
	return resources
}

// modifyWebhooks - modifies the webhook configurations to use host.docker.internal URL and removes service from clientConfig
// during local development of webhooks api server will forward the request to host machine where the webhook is running at port 9443
func (m *Manifest) modifyWebhooks(resources []map[string]any, webhookURL string) ([]map[string]any, error) {
	modifiedWebhooks := make([]map[string]any, 0)
	for _, resource := range resources {
		if k, ok := resource["kind"].(string); ok {
			var hookBytes []byte
			var err error
			switch k {
			case MutatingWebhookConfigurationKind:
				hookBytes, err = m.modifyWebhook(resource, &aregv1.MutatingWebhookConfiguration{}, webhookURL)
			case ValidatingWebhookConfigurationKind:
				hookBytes, err = m.modifyWebhook(resource, &aregv1.ValidatingWebhookConfiguration{}, webhookURL)
			}
			if err != nil {
				return nil, err
			}
			if hookBytes != nil {
				hookInterface, err := utils.RawK8sInterface(hookBytes)
				if err != nil {
					return nil, err
				}
				modifiedWebhooks = append(modifiedWebhooks, hookInterface)
			}
		}
	}
	return modifiedWebhooks, nil
}

func (m *Manifest) modifyWebhook(resource map[string]any, hook client.Object, webhookURL string) ([]byte, error) {
	resStr, err := utils.Stringy(resource)
	if err != nil {
		return nil, err
	}
	// convert yaml to aregv1.MutatingWebhookConfiguration{} or aregv1.ValidatingWebhookConfiguration{}
	err = utils.FromYamlToK8sObject(resStr, hook)
	if err != nil {
		return nil, err
	}
	switch modifiedHook := any(hook).(type) {
	case *aregv1.MutatingWebhookConfiguration:
		utils.Logf("modifying mutating webhook %s...", modifiedHook.Name)
		utils.Logf("setting webhook client config to %s...", webhookURL)
		for i, c := range modifiedHook.Webhooks {
			if c.ClientConfig.Service.Path != nil {
				url := "https://" + net.JoinHostPort(webhookURL, "9443") + *c.ClientConfig.Service.Path
				modifiedHook.Webhooks[i].ClientConfig.URL = utils.StringP(url)
				modifiedHook.Webhooks[i].ClientConfig.Service = nil
			}
			modifiedHook.Webhooks[i].TimeoutSeconds = utils.Int32P(30)
		}
		// convert from aregv1.MutatingWebhookConfiguration{} to yaml
		return utils.FromK8sObjectToYaml(modifiedHook, aregv1.SchemeGroupVersion)
	case *aregv1.ValidatingWebhookConfiguration:
		utils.Logf("modifying validating webhook %s...", modifiedHook.Name)
		utils.Logf("setting webhook client config to %s...", webhookURL)
		for i, c := range modifiedHook.Webhooks {
			if c.ClientConfig.Service.Path != nil {
				url := "https://" + net.JoinHostPort(webhookURL, "9443") + *c.ClientConfig.Service.Path
				modifiedHook.Webhooks[i].ClientConfig.URL = utils.StringP(url)
				modifiedHook.Webhooks[i].ClientConfig.Service = nil
			}
			modifiedHook.Webhooks[i].TimeoutSeconds = utils.Int32P(30)
		}
		// convert from aregv1.ValidatingWebhookConfiguration{} to yaml
		return utils.FromK8sObjectToYaml(modifiedHook, aregv1.SchemeGroupVersion)
	default:
		return nil, fmt.Errorf("unexpected webhook type: %T", hook)
	}
}

// buildAndLoadImage - builds the manager image as greenhouse/manager:local and loads it to the kind Cluster
func (m *Manifest) buildAndLoadImage(clusterName string) error {
	if !utils.CheckIfFileExists(m.Webhook.DockerFile) {
		return fmt.Errorf("docker file not found: %s", m.Webhook.DockerFile)
	}
	utils.Log("building greenhouse local development image...")
	err := klient.BuildImage(LocalDevIMG, utils.GetHostPlatform(), m.Webhook.DockerFile)
	if err != nil {
		return err
	}
	utils.Log("loading manager image to Cluster...")
	return klient.LoadImage(LocalDevIMG, clusterName)
}

func getContainerIndex(deployment *appsv1.Deployment, containerName string) int {
	for i, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name == containerName {
			return i
		}
	}
	return -1
}

func extractResourceByNameKind(resources []map[string]any, name, kind string) (map[string]any, error) {
	for _, resource := range resources {
		if k, ok := resource["kind"].(string); ok && k == kind {
			if n, ok := resource["metadata"].(map[string]any)["name"].(string); ok && n == name {
				return resource, nil
			}
		}
	}
	return nil, fmt.Errorf("resource not found: %s", name)
}

func extractResourcesByKinds(resources []map[string]any, kinds ...string) []map[string]any {
	extractedResources := make([]map[string]any, 0)
	for _, k := range kinds {
		resource := extractResourceByKind(resources, k)
		if resource != nil {
			extractedResources = append(extractedResources, resource)
		}
	}
	return extractedResources
}

func extractResourceByKind(resources []map[string]any, kind string) map[string]any {
	for _, resource := range resources {
		if k, ok := resource["kind"].(string); ok && k == kind {
			return resource
		}
	}
	return nil
}

func (m *Manifest) waitUntilDeploymentReady(ctx context.Context, clusterName, name, namespace string) error {
	cl, err := getKubeClient(clusterName)
	if err != nil {
		return err
	}
	b := utils.StandardBackoff()
	b.Reset()
	retries := 0
	maxRetries := 10
	op := func() (op bool, err error) {
		if retries >= maxRetries {
			err = backoff.Permanent(fmt.Errorf("resource %s did not become ready after %d retries", name, maxRetries))
			return
		}
		utils.Logf("waiting for deployment %s to be ready...", name)
		deployment := &appsv1.Deployment{}
		err = cl.Get(ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}, deployment)
		if err != nil {
			retries++
			return
		}
		if deployment.Status.Conditions == nil {
			retries++
			err = errors.New("deployment is not yet ready")
			return
		}
		available := false
		for _, condition := range deployment.Status.Conditions {
			if condition.Type == appsv1.DeploymentAvailable && condition.Status == v1.ConditionTrue {
				available = true
				break
			}
		}
		if !available {
			retries++
			err = errors.New("deployment is not yet ready")
			return
		}
		op = true
		return
	}
	_, err = backoff.Retry(ctx, op, backoff.WithBackOff(b))
	return err
}

func getKubeClient(clusterName string) (client.Client, error) {
	kubeconfig, err := klient.GetKubeCfg(clusterName, false)
	if err != nil {
		return nil, err
	}
	cl, err := klient.NewKubeClientFromConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	return cl, nil
}

// extractWebhookCerts - extracts the webhook cert secret generated by the cert job and writes them to tmp/k8s-webhook-server/serving-certs directory
func (m *Manifest) extractWebhookCerts(ctx context.Context, clusterName, namespace string) error {
	secName := m.ReleaseName + webhookCertSecSuffix
	cl, err := getKubeClient(clusterName)
	if err != nil {
		return err
	}
	if err = utils.WaitUntilSecretCreated(ctx, cl, secName, namespace); err != nil {
		return err
	}
	return writeCertsToTemp(ctx, cl, secName, namespace)
}

func writeCertsToTemp(ctx context.Context, cl client.Client, name, namespace string) error {
	secret := &v1.Secret{}
	err := cl.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, secret)
	if err != nil {
		return err
	}
	cert, ok := secret.Data["tls.crt"]
	if !ok {
		return fmt.Errorf("tls.crt not found in secret %s", name)
	}
	key, ok := secret.Data["tls.key"]
	if !ok {
		return fmt.Errorf("tls.key not found in secret %s", name)
	}
	dirPath := filepath.Join(os.TempDir(), "k8s-webhook-server", "serving-certs")
	err = utils.WriteToPath(dirPath, "tls.crt", string(cert))
	if err != nil {
		return err
	}
	err = utils.WriteToPath(dirPath, "tls.key", string(key))
	if err != nil {
		return err
	}
	utils.Logf("webhook certs written to %s", dirPath)
	return nil
}

func getWebhookURL() (webhookURL string) {
	switch runtime.GOOS {
	case "darwin":
		utils.Log("detected macOS...")
		webhookURL = "host.docker.internal"
	case "linux":
		utils.Log("detected linux...")
		webhookURL = strings.TrimSpace(getHostIPFromInterface())
	default:
		utils.Logf("detected %s ...", runtime.GOOS)
		webhookURL = "host.docker.internal"
	}
	return
}

// getHostIPFromInterface - returns the IP address of the docker0 interface (only for linux)
func getHostIPFromInterface() string {
	i, err := net.InterfaceByName("docker0")
	if err != nil {
		utils.LogErr("failed to get docker0 interface - %s", err.Error())
		return ""
	}
	addresses, err := i.Addrs()
	if err != nil {
		utils.LogErr("failed to get addresses for docker0 interface - %s", err.Error())
		return ""
	}
	for _, addr := range addresses {
		if ipv4 := addr.(*net.IPNet).IP.To4(); ipv4 != nil { //nolint:errcheck
			utils.Logf("found IP address for docker0 interface: %s", ipv4.String())
			return ipv4.String()
		}
	}
	utils.LogErr("failed to get IP address for docker0 interface")
	return ""
}
