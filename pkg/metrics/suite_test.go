package metrics

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"

	greenhousecluster "github.com/cloudoperators/greenhouse/pkg/controllers/cluster"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

func TestHelmController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MetricsSuite")
}

var _ = BeforeSuite(func() {
	test.RegisterController("cluster", (&greenhousecluster.RemoteClusterReconciler{}).SetupWithManager)
	test.TestBeforeSuite()

	// return the test.Cfg, as the in-cluster config is not available
	ctrl.GetConfig = func() (*rest.Config, error) {
		return test.Cfg, nil
	}
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	test.TestAfterSuite()
})
