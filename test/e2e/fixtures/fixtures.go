// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package fixtures

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

var NginxPluginDefinition = &greenhousev1alpha1.PluginDefinition{
	TypeMeta: metav1.TypeMeta{
		Kind:       "PluginDefinition",
		APIVersion: greenhousev1alpha1.GroupVersion.String(),
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "nginx-18.1.7",
		Namespace: test.TestNamespace,
	},
	Spec: greenhousev1alpha1.PluginDefinitionSpec{
		Description: "nginx",
		Version:     "18.1.7",
		HelmChart: &greenhousev1alpha1.HelmChartReference{
			Name:       "bitnamicharts/nginx",
			Repository: "oci://registry-1.docker.io",
			Version:    "18.1.7",
		},
		Options: []greenhousev1alpha1.PluginOption{
			{
				Default:     &apiextensionsv1.JSON{Raw: []byte("false")},
				Description: "autoscaling.enabled",
				Name:        "autoscaling.enabled",
				Type:        "bool",
			},
			{
				Default:     &apiextensionsv1.JSON{Raw: []byte("\"\"")},
				Description: "autoscaling.maxReplicas",
				Name:        "autoscaling.maxReplicas",
				Type:        "string",
			},
			{
				Default:     &apiextensionsv1.JSON{Raw: []byte("\"\"")},
				Description: "autoscaling.minReplicas",
				Name:        "autoscaling.minReplicas",
				Type:        "string",
			},
			{
				Default:     &apiextensionsv1.JSON{Raw: []byte("true")},
				Description: "containerSecurityContext.enabled",
				Name:        "containerSecurityContext.enabled",
				Type:        "bool",
			},
			{
				Default:     &apiextensionsv1.JSON{Raw: []byte("1")},
				Description: "replicaCount",
				Name:        "replicaCount",
				Type:        "int",
			},
			{
				Default:     &apiextensionsv1.JSON{Raw: []byte("true")},
				Description: "podSecurityContext.enabled",
				Name:        "podSecurityContext.enabled",
				Type:        "bool",
			},
		},
		UIApplication: &greenhousev1alpha1.UIApplicationReference{
			Name:    "nginx",
			URL:     "TODO: Javascript asset server URL.",
			Version: "latest",
		},
	},
}
