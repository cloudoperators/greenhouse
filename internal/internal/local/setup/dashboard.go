// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package setup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudoperators/greenhouse/pkg/clientutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloudoperators/greenhouse/pkg/internal/local/utils"
)

const (
	ConfigMapKind             = "ConfigMap"
	ServiceKind               = "Service"
	DashboardContainer        = "dashboard"
	DashboardIMG              = "ghcr.io/cloudoperators/juno-app-greenhouse:latest"
	dashboardSA               = "greenhouse-demo-service-account"
	dashboardSAToken          = "greenhouse-demo-service-account-token"
	dashboardCRB              = "greenhouse-demo-cluster-role-binding"
	dashboardCMPropsKey       = "appProps.json"
	dashboardDeploymentSuffix = "-dashboard"
	dashboardConfigMapSuffix  = "-dashboard-app-props"
	apiEndpointKey            = "apiEndpoint"
	mockAuthKey               = "mockAuth"
	demoUserTokenKey          = "demoUserToken"
	demoOrgKey                = "demoOrg"
	apiProxyURL               = "http://127.0.0.1:9090"
)

func createServiceAccount(ctx context.Context, cl client.Client, name, namespace string) error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	_, err := clientutil.CreateOrPatch(ctx, cl, sa, func() error {
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func createServiceAccountSecret(ctx context.Context, cl client.Client, name, serviceAccountName, namespace string) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				"kubernetes.io/service-account.name": serviceAccountName,
			},
		},
		Type: corev1.SecretTypeServiceAccountToken,
	}
	_, err := clientutil.CreateOrPatch(ctx, cl, secret, func() error {
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func assignClusterAdminRole(ctx context.Context, cl client.Client, serviceAccountName, namespace string) error {
	binding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: dashboardCRB,
		},
		Subjects: []rbacv1.Subject{{Kind: rbacv1.ServiceAccountKind, Name: serviceAccountName, Namespace: namespace}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
	}
	_, err := clientutil.CreateOrPatch(ctx, cl, binding, func() error {
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (m *Manifest) setupDashboard(ctx context.Context, clusterName, namespace string) ([]map[string]interface{}, error) {
	cl, err := getKubeClient(clusterName)
	if err != nil {
		return nil, err
	}
	err = createServiceAccount(ctx, cl, dashboardSA, namespace)
	if err != nil {
		return nil, err
	}
	err = createServiceAccountSecret(ctx, cl, dashboardSAToken, dashboardSA, namespace)
	if err != nil {
		return nil, err
	}
	err = assignClusterAdminRole(ctx, cl, dashboardSA, namespace)
	if err != nil {
		return nil, err
	}

	dashboardResources := make([]map[string]interface{}, 0)
	resources, err := m.generateManifests(ctx)
	if err != nil {
		return nil, err
	}
	releaseName := m.ReleaseName

	dashboardDeployment, err := extractResourceByNameKind(resources, releaseName+dashboardDeploymentSuffix, DeploymentKind)
	if err != nil {
		return nil, err
	}
	dashboardDeployment, err = m.modifyDashboardDeployment(dashboardDeployment)
	if err != nil {
		return nil, err
	}

	dashboardConfigMap, err := extractResourceByNameKind(resources, releaseName+dashboardConfigMapSuffix, ConfigMapKind)
	if err != nil {
		return nil, err
	}
	dashboardConfigMap, err = m.modifyDashboardProps(ctx, cl, dashboardConfigMap, namespace, dashboardCMPropsKey, dashboardSAToken)
	if err != nil {
		return nil, err
	}

	kService, err := extractResourceByNameKind(resources, releaseName+dashboardDeploymentSuffix, ServiceKind)
	if err != nil {
		return nil, err
	}
	dashboardResources = append(dashboardResources, dashboardDeployment, dashboardConfigMap, kService)
	return dashboardResources, nil
}

func (m *Manifest) modifyDashboardDeployment(deploymentResource map[string]interface{}) (map[string]interface{}, error) {
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
	index := getContainerIndex(deployment, DashboardContainer)
	if index == -1 {
		return nil, errors.New("dashboard container not found in deployment")
	}
	deployment.Spec.Template.Spec.Containers[index].Image = DashboardIMG
	depBytes, err := utils.FromK8sObjectToYaml(deployment, appsv1.SchemeGroupVersion)
	if err != nil {
		return nil, err
	}
	return utils.RawK8sInterface(depBytes)
}

func (m *Manifest) modifyDashboardProps(ctx context.Context, cl client.Client, configMapResource map[string]interface{}, namespace, propsKey, tokenSecretName string) (map[string]interface{}, error) {
	configMap := &corev1.ConfigMap{}
	configMapStr, err := utils.Stringy(configMapResource)
	if err != nil {
		return nil, err
	}
	err = utils.FromYamlToK8sObject(configMapStr, configMap)
	if err != nil {
		return nil, err
	}

	props := map[string]string{}
	propStr := configMap.Data[propsKey]
	err = json.Unmarshal([]byte(propStr), &props)
	if err != nil {
		return nil, err
	}

	secret := &corev1.Secret{}
	err = cl.Get(ctx, types.NamespacedName{Name: tokenSecretName, Namespace: namespace}, secret)
	if err != nil {
		return nil, err
	}

	token := string(secret.Data["token"])
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("empty token found in service account secret")
	}
	props[apiEndpointKey] = apiProxyURL
	props[mockAuthKey] = "true"
	props[demoUserTokenKey] = token
	props[demoOrgKey] = "demo"

	propsBytes, err := json.Marshal(props)
	if err != nil {
		return nil, err
	}
	configMap.Data = map[string]string{propsKey: string(propsBytes)}
	configMapBytes, err := utils.FromK8sObjectToYaml(configMap, corev1.SchemeGroupVersion)
	if err != nil {
		return nil, err
	}

	return utils.RawK8sInterface(configMapBytes)
}

func (m *Manifest) getDashboardSetupInfo() string {
	proxyForward := fmt.Sprintf("port-forward the cors-proxy service: kubectl port-forward svc/%s-cors-proxy 9090:80 -n greenhouse (use & at the end to re-use terminal)", m.ReleaseName)
	dashboardForward := fmt.Sprintf("port-forward the dashboard service: kubectl port-forward svc/%s-dashboard <LOCAL_PORT>:80 -n greenhouse (use & at the end to re-use terminal)", m.ReleaseName)
	dashboardInfo := "access the dashboard at http://localhost:<LOCAL_PORT> (use the port you used in the dashboard port-forward command)"
	killInfo := "stop port-forwarding with CTRL+C or CMD+C, if you are using & at the end, do lsof -i :<PORT> and then kill -9 <PID>"
	return fmt.Sprintf("dashboard setup complete... \n-- %s \n-- %s\n-- %s\n-- %s", proxyForward, dashboardForward, dashboardInfo, killInfo)
}
