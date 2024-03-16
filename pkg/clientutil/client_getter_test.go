// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package clientutil_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

var (
	wrongSecretType corev1.SecretType = "someSecret/type"
)

var _ = Describe("Testing the RestClientGetter", func() {
	It("should create a RestClientGetter and retrieve cluster version", func() {
		restClientGetter := clientutil.NewRestClientGetterFromRestConfig(test.Cfg, corev1.NamespaceDefault, clientutil.WithRuntimeOptions(clientutil.RuntimeOptions{42, 1337}))
		Expect(restClientGetter).ToNot(BeNil(), "the RestClientGetter should not be nil")

		var actServerVersion, expServerVersion *version.Info

		By("checking the rate limitting defaults are overwritten", func() {
			cut, err := restClientGetter.ToRESTConfig()
			Expect(err).ToNot(HaveOccurred(), "there should be no error creating a rest config")
			Expect(cut.QPS).To(Equal(float32(42)), "the QPS should be 42")
			Expect(cut.Burst).To(Equal(1337), "the Burst should be 1337")
		})

		By("getting the cluster version via MemoryRESTClient", func() {
			dc, err := restClientGetter.ToDiscoveryClient()
			Expect(err).ToNot(HaveOccurred(), "there should be no error creating a discovery client")
			actServerVersion, err = dc.ServerVersion()
			Expect(err).ToNot(HaveOccurred(), "there should be no error getting the server version from the discovery client")
		})

		By("getting the cluster version via Manager Config", func() {
			dcManager, err := discovery.NewDiscoveryClientForConfig(test.K8sManager.GetConfig())
			Expect(err).ToNot(HaveOccurred(), "there should be no error creating a discovery client from manager config")
			expServerVersion, err = dcManager.ServerVersion()
			Expect(err).ToNot(HaveOccurred(), "there should be no error getting the server version from the discovery client created from manager config")
		})

		Expect(actServerVersion).To(Equal(expServerVersion), "the discovered version should be the same")
	})

	It("should create a RestClientGetter and retrieve RESTMapper", func() {
		cut := clientutil.NewRestClientGetterFromRestConfig(test.Cfg, corev1.NamespaceDefault)
		Expect(cut).ToNot(BeNil(), "the RestClientGetter should not be nil")

		mapper, err := cut.ToRESTMapper()
		Expect(err).ToNot(HaveOccurred(), "there should be no error creating a rest mapper")

		kinds, err := mapper.KindsFor(greenhousev1alpha1.GroupVersion.WithResource("plugins"))
		Expect(err).ToNot(HaveOccurred(), "there should be no error getting the kinds for the resource")
		Expect(kinds).ToNot(BeEmpty(), "the kinds should not be empty")
		Expect(kinds).To(ContainElement(schema.GroupVersionKind{Group: greenhousev1alpha1.GroupVersion.Group, Version: greenhousev1alpha1.GroupVersion.Version, Kind: "Plugin"}), "the kinds should contain the Plugin kind")
	})

	It("should create a RestClientGetter from InCluster config and retrieve cluster version", func() {
		restClientGetter, err := clientutil.NewRestClientGetterForInCluster(corev1.NamespaceDefault, clientutil.WithRuntimeOptions(clientutil.RuntimeOptions{42, 1337}))
		Expect(err).ToNot(HaveOccurred(), "there should be no error creating the in-cluster RestClientGetter")
		Expect(restClientGetter).ToNot(BeNil(), "the RestClientGetter should not be nil")

		var actServerVersion, expServerVersion *version.Info

		By("checking the rate limitting defaults are overwritten", func() {
			cut, err := restClientGetter.ToRESTConfig()
			Expect(err).ToNot(HaveOccurred(), "there should be no error creating a rest config")
			Expect(cut.QPS).To(Equal(float32(42)), "the QPS should be 42")
			Expect(cut.Burst).To(Equal(1337), "the Burst should be 1337")
		})

		By("getting the cluster version via MemoryRESTClient", func() {
			dc, err := restClientGetter.ToDiscoveryClient()
			Expect(err).ToNot(HaveOccurred(), "there should be no error creating a discovery client")
			actServerVersion, err = dc.ServerVersion()
			Expect(err).ToNot(HaveOccurred(), "there should be no error getting the server version from the discovery client")
		})

		By("getting the cluster version via Manager Config", func() {
			dcManager, err := discovery.NewDiscoveryClientForConfig(test.K8sManager.GetConfig())
			Expect(err).ToNot(HaveOccurred(), "there should be no error creating a discovery client from manager config")
			expServerVersion, err = dcManager.ServerVersion()
			Expect(err).ToNot(HaveOccurred(), "there should be no error getting the server version from the discovery client created from manager config")
		})

		Expect(actServerVersion).To(Equal(expServerVersion), "the discovered version should be the same")
	})

	When("retrieving a RestClientGetter from a secret", func() {

		DescribeTable("it should get a RestClientGetter from a valid kubeconfig secret",
			expectSuccess,
			Entry("at data.kubeconfig", greenhouseapis.SecretTypeKubeConfig, greenhouseapis.KubeConfigKey),
		)

		DescribeTable("it should fail getting a RestClientGetter from an invalid kubeconfig secret",
			expectFailure,
			Entry("with secret of wrong type", wrongSecretType, greenhouseapis.KubeConfigKey, "is not of type"),
		)

	})
})

