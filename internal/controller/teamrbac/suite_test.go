// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package teamrbac_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
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
	test.RegisterWebhook("teamRoleBindingWebhookV1alpha1", setupTeamRoleBindingV1alpha1WebhookForTest)
	test.RegisterWebhook("teamRoleBindingWebhookV1alpha2", setupTeamRoleBindingV1alpha2WebhookForTest)
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

// setupTeamRoleBindingV1alpha1WebhookForTest adds an indexField for '.spec.teamRoleRef', additionally to setting up the webhook for the v1alpha1 TeamRoleBinding resource. It is used in the integration tests.
// We can't add this to the webhook setup because it's already indexed in the main.go and indexing the field twice is not possible.
func setupTeamRoleBindingV1alpha1WebhookForTest(mgr manager.Manager) error {
	err := mgr.GetFieldIndexer().IndexField(context.Background(), &greenhousev1alpha1.TeamRoleBinding{}, greenhouseapis.RolebindingTeamRoleRefField, func(rawObj client.Object) []string {
		// Extract the Role name from the RoleBinding Spec, if one is provided
		roleBinding, ok := rawObj.(*greenhousev1alpha1.TeamRoleBinding)
		if roleBinding.Spec.TeamRoleRef == "" || !ok {
			return nil
		}
		return []string{roleBinding.Spec.TeamRoleRef}
	})
	Expect(err).ToNot(HaveOccurred(), "there should be no error indexing the v1alpha1 TeamRoleBindings by teamRoleRef")
	return webhookv1alpha1.SetupTeamRoleBindingWebhookWithManager(mgr)
}

// setupTeamRoleBindingV1alpha2WebhookForTest adds an indexField for '.spec.teamRoleRef', additionally to setting up the webhook for the v1alpha2 TeamRoleBinding resource. It is used in the integration tests.
// We can't add this to the webhook setup because it's already indexed in the main.go and indexing the field twice is not possible.
func setupTeamRoleBindingV1alpha2WebhookForTest(mgr manager.Manager) error {
	err := mgr.GetFieldIndexer().IndexField(context.Background(), &greenhousev1alpha2.TeamRoleBinding{}, greenhouseapis.RolebindingTeamRoleRefField, func(rawObj client.Object) []string {
		// Extract the Role name from the TeamRoleBinding Spec, if one is provided
		teamRoleBinding, ok := rawObj.(*greenhousev1alpha2.TeamRoleBinding)
		if teamRoleBinding.Spec.TeamRoleRef == "" || !ok {
			return nil
		}
		return []string{teamRoleBinding.Spec.TeamRoleRef}
	})
	Expect(err).ToNot(HaveOccurred(), "there should be no error indexing the v1alpha2 TeamRoleBindings by teamRoleRef")
	return webhookv1alpha2.SetupTeamRoleBindingWebhookWithManager(mgr)
}
