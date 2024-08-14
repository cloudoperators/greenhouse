package manifests

import (
	"fmt"
	"github.com/cloudoperators/greenhouse/hack/localenv/pkg/kind"
	"github.com/cloudoperators/greenhouse/hack/localenv/pkg/klient"
	"github.com/cloudoperators/greenhouse/hack/localenv/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	"strings"
)

type Webhook struct {
	Envs       []WebhookEnv `json:"envs"`
	DockerFile string       `json:"dockerFile"`
}

type WebhookEnv struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

const (
	MangerIMG       = "greenhouse/manager:local"
	MangerContainer = "manager"
)

func (m *manifests) SetupWebhookManifest(resources []map[string]interface{}) (map[string]interface{}, error) {
	clusterName := strings.TrimSpace(m.hc.GetClusterName())
	if clusterName == "" {
		utils.Log("no cluster name provided, skipping webhook setup")
		return nil, nil
	}

	if !utils.CheckIfFileExists(m.webhook.DockerFile) {
		return nil, fmt.Errorf("docker file not found: %s", m.webhook.DockerFile)
	}
	err := klient.BuildImage(MangerIMG, utils.GetHostPlatform(), m.webhook.DockerFile)
	if err != nil {
		return nil, err
	}
	err = kind.LoadImage(MangerIMG, clusterName)
	if err != nil {
		return nil, err
	}
	deployment := filterResourcesBy(resources, "Deployment")
	if len(deployment) == 0 {
		return nil, fmt.Errorf("no deployment found in resources")
	}
	manifest, err := m.modifyDeployment(deployment[0])
	if err != nil {
		return nil, err
	}
	return utils.RawK8sInterface(manifest)
}

func (m *manifests) modifyDeployment(deploymentResource map[string]interface{}) ([]byte, error) {
	deployment := &appsv1.Deployment{}
	deploymentStr, err := utils.Stringy(deploymentResource)
	if err != nil {
		return nil, err
	}
	err = utils.FromYamlToK8sObject(deploymentStr, deployment)
	if err != nil {
		return nil, err
	}
	index := getManagerContainerIndex(deployment)
	if index == -1 {
		return nil, fmt.Errorf("manager container not found in deployment")
	}
	for _, env := range m.webhook.Envs {
		deployment.Spec.Template.Spec.Containers[index].Env = append(deployment.Spec.Template.Spec.Containers[index].Env, v1.EnvVar{
			Name:  env.Name,
			Value: env.Value,
		})
	}
	deployment.Spec.Template.Spec.Containers[index].Image = MangerIMG
	deployment.Spec.Replicas = utils.Int32P(1)
	return utils.FromK8sObjectToYaml(deployment, appsv1.SchemeGroupVersion)
}

func getManagerContainerIndex(deployment *appsv1.Deployment) int {
	for i, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name == MangerContainer {
			return i
		}
	}
	return -1
}
