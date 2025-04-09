// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"

	greenhousecluster "github.com/cloudoperators/greenhouse/internal/controller/cluster"
	"github.com/cloudoperators/greenhouse/internal/test"
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
