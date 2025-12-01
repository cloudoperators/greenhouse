// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin_integration

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	greenhousecluster "github.com/cloudoperators/greenhouse/internal/controller/cluster"
	"github.com/cloudoperators/greenhouse/internal/controller/plugin"
	greenhouseDef "github.com/cloudoperators/greenhouse/internal/controller/plugindefinition"
	"github.com/cloudoperators/greenhouse/internal/test"
	webhookv1alpha1 "github.com/cloudoperators/greenhouse/internal/webhook/v1alpha1"
)

func TestPluginIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PluginIntegrationSuite")
}

var _ = BeforeSuite(func() {
	test.RegisterController("plugin", (&plugin.PluginReconciler{
		IntegrationEnabled:    true, // Enable integration features for tracking
		DefaultDeploymentTool: ptr.To("flux"),
	}).SetupWithManager)
	test.RegisterWebhook("organizationWebhook", webhookv1alpha1.SetupOrganizationWebhookWithManager)
	test.RegisterController("clusterPluginDefinition", (&greenhouseDef.ClusterPluginDefinitionReconciler{}).SetupWithManager)
	test.RegisterController("cluster", (&greenhousecluster.RemoteClusterReconciler{}).SetupWithManager)
	test.RegisterWebhook("clusterPluginDefinitionWebhook", webhookv1alpha1.SetupClusterPluginDefinitionWebhookWithManager)
	test.RegisterWebhook("pluginWebhook", webhookv1alpha1.SetupPluginWebhookWithManager)
	test.TestBeforeSuite()

	// return the test.Cfg, as the in-cluster config is not available
	ctrl.GetConfig = func() (*rest.Config, error) {
		return test.Cfg, nil
	}
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	test.TestAfterSuite()
})
