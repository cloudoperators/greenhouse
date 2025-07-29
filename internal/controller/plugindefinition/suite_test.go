// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugindefinition

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudoperators/greenhouse/internal/test"
	webhookv1alpha1 "github.com/cloudoperators/greenhouse/internal/webhook/v1alpha1"
)

func TestPluginDefinition(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PluginDefinitionControllerSuite")
}

var _ = BeforeSuite(func() {
	test.RegisterController("pluginDefinition", (&PluginDefinitionReconciler{}).SetupWithManager)
	test.RegisterController("clusterPluginDefinition", (&ClusterPluginDefinitionReconciler{}).SetupWithManager)
	test.RegisterWebhook("pluginDefinitionWebhook", webhookv1alpha1.SetupPluginDefinitionWebhookWithManager)
	test.RegisterWebhook("clusterPluginDefinitionWebhook", webhookv1alpha1.SetupClusterPluginDefinitionWebhookWithManager)
	test.TestBeforeSuite()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment and remote cluster")
	test.TestAfterSuite()
})
