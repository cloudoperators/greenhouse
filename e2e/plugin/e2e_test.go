// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

//go:build pluginE2E

package cluster

import (
	"context"
	"testing"
	"time"

	"github.com/cloudoperators/greenhouse/pkg/clientutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/cloudoperators/greenhouse/e2e/shared"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

const remoteClusterName = "remote-plugin-cluster"

var (
	env           *shared.TestEnv
	ctx           context.Context
	adminClient   client.Client
	remoteClient  client.Client
	testStartTime time.Time
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin E2E Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	var err error
	ctx = context.Background()
	env = shared.NewExecutionEnv()
	adminClient, err = clientutil.NewK8sClientFromRestClientGetter(env.AdminRestClientGetter)
	Expect(err).ToNot(HaveOccurred(), "there should be no error creating the admin client")
	remoteClient, err = clientutil.NewK8sClientFromRestClientGetter(env.RemoteRestClientGetter)
	Expect(err).ToNot(HaveOccurred(), "there should be no error creating the remote client")
	env = env.WithOrganization(ctx, adminClient, "./testdata/organization.yaml")
	testStartTime = time.Now().UTC()
})

var _ = AfterSuite(func() {
	shared.OffBoardRemoteCluster(ctx, adminClient, remoteClient, testStartTime, remoteClusterName, env.TestNamespace)
	env.GenerateControllerLogs(ctx, testStartTime)
})

var _ = Describe("Plugin E2E", Ordered, func() {
	It("should onboard remote cluster", func() {
		By("onboarding remote cluster")
		shared.OnboardRemoteCluster(ctx, adminClient, env.RemoteKubeConfigBytes, remoteClusterName, env.TestNamespace)
	})
	It("should have a cluster resource created", func() {
		By("verifying if the cluster resource is created")
		Eventually(func(g Gomega) bool {
			err := adminClient.Get(ctx, client.ObjectKey{Name: remoteClusterName, Namespace: env.TestNamespace}, &greenhousev1alpha1.Cluster{})
			g.Expect(err).ToNot(HaveOccurred())
			return true
		}).Should(BeTrue(), "cluster resource should be created")

		By("verifying the cluster status is ready")
		shared.ClusterIsReady(ctx, adminClient, remoteClusterName, env.TestNamespace)
	})
})
