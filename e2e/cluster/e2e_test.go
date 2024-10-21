//go:build clusterE2E

package cluster

import (
	"context"
	"github.com/cloudoperators/greenhouse/e2e/cluster/expect"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/e2e"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"testing"
)

const remoteClusterName = "remote-int-cluster"

var (
	env         *e2e.TestEnv
	ctx         context.Context
	adminClient client.Client
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cluster E2E Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx = context.Background()
	env = e2e.NewExecutionEnv(greenhousev1alpha1.AddToScheme).WithOrganization(ctx, "./testdata/organization.yaml")
	adminClient = env.GetAdminClient()
})

var _ = Describe("Cluster E2E", func() {
	Context("Cluster", Ordered, func() {
		It("should onboard remote cluster", func() {
			expect.OnboardRemoteCluster(ctx, adminClient, env.RemoteIntKubeConfigBytes, remoteClusterName, env.TestNamespace)
		})
		It("should have a cluster resource created", func() {
			By("checking the cluster status is ready")
			expect.ClusterResourceIsReady(ctx, adminClient, remoteClusterName, env.TestNamespace)
		})
	})
})
