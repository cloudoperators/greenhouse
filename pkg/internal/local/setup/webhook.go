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
	"time"

	"k8s.io/utils/ptr"

	"github.com/cenkalti/backoff/v5"

	aregv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloudoperators/greenhouse/pkg/internal/local/klient"
	"github.com/cloudoperators/greenhouse/pkg/internal/local/utils"
)

type Webhook struct {
	Envs       []WebhookEnv `yaml:"envs" json:"envs"`
	DockerFile string       `yaml:"dockerFile" json:"dockerFile"`
	DevMode    bool         `yaml:"devMode" json:"devMode"`
}

type WebhookEnv struct {
	Name  string `yaml:"name" json:"name"`
	Value string `yaml:"value" json:"value"`
}

const (
	MangerIMG                          = "greenhouse/manager:local"
	MangerContainer                    = "manager"
	DeploymentKind                     = "Deployment"
	ManagerDeploymentNameSuffix        = "-controller-manager"
	JobKind                            = "Job"
	JobNameSuffix                      = "-kube-webhook-certgen"
	MutatingWebhookConfigurationKind   = "MutatingWebhookConfiguration"
	ValidatingWebhookConfigurationKind = "ValidatingWebhookConfiguration"
	webhookCertSecSuffix               = "-webhook-server-cert"
)

// setupWebhookManifest - sets up the webhook manifest by modifying the manager deployment, cert job and webhook configurations
// deploys manager in WEBHOOK_ONLY mode so that you don't need to run webhooks locally during controller development
// modifies cert job (charts/manager/templates/kube-webhook-certgen.yaml) to include host.docker.internal
// if devMode is enabled, modifies mutating and validating webhook configurations to use host.docker.internal URL and removes service from clientConfig
// extracts the webhook certs from the secret and writes them to tmp/k8s-webhook-server/serving-certs directory
func (m *Manifest) setupWebhookManifest(resources []map[string]interface{}, clusterName string) ([]map[string]interface{}, error) {
	webhookManifests := make([]map[string]interface{}, 0)
	releaseName := m.ReleaseName
	managerDeployment, err := extractResourceByNameKind(resources, releaseName+ManagerDeploymentNameSuffix, DeploymentKind)
	if err != nil {
		return nil, err
	}

	utils.Log("modifying manager deployment...")
	managerDeployment, err = m.modifyManagerDeployment(managerDeployment)
	if err != nil {
		return nil, err
	}

	certJob, err := extractResourceByNameKind(resources, releaseName+JobNameSuffix, JobKind)
	if err != nil {
		return nil, err
	}
	utils.Log("modifying cert job...")
	webhookURL := getWebhookURL()
	certJob, err = m.modifyCertJob(certJob, webhookURL)
	if err != nil {
		return nil, err
	}

	webhookManifests = append(webhookManifests, managerDeployment, certJob)
	webhookResources := extractResourcesByKinds(resources, MutatingWebhookConfigurationKind, ValidatingWebhookConfigurationKind)
	if m.Webhook.DevMode {
		utils.Log("enabling webhook local development...")
		if webhookURL != "" {
			webhooks, err := m.modifyWebhooks(webhookResources, webhookURL)
			if err != nil {
				return nil, err
			}
			if len(webhooks) > 0 {
				webhookManifests = append(webhookManifests, webhooks...)
			}
		}
	} else {
		webhookManifests = append(webhookManifests, webhookResources...)
	}

	err = m.buildAndLoadImage(clusterName)
	if err != nil {
		return nil, err
	}
	return webhookManifests, nil
}

func (m *Manifest) setHostPathVolume(deployment *appsv1.Deployment) {
	hostVolume := v1.Volume{
		Name: "plugin",
		VolumeSource: v1.VolumeSource{
			HostPath: &v1.HostPathVolumeSource{
				Path: utils.PluginHostPath,
				Type: ptr.To[v1.HostPathType](v1.HostPathDirectory),
			},
		},
	}
	if len(deployment.Spec.Template.Spec.Volumes) == 0 {
		deployment.Spec.Template.Spec.Volumes = []v1.Volume{hostVolume}
	} else {
		deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, hostVolume)
	}
}

func (m *Manifest) setHostPathVolumeMount(containerIndex int, deployment *appsv1.Deployment) {
	hostMount := v1.VolumeMount{
		Name:      "plugin",
		MountPath: utils.ManagerHostPathMount,
	}
	if len(deployment.Spec.Template.Spec.Containers[containerIndex].VolumeMounts) == 0 {
		deployment.Spec.Template.Spec.Containers[containerIndex].VolumeMounts = []v1.VolumeMount{hostMount}
	} else {
		deployment.Spec.Template.Spec.Containers[containerIndex].VolumeMounts = append(deployment.Spec.Template.Spec.Containers[containerIndex].VolumeMounts, hostMount)
	}
}

// modifyManagerDeployment - appends the env in manager container by setting WEBHOOK_ONLY=true
func (m *Manifest) modifyManagerDeployment(deploymentResource map[string]interface{}) (map[string]interface{}, error) {
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
	index := getContainerIndex(deployment, MangerContainer)
	if index == -1 {
		return nil, errors.New("manager container not found in deployment")
	}
	for _, e := range m.Webhook.Envs {
		deployment.Spec.Template.Spec.Containers[index].Env = append(deployment.Spec.Template.Spec.Containers[index].Env, v1.EnvVar{
			Name:  e.Name,
			Value: e.Value,
		})
	}
	deployment.Spec.Template.Spec.Containers[index].Image = MangerIMG
	deployment.Spec.Replicas = utils.Int32P(1)
	if m.enableLocalPluginDev {
		m.setHostPathVolume(deployment)
		m.setHostPathVolumeMount(index, deployment)
		deployment.Spec.Template.Spec.Containers[index].SecurityContext.RunAsGroup = ptr.To[int64](65532)
	}

	depBytes, err := utils.FromK8sObjectToYaml(deployment, appsv1.SchemeGroupVersion)
	if err != nil {
		return nil, err
	}
	return utils.RawK8sInterface(depBytes)
}

