// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudoperators/greenhouse/internal/admission"
	clusterpkg "github.com/cloudoperators/greenhouse/internal/controllers/cluster"
	"github.com/cloudoperators/greenhouse/internal/test"
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

	test.RegisterWebhook("clusterValidation", admission.SetupClusterWebhookWithManager)
	test.RegisterWebhook("secretsWebhook", admission.SetupSecretWebhookWithManager)

	// orgWebhook is required by cluster-kubeconfig since it uses organization-level resources
	test.RegisterController("kubeconfig", (&clusterpkg.KubeconfigReconciler{}).SetupWithManager)
	test.RegisterWebhook("orgWebhook", admission.SetupOrganizationWebhookWithManager)

	test.TestBeforeSuite()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment and remote cluster")
	test.TestAfterSuite()
})
