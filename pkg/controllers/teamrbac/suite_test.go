// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package teamrbac

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/cloudoperators/greenhouse/pkg/admission"
	"github.com/cloudoperators/greenhouse/pkg/test"
	//+kubebuilder:scaffold:imports
)

const (
	testTeamIDPGroup = "test-idp-group"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.
var (
	k8sClient        client.Client
	remoteKubeConfig []byte
	remoteK8sClient  client.Client
	remoteTestEnv    *envtest.Environment
)

func TestRBACController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Team RBAC Controller Suite")
}

var _ = BeforeSuite(func() {
	test.RegisterController("roleBindingController", (&TeamRoleBindingReconciler{}).SetupWithManager)
	test.RegisterWebhook("clusterWebhook", admission.SetupClusterWebhookWithManager)
	test.RegisterWebhook("teamsWebhook", admission.SetupTeamWebhookWithManager)
	test.RegisterWebhook("teamRoleBindingWebhook", admission.SetupTeamRoleBindingWebhookWithManager)
	test.RegisterWebhook("teamRoleWebhook", admission.SetupTeamRoleWebhookWithManager)
	test.TestBeforeSuite()
	k8sClient = test.K8sClient
	bootstrapRemoteCluster()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	test.TestAfterSuite()
	By("tearing down the remote test environment")
	err := remoteTestEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func bootstrapRemoteCluster() {
	_, remoteK8sClient, remoteTestEnv, remoteKubeConfig = test.StartControlPlane("6885", false, false)
}
