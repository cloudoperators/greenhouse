// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package helm_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/test"
	webhookv1alpha1 "github.com/cloudoperators/greenhouse/internal/webhook/v1alpha1"
)

func TestHelm(t *testing.T) {
	RunSpecs(t, "Helm")
}

var _ = BeforeSuite(func() {
	test.RegisterWebhook("clusterPluginDefinitionWebhook", webhookv1alpha1.SetupClusterPluginDefinitionWebhookWithManager)
	test.RegisterWebhook("pluginWebhook", webhookv1alpha1.SetupPluginWebhookWithManager)
	test.RegisterWebhook("teamWebhook", webhookv1alpha1.SetupTeamWebhookWithManager)
	test.RegisterWebhook("secretsWebhook", webhookv1alpha1.SetupSecretWebhookWithManager)
	test.TestBeforeSuite()
})

var _ = AfterSuite(func() {
	test.TestAfterSuite()
})

var (
	namespace = "test-org"

	optionValue = &greenhousev1alpha1.PluginOptionValue{
		Name:  "key1",
		Value: test.MustReturnJSONFor("pluginValue1"),
	}
	secretOptionValue = &greenhousev1alpha1.PluginOptionValue{
		Name: "secretValue",
		ValueFrom: &greenhousev1alpha1.ValueFromSource{
			Secret: &greenhousev1alpha1.SecretKeyReference{
				Name: "plugindefinition-secret",
				Key:  "secretKey",
			},
		},
	}

	testPluginWithoutHelmChart = &greenhousev1alpha1.ClusterPluginDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "test-plugindefinition",
		},
		Spec: greenhousev1alpha1.PluginDefinitionSpec{
			Options: []greenhousev1alpha1.PluginOption{
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

	testPluginWithHelmChart = &greenhousev1alpha1.ClusterPluginDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "test-plugindefinition",
		},
		Spec: greenhousev1alpha1.PluginDefinitionSpec{
			Version: "1.0.0",
			HelmChart: &greenhousev1alpha1.HelmChartReference{
				Name:    "./../test/fixtures/myChart",
				Version: "1.0.0",
			},
			Options: []greenhousev1alpha1.PluginOption{
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

	testPluginWithHelmChartOCI = &greenhousev1alpha1.ClusterPluginDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "test-plugindefinition",
		},
		Spec: greenhousev1alpha1.PluginDefinitionSpec{
			HelmChart: &greenhousev1alpha1.HelmChartReference{
				Name:       "dummy",
				Repository: "oci://greenhouse/helm-charts",
				Version:    "1.0.0",
			},
			Options: []greenhousev1alpha1.PluginOption{
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

	testPluginWithHelmChartCRDs = &greenhousev1alpha1.ClusterPluginDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "test-plugindefinition",
		},
		Spec: greenhousev1alpha1.PluginDefinitionSpec{
			Version: "1.0.0",
			HelmChart: &greenhousev1alpha1.HelmChartReference{
				Name:    "./../test/fixtures/myChartWithCRDs",
				Version: "1.0.0",
			},
			Options: []greenhousev1alpha1.PluginOption{
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

	plugin = &greenhousev1alpha1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "test-plugin",
			// add owning team label
			Labels: map[string]string{
				greenhouseapis.LabelKeyOwnedBy: "test-team-1",
			},
		},
		Spec: greenhousev1alpha1.PluginSpec{
			PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
				Name: "test-plugindefinition",
				Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
			},
			ClusterName:      "test-cluster",
			OptionValues:     []greenhousev1alpha1.PluginOptionValue{},
			ReleaseNamespace: "test-release-namespace",
		},
	}

	pluginSecret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "plugindefinition-secret",
			Labels:    map[string]string{greenhouseapis.LabelKeyOwnedBy: "test-team-1"},
		},
		Data: map[string][]byte{
			"secretKey": []byte("pluginSecretValue1"),
		},
	}

	team = &greenhousev1alpha1.Team{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousev1alpha1.GroupVersion.Group,
			Kind:       "Team",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-team-1",
			Namespace: namespace,
		},
		Spec: greenhousev1alpha1.TeamSpec{
			Description:    "Test Team 1",
			MappedIDPGroup: "IDP_GROUP_NAME_MATCHING_TEAM_1",
		},
	}
)
