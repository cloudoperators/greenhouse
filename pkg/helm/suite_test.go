// Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	test.RegisterWebhook("pluginWebhook", admission.SetupPluginWebhookWithManager)
	test.RegisterWebhook("pluginConfigWebhook", admission.SetupPluginConfigWebhookWithManager)
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
				Name: "plugin-secret",
				Key:  "secretKey",
			},
		},
	}

	testPluginWithoutHelmChart = &greenhousesapv1alpha1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-org",
			Name:      "test-plugin",
		},
		Spec: greenhousesapv1alpha1.PluginSpec{
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

	testPluginWithHelmChart = &greenhousesapv1alpha1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-org",
			Name:      "test-plugin",
		},
		Spec: greenhousesapv1alpha1.PluginSpec{
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

	testPluginWithHelmChartOCI = &greenhousesapv1alpha1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-org",
			Name:      "test-plugin",
		},
		Spec: greenhousesapv1alpha1.PluginSpec{
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

	pluginConfig = &greenhousesapv1alpha1.PluginConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-org",
			Name:      "test-plugin-config",
		},
		Spec: greenhousesapv1alpha1.PluginConfigSpec{
			Plugin:       "test-plugin",
			ClusterName:  "test-cluster",
			OptionValues: []greenhousesapv1alpha1.PluginOptionValue{},
		},
	}

	pluginSecret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-org",
			Name:      "plugin-secret",
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
