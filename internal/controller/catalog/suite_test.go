package catalog

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudoperators/greenhouse/internal/test"
)

func TestPluginDefinitionCatalog(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PluginDefinitionCatalogControllerSuite")
}

var _ = BeforeSuite(func() {
	test.RegisterController("pluginDefinitionCatalog", (&CatalogReconciler{}).SetupWithManager)
	test.TestBeforeSuite()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment and remote cluster")
	test.TestAfterSuite()
})
