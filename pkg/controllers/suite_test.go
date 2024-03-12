// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
