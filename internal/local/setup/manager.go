// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package setup

import (
	"errors"
	"os"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	"github.com/cloudoperators/greenhouse/internal/local/utils"
)

const (
	ControllerEnabledEnvVar = "CONTROLLER_ENABLED"
)

const (
	ManagerContainer            = "manager"
	ManagerDeploymentNameSuffix = "-controller-manager"
)

func isControllerEnabled() (bool, error) {
	enabled, ok := os.LookupEnv(ControllerEnabledEnvVar)
	if !ok {
		return false, nil
	}
	return strconv.ParseBool(enabled)
}

func (m *Manifest) setupManagerManifest(resources []map[string]any) ([]map[string]any, error) {
	manifests := make([]map[string]any, 0)
	enabled, err := isControllerEnabled()
	if err != nil {
		return nil, err
	}
	if enabled {
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
		manifests = append(manifests, managerDeployment)
	}
	return manifests, nil
}

// modifyManagerDeployment - sets the manager image to the local dev image and sets hostPath volume mounts if needed
func (m *Manifest) modifyManagerDeployment(deploymentResource map[string]any) (map[string]any, error) {
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
	index := getContainerIndex(deployment, ManagerContainer)
	if index == -1 {
		return nil, errors.New("manager container not found in deployment")
	}
	deployment.Spec.Template.Spec.Containers[index].Image = LocalDevIMG
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
