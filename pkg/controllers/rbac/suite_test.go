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

package rbac

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/cloudoperators/greenhouse/pkg/admission"
	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/test"
	//+kubebuilder:scaffold:imports
)

const (
	testTeamName     = "test-team"
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

var testCluster = &greenhousev1alpha1.Cluster{
	TypeMeta: metav1.TypeMeta{
		Kind:       "Cluster",
		APIVersion: greenhousev1alpha1.GroupVersion.String(),
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-cluster",
		Namespace: test.TestNamespace,
	},
	Spec: greenhousev1alpha1.ClusterSpec{
		AccessMode: greenhousev1alpha1.ClusterAccessModeDirect,
	},
}

var testClusterK8sSecret = &corev1.Secret{
	TypeMeta: metav1.TypeMeta{
		Kind:       "Secret",
		APIVersion: corev1.GroupName,
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-cluster",
		Namespace: test.TestNamespace,
	},
	Type: greenhouseapis.SecretTypeKubeConfig,
}

var testTeam = &greenhousev1alpha1.Team{
	TypeMeta: metav1.TypeMeta{
		Kind:       "Team",
		APIVersion: greenhousev1alpha1.GroupVersion.String(),
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      testTeamName,
		Namespace: test.TestNamespace,
	},
	Spec: greenhousev1alpha1.TeamSpec{
		Description:    "Test Team",
		MappedIDPGroup: testTeamIDPGroup,
	},
}

func TestRBACController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RBAC Controller Suite")
}

var _ = BeforeSuite(func() {
	test.RegisterController("roleBindingController", (&RoleBindingReconciler{}).SetupWithManager)
	test.RegisterWebhook("clusterWebhook", admission.SetupClusterWebhookWithManager)
	test.RegisterWebhook("teamsWebhook", admission.SetupTeamWebhookWithManager)
	test.RegisterWebhook("roleBindingWebhook", admission.SetupRoleBindingWebhookWithManager)
	test.RegisterWebhook("roleWebhook", admission.SetupRoleWebhookWithManager)
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
