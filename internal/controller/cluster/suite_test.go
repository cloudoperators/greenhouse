// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	clusterpkg "github.com/cloudoperators/greenhouse/internal/controller/cluster"
	"github.com/cloudoperators/greenhouse/internal/test"
	webhookv1alpha1 "github.com/cloudoperators/greenhouse/internal/webhook/v1alpha1"
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
	test.RegisterController("clusterDirectAccess", (&clusterpkg.RemoteClusterReconciler{
		RemoteClusterBearerTokenValidity:   10 * time.Minute,
		RenewRemoteClusterBearerTokenAfter: 9 * time.Minute,
	}).SetupWithManager)

	test.RegisterWebhook("teamsWebhook", webhookv1alpha1.SetupTeamWebhookWithManager)
	test.RegisterWebhook("clusterValidation", webhookv1alpha1.SetupClusterWebhookWithManager)
	test.RegisterWebhook("secretsWebhook", webhookv1alpha1.SetupSecretWebhookWithManager)

	// orgWebhook is required by cluster-kubeconfig since it uses organization-level resources
	test.RegisterController("kubeconfig", (&clusterpkg.KubeconfigReconciler{}).SetupWithManager)
	test.RegisterWebhook("orgWebhook", webhookv1alpha1.SetupOrganizationWebhookWithManager)

	test.TestBeforeSuite()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment and remote cluster")
	test.TestAfterSuite()
})
