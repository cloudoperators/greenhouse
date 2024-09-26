// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package helm_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cloudoperators/greenhouse/pkg/admission"
	greenhousesapv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

func TestHelm(t *testing.T) {
	RunSpecs(t, "Helm")
}

var _ = BeforeSuite(func() {
	test.RegisterWebhook("pluginDefinitionWebhook", admission.SetupPluginDefinitionWebhookWithManager)
	test.RegisterWebhook("pluginWebhook", admission.SetupPluginWebhookWithManager)
	test.RegisterWebhook("teamWebhook", admission.SetupTeamWebhookWithManager)
	test.RegisterWebhook("secretsWebhook", admission.SetupSecretWebhookWithManager)
	test.TestBeforeSuite()
})

var _ = AfterSuite(func() {
	test.TestAfterSuite()
})

var (
	namespace = "test-org"

	optionValue = &greenhousesapv1alpha1.PluginOptionValue{
		Name:  "key1",
		Value: test.MustReturnJSONFor("pluginValue1"),
	}
	secretOptionValue = &greenhousesapv1alpha1.PluginOptionValue{
		Name: "secretValue",
		ValueFrom: &greenhousesapv1alpha1.ValueFromSource{
			Secret: &greenhousesapv1alpha1.SecretKeyReference{
				Name: "plugindefinition-secret",
				Key:  "secretKey",
			},
		},
	}

	testPluginWithoutHelmChart = &greenhousesapv1alpha1.PluginDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-org",
			Name:      "test-plugindefinition",
		},
		Spec: greenhousesapv1alpha1.PluginDefinitionSpec{
			Options: []greenhousesapv1alpha1.PluginOption{
				{
					Name:        "key1",
					Description: "key1 description",
					Required:    true,
					Default:     test.MustReturnJSONFor("defaultKey1"),
					Type:        "string",
				},
			},
		},
	}

	testPluginWithHelmChart = &greenhousesapv1alpha1.PluginDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-org",
			Name:      "test-plugindefinition",
		},
		Spec: greenhousesapv1alpha1.PluginDefinitionSpec{
			Version: "1.0.0",
			HelmChart: &greenhousesapv1alpha1.HelmChartReference{
				Name:       "./../test/fixtures/myChart",
				Repository: "dummy",
				Version:    "1.0.0",
			},
			Options: []greenhousesapv1alpha1.PluginOption{
				{
					Name:        "key1",
					Description: "key1 description",
					Required:    true,
					Default:     test.MustReturnJSONFor("defaultKey1"),
					Type:        "string",
				},
			},
		},
	}

	testPluginWithHelmChartOCI = &greenhousesapv1alpha1.PluginDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-org",
			Name:      "test-plugindefinition",
		},
		Spec: greenhousesapv1alpha1.PluginDefinitionSpec{
			HelmChart: &greenhousesapv1alpha1.HelmChartReference{
				Name:       "dummy",
				Repository: "oci://greenhouse/helm-charts",
				Version:    "1.0.0",
			},
			Options: []greenhousesapv1alpha1.PluginOption{
				{
					Name:        "key1",
					Description: "key1 description",
					Required:    true,
					Default:     test.MustReturnJSONFor("defaultKey1"),
					Type:        "string",
				},
			},
		},
	}

	testPluginWithHelmChartCRDs = &greenhousesapv1alpha1.PluginDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-org",
			Name:      "test-plugindefinition",
		},
		Spec: greenhousesapv1alpha1.PluginDefinitionSpec{
			Version: "1.0.0",
			HelmChart: &greenhousesapv1alpha1.HelmChartReference{
				Name:       "./../test/fixtures/myChartWithCRDs",
				Repository: "dummy",
				Version:    "1.0.0",
			},
			Options: []greenhousesapv1alpha1.PluginOption{
				{
					Name:        "key1",
					Description: "key1 description",
					Required:    true,
					Default:     test.MustReturnJSONFor("defaultKey1"),
					Type:        "string",
				},
			},
		},
	}

	plugin = &greenhousesapv1alpha1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-org",
			Name:      "test-plugin",
		},
		Spec: greenhousesapv1alpha1.PluginSpec{
			PluginDefinition: "test-plugindefinition",
			ClusterName:      "test-cluster",
			OptionValues:     []greenhousesapv1alpha1.PluginOptionValue{},
			ReleaseNamespace: "test-release-namespace",
		},
	}

	pluginSecret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-org",
			Name:      "plugindefinition-secret",
		},
		Data: map[string][]byte{
			"secretKey": []byte("pluginSecretValue1"),
		},
	}

	team = &greenhousesapv1alpha1.Team{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousesapv1alpha1.GroupVersion.Group,
			Kind:       "Team",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-team-1",
			Namespace: "test-org",
		},
		Spec: greenhousesapv1alpha1.TeamSpec{
			Description:    "Test Team 1",
			MappedIDPGroup: "IDP_GROUP_NAME_MATCHING_TEAM_1",
		},
	}
)
