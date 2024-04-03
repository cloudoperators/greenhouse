// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package controllers_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/cloudoperators/greenhouse/pkg/test"
)

func TestBaseController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BaseControllerSuite")
}

var (
	remoteCfg                      *rest.Config
	remoteKubeConfig               []byte
	remoteEnvTest                  *envtest.Environment
	remoteClient                   client.Client
	otherRemoteCfg                 *rest.Config
	otherRemoteKubeConfig          []byte
	otherRemoteEnvTest             *envtest.Environment
	otherRemoteClient              client.Client
	testDummyPropagationReconciler = TestDummyPropagationReconciler{}
)

var _ = BeforeSuite(func() {
	test.RegisterController("dummyResourcePropagation", testDummyPropagationReconciler.SetupWithManager)
	test.TestBeforeSuite()

	remoteCfg, remoteClient, remoteEnvTest, remoteKubeConfig = test.StartControlPlane("6885", false, false)
	otherRemoteCfg, otherRemoteClient, otherRemoteEnvTest, otherRemoteKubeConfig = test.StartControlPlane("6885", false, false)
})

var _ = AfterSuite(func() {
	By("tearing down the test environment and remote cluster")
	test.TestAfterSuite()
	err := remoteEnvTest.Stop()
	Expect(err).
		NotTo(HaveOccurred(), "there must be no error stopping the remote environment")
	err = otherRemoteEnvTest.Stop()
	Expect(err).
		NotTo(HaveOccurred(), "there must be no error stopping the remote environment")
})
