// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudoperators/greenhouse/pkg/admission"
	clusterpkg "github.com/cloudoperators/greenhouse/pkg/controllers/cluster"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

var (
	bootstrapReconciler *clusterpkg.BootstrapReconciler
)

func TestClusterBootstrap(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ClusterControllerSuite")
}

var _ = BeforeSuite(func() {

	bootstrapReconciler = &clusterpkg.BootstrapReconciler{}
	test.RegisterController("clusterBootstrap", (bootstrapReconciler).SetupWithManager)
	test.RegisterController("clusterDirectAccess", (&clusterpkg.DirectAccessReconciler{
		RemoteClusterBearerTokenValidity:   10 * time.Minute,
		RenewRemoteClusterBearerTokenAfter: 9 * time.Minute,
	}).SetupWithManager)

	test.RegisterController("clusterStatus", (&clusterpkg.ClusterStatusReconciler{}).SetupWithManager)
	test.RegisterWebhook("clusterValidation", admission.SetupClusterWebhookWithManager)
	test.RegisterWebhook("secretsWebhook", admission.SetupSecretWebhookWithManager)

	test.TestBeforeSuite()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment and remote cluster")
	test.TestAfterSuite()
})
