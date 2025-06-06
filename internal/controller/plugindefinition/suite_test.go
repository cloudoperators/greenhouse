// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugindefinition

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudoperators/greenhouse/internal/test"
	admission "github.com/cloudoperators/greenhouse/internal/webhook"
)

func TestPluginDefinition(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PluginDefinitionControllerSuite")
}

var _ = BeforeSuite(func() {
	test.RegisterController("cluster", (&PluginDefinitionReconciler{}).SetupWithManager)
	test.RegisterWebhook("pluginDefinitionWebhook", admission.SetupPluginDefinitionWebhookWithManager)
	test.RegisterWebhook("clusterPluginDefinitionWebhook", admission.SetupClusterPluginDefinitionWebhookWithManager)
	test.TestBeforeSuite()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment and remote cluster")
	test.TestAfterSuite()
})
