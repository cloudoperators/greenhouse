package v1alpha1_test

import (
	"github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/test"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	setup    *test.TestSetup
	clusterA *v1alpha1.Cluster
	clusterB *v1alpha1.Cluster
	clusterC *v1alpha1.Cluster
	clusterD *v1alpha1.Cluster
	clusterE *v1alpha1.Cluster
	clusterF *v1alpha1.Cluster
	clusterG *v1alpha1.Cluster
)

var _ = Describe("ClusterSelector type's ListClusters method", Ordered, func() {
	BeforeAll(func() {
		setup = test.NewTestSetup(test.Ctx, test.K8sClient, "test-org")

		By("creating test clusters")
		clusterA = setup.CreateCluster(test.Ctx, "cluster-a")
		clusterB = setup.CreateCluster(test.Ctx, "cluster-b", test.WithLabel("group", "first"))
		clusterC = setup.CreateCluster(test.Ctx, "cluster-c", test.WithLabel("group", "second"))
		clusterD = setup.CreateCluster(test.Ctx, "cluster-d", test.WithLabel("group", "first"))
		clusterE = setup.CreateCluster(test.Ctx, "cluster-e", test.WithLabel("group", "second"))
		clusterF = setup.CreateCluster(test.Ctx, "cluster-f", test.WithLabel("group", "second"))
		clusterG = setup.CreateCluster(test.Ctx, "cluster-g", test.WithLabel("group", "second"))
	})

	AfterAll(func() {
		By("cleaning up test clusters")
		test.EventuallyDeleted(test.Ctx, setup.Client, clusterA)
		test.EventuallyDeleted(test.Ctx, setup.Client, clusterB)
		test.EventuallyDeleted(test.Ctx, setup.Client, clusterC)
		test.EventuallyDeleted(test.Ctx, setup.Client, clusterD)
		test.EventuallyDeleted(test.Ctx, setup.Client, clusterE)
		test.EventuallyDeleted(test.Ctx, setup.Client, clusterF)
		test.EventuallyDeleted(test.Ctx, setup.Client, clusterG)
	})

	It("should return correct cluster by Name", func() {
		By("setting up a ClusterSelector")
		cs := new(v1alpha1.ClusterSelector)
		cs.Name = "cluster-a"

		By("executing ListClusters method")
		clusters, err := cs.ListClusters(test.Ctx, setup.Client, setup.Namespace())
		Expect(err).ToNot(HaveOccurred(), "there should be no error listing the clusters")

		By("checking returned clusters")
		Expect(clusters.Items).To(HaveLen(1), "ListClusters should match exactly one cluster")
		Expect(clusters.Items[0].Name).To(Equal("cluster-a"), "ListClusters should return cluster with name cluster-a")
	})

	It("should list all clusters matching LabelSelector", func() {
		By("setting up a ClusterSelector")
		cs := new(v1alpha1.ClusterSelector)
		cs.LabelSelector = v1.LabelSelector{
			MatchLabels: map[string]string{
				"group": "first",
			},
		}

		By("executing ListClusters method")
		clusters, err := cs.ListClusters(test.Ctx, setup.Client, setup.Namespace())
		Expect(err).ToNot(HaveOccurred(), "there should be no error listing the clusters")

		By("checking returned clusters")
		Expect(clusters.Items).To(HaveLen(2), "ListClusters should match exactly two clusters")

		clusterNames := make([]string, 0, 2)
		for _, v := range clusters.Items {
			clusterNames = append(clusterNames, v.Name)
		}
		Expect(clusterNames).To(ConsistOf("cluster-b", "cluster-d"), "ListClusters should return clusters with names cluster-b and cluster-d")
	})

	It("should list all clusters matching LabelSelector except for those in ExcludeList", func() {
		By("setting up a ClusterSelector")
		cs := new(v1alpha1.ClusterSelector)
		cs.LabelSelector = v1.LabelSelector{
			MatchLabels: map[string]string{
				"group": "second",
			},
		}
		cs.ExcludeList = []string{"cluster-c", "cluster-f"}

		By("executing ListClusters method")
		clusters, err := cs.ListClusters(test.Ctx, setup.Client, setup.Namespace())
		Expect(err).ToNot(HaveOccurred(), "there should be no error listing the clusters")

		By("checking returned clusters")
		Expect(clusters.Items).To(HaveLen(2), "ListClusters should match exactly two clusters")

		clusterNames := make([]string, 0, 2)
		for _, v := range clusters.Items {
			clusterNames = append(clusterNames, v.Name)
		}
		Expect(clusterNames).To(ConsistOf("cluster-e", "cluster-g"), "ListClusters should return clusters with names cluster-e and cluster-g")
	})
})