var expectSuccess = func(secretType corev1.SecretType, dataKey string) {
	validKubeConfigSecret := returnTestKubeConfigSecret(secretType, dataKey, test.KubeConfig)
	Expect(test.K8sClient.Create(test.Ctx, &validKubeConfigSecret, &client.CreateOptions{})).
		Should(Succeed(), "there should be no error creating the kubeconfig secret")

	restClientGetter, err := clientutil.NewRestClientGetterFromSecret(&validKubeConfigSecret, corev1.NamespaceDefault)
	Expect(err).ToNot(HaveOccurred(), "there should be no error getting the RestClientGetter")
	Expect(restClientGetter).ToNot(BeNil(), "the RestClientGetter should not be nil")

	var actServerVersion, expServerVersion *version.Info

	By("getting the cluster version via MemoryRESTClient", func() {
		dc, err := restClientGetter.ToDiscoveryClient()
		Expect(err).ToNot(HaveOccurred(), "there should be no error creating a discovery client")
		actServerVersion, err = dc.ServerVersion()
		Expect(err).ToNot(HaveOccurred(), "there should be no error getting the server version from the discovery client")
	})

	By("getting the cluster version via Manager Config", func() {
		dcManager, err := discovery.NewDiscoveryClientForConfig(test.K8sManager.GetConfig())
		Expect(err).ToNot(HaveOccurred(), "there should be no error creating a discovery client from manager config")
		expServerVersion, err = dcManager.ServerVersion()
		Expect(err).ToNot(HaveOccurred(), "there should be no error getting the server version from the discovery client created from manager config")
	})

	Expect(actServerVersion).To(Equal(expServerVersion), "the discovered version should be the same")

	Expect(test.K8sClient.Delete(test.Ctx, &validKubeConfigSecret, &client.DeleteOptions{})).
		Should(Succeed(), "there should be no error deleting the kubeconfig secret")
}

var expectFailure = func(secretType corev1.SecretType, dataKey string, expectedErrorString string) {

	invalidKubeConfigSecret := returnTestKubeConfigSecret(secretType, dataKey, test.KubeConfig)
	Expect(test.K8sClient.Create(test.Ctx, &invalidKubeConfigSecret, &client.CreateOptions{})).
		Should(Succeed(), "there should be no error creating the kubeconfig secret")

	restClientGetter, err := clientutil.NewRestClientGetterFromSecret(&invalidKubeConfigSecret, corev1.NamespaceDefault)
	Expect(err).To(HaveOccurred(), "there should be an error getting the RestClientGetter")
	Expect(restClientGetter).To(BeNil(), "the RestClientGetter should be nil")
	Expect(err.Error()).To(ContainSubstring(expectedErrorString), "the error should contain the expected error string")

	Expect(test.K8sClient.Delete(test.Ctx, &invalidKubeConfigSecret, &client.DeleteOptions{})).
		Should(Succeed(), "there should be no error deleting the kubeconfig secret")

}
