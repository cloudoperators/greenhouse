// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloudoperators/greenhouse/pkg/test"
)

var (
	remoteCfg        *rest.Config
	remoteKubeConfig []byte
	remoteClient     client.Client
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2ESuite")
}

var _ = BeforeSuite(func() {
	// Register all known controllers and webhooks if we run the e2e tests locally
	// Register controllers.
	for controllerName, hookFunc := range knownControllers {
		test.RegisterController(controllerName, hookFunc)
	}
	// register webhooks
	for webhookName, hookFunc := range knownWebhooks {
		test.RegisterWebhook(webhookName, hookFunc)
	}

	if !test.IsUseExistingCluster {
		remoteCfg, remoteClient, _, remoteKubeConfig = test.StartControlPlane("6885", true, false)
	} else {
		remoteCfg = test.Cfg
		remoteClient = test.K8sClient
		remoteKubeConfig = test.KubeConfig
	}

	test.TestBeforeSuite()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	test.TestAfterSuite()
})