// modifyWebhooks - modifies the webhook configurations to use host.docker.internal URL and removes service from clientConfig
// during local development of webhooks api server will forward the request to host machine where the webhook is running at port 9443
func (m *Manifest) modifyWebhooks(resources []map[string]interface{}, webhookURL string) ([]map[string]interface{}, error) {
	modifiedWebhooks := make([]map[string]interface{}, 0)
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

func (m *Manifest) modifyWebhook(resource map[string]interface{}, hook client.Object, webhookURL string) ([]byte, error) {
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

// modifyCertJob - appends host.docker.internal to the args in cert job
// certs generated are valid only for a set of defined DNS names, adding host.docker.internal to hosts will prevent TLS errors
func (m *Manifest) modifyCertJob(resources map[string]interface{}, webhookURL string) (map[string]interface{}, error) {
	job := &batchv1.Job{}
	jobStr, err := utils.Stringy(resources)
	if err != nil {
		return nil, err
	}
	err = utils.FromYamlToK8sObject(jobStr, job)
	if err != nil {
		return nil, err
	}
	args := job.Spec.Template.Spec.InitContainers[0].Args
	for i, arg := range args {
		if strings.Contains(arg, "host") {
			args[i] = fmt.Sprintf("%s,%s", arg, webhookURL)
		}
	}
	job.Spec.Template.Spec.InitContainers[0].Args = args
	jobBytes, err := utils.FromK8sObjectToYaml(job, batchv1.SchemeGroupVersion)
	if err != nil {
		return nil, err
	}
	return utils.RawK8sInterface(jobBytes)
}

// buildAndLoadImage - builds the manager image as greenhouse/manager:local and loads it to the kind Cluster
func (m *Manifest) buildAndLoadImage(clusterName string) error {
	if !utils.CheckIfFileExists(m.Webhook.DockerFile) {
		return fmt.Errorf("docker file not found: %s", m.Webhook.DockerFile)
	}
	utils.Log("building manager image...")
	err := klient.BuildImage(MangerIMG, utils.GetHostPlatform(), m.Webhook.DockerFile)
	if err != nil {
		return err
	}
	utils.Log("loading manager image to Cluster...")
	return klient.LoadImage(MangerIMG, clusterName)
}

func getContainerIndex(deployment *appsv1.Deployment, containerName string) int {
	for i, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name == containerName {
			return i
		}
	}
	return -1
}

func extractResourceByNameKind(resources []map[string]interface{}, name, kind string) (map[string]interface{}, error) {
	for _, resource := range resources {
		if k, ok := resource["kind"].(string); ok && k == kind {
			if n, ok := resource["metadata"].(map[string]interface{})["name"].(string); ok && n == name {
				return resource, nil
			}
		}
	}
	return nil, fmt.Errorf("resource not found: %s", name)
}

func extractResourcesByKinds(resources []map[string]interface{}, kinds ...string) []map[string]interface{} {
	extractedResources := make([]map[string]interface{}, 0)
	for _, k := range kinds {
		resource := extractResourceByKind(resources, k)
		if resource != nil {
			extractedResources = append(extractedResources, resource)
		}
	}
	return extractedResources
}

func extractResourceByKind(resources []map[string]interface{}, kind string) map[string]interface{} {
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
	b := &backoff.ExponentialBackOff{
		InitialInterval:     500 * time.Millisecond, // Start with 500ms delay
		RandomizationFactor: 0.5,                    // Randomize interval by ±50%
		Multiplier:          2.0,                    // Double the interval each time
		MaxInterval:         15 * time.Second,       // Cap at 15s between retries
	}
	b.Reset()
	retries := 0
	maxRetries := 10
	op := func() (op bool, err error) {
		if retries >= maxRetries {
			err = backoff.Permanent(fmt.Errorf("resource %s did not become ready after %d retries", name, maxRetries))
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
	jobName := m.ReleaseName + JobNameSuffix
	secName := m.ReleaseName + webhookCertSecSuffix
	cl, err := getKubeClient(clusterName)
	if err != nil {
		return err
	}

	if err = utils.WaitUntilJobSucceeds(ctx, cl, jobName, namespace); err != nil {
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

func getWebhookURL() string {
	switch runtime.GOOS {
	case "darwin":
		utils.Log("detected macOS...")
		return "host.docker.internal"
	case "linux":
		utils.Log("detected linux...")
		return strings.TrimSpace(getHostIPFromInterface())
	default:
		utils.Logf("detected %s ...", runtime.GOOS)
		return "host.docker.internal"
	}
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

func (w *Webhook) AddOrOverrideEnv(envs []string) {
	// Convert the input array of key=value strings into WebhookEnv objects
	for _, env := range envs {
		// Split the string into key and value
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			// Skip invalid key=value strings
			continue
		}
		key := parts[0]
		value := parts[1]

		// Check if the environment variable already exists in w.Envs
		found := false
		for i, existingEnv := range w.Envs {
			if existingEnv.Name == key {
				// Override the value if the key matches
				w.Envs[i].Value = value
				found = true
				break
			}
		}

		// If not found, add the new environment variable
		if !found {
			w.Envs = append(w.Envs, WebhookEnv{Name: key, Value: value})
		}
	}
}
