// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package teamrbac_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	teamrbacpkg "github.com/cloudoperators/greenhouse/internal/controller/teamrbac"

	"github.com/cloudoperators/greenhouse/internal/test"
	webhookv1alpha1 "github.com/cloudoperators/greenhouse/internal/webhook/v1alpha1"
	webhookv1alpha2 "github.com/cloudoperators/greenhouse/internal/webhook/v1alpha2"
	//+kubebuilder:scaffold:imports
)

const (
	testTeamIDPGroup = "test-idp-group"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.
var (
	k8sClient client.Client
	// clusterA
	clusterAKubeConfig []byte
	clusterAKubeClient client.Client
	clusterARemoteEnv  *envtest.Environment
	// clusterB
	clusterBKubeConfig []byte
	clusterBKubeClient client.Client
	clusterBRemoteEnv  *envtest.Environment
)

func TestRBACController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Team RBAC Controller Suite")
}

var _ = BeforeSuite(func() {
	test.RegisterController("roleBindingController", (&teamrbacpkg.TeamRoleBindingReconciler{}).SetupWithManager)
	test.RegisterWebhook("clusterWebhook", webhookv1alpha1.SetupClusterWebhookWithManager)
	test.RegisterWebhook("teamsWebhook", webhookv1alpha1.SetupTeamWebhookWithManager)
	test.RegisterWebhook("teamRoleBindingWebhookV1alpha2", webhookv1alpha2.SetupTeamRoleBindingWebhookWithManager)
	test.RegisterWebhook("teamRoleWebhook", webhookv1alpha1.SetupTeamRoleWebhookWithManager)
	test.TestBeforeSuite()
	k8sClient = test.K8sClient
	bootstrapRemoteClusters()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	test.TestAfterSuite()
	By("tearing down the remote test environment")
	err := clusterARemoteEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func bootstrapRemoteClusters() {
	_, clusterAKubeClient, clusterARemoteEnv, clusterAKubeConfig = test.StartControlPlane("6885", false, false)
	_, clusterBKubeClient, clusterBRemoteEnv, clusterBKubeConfig = test.StartControlPlane("6886", false, false)
}
