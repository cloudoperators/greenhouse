// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package flux

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/controller/cluster"
	"github.com/cloudoperators/greenhouse/internal/controller/plugin"
	"github.com/cloudoperators/greenhouse/internal/controller/plugindefinition"
	"github.com/cloudoperators/greenhouse/internal/test"
	webhookv1alpha1 "github.com/cloudoperators/greenhouse/internal/webhook/v1alpha1"
)

func TestTeamController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Team Suite")
}

var _ = BeforeSuite(func() {
	test.RegisterController("pluginFlux", (&FluxReconciler{KubeRuntimeOpts: clientutil.RuntimeOptions{QPS: 5, Burst: 10}}).SetupWithManager)
	test.RegisterController("pluginPreset", (&plugin.PluginPresetReconciler{}).SetupWithManager)
	test.RegisterController("pluginDefinition", (&plugindefinition.PluginDefinitionReconciler{}).SetupWithManager)
	test.RegisterController("clusterPluginDefinition", (&plugindefinition.ClusterPluginDefinitionReconciler{}).SetupWithManager)
	test.RegisterController("cluster", (&cluster.RemoteClusterReconciler{}).SetupWithManager)
	test.RegisterWebhook("teamWebhook", webhookv1alpha1.SetupTeamWebhookWithManager)
	test.RegisterWebhook("clusterPluginDefinitionWebhook", webhookv1alpha1.SetupClusterPluginDefinitionWebhookWithManager)
	test.RegisterWebhook("pluginWebhook", webhookv1alpha1.SetupPluginWebhookWithManager)
	test.RegisterWebhook("clusterWebhook", webhookv1alpha1.SetupClusterWebhookWithManager)
	test.RegisterWebhook("secretsWebhook", webhookv1alpha1.SetupSecretWebhookWithManager)
	test.RegisterWebhook("pluginPresetWebhook", webhookv1alpha1.SetupPluginPresetWebhookWithManager)
	test.TestBeforeSuite()

	// return the test.Cfg, as the in-cluster config is not available
	ctrl.GetConfig = func() (*rest.Config, error) {
		return test.Cfg, nil
	}
})
