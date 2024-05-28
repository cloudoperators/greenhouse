// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"os"
	"testing"
	"time"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2ESuite")
}

var centralClusterClient client.Client
var centralClusterKubeconfigData []byte
var centralClusterNamespace = "e2e-org"
var centralClusterNamespaceEnvVar = "TEST_E2E_NAMESPACE"

const (
	timeout  = time.Second * 60
	interval = time.Second
)

var _ = BeforeSuite(func() {
	By("Check the Kubernetes client by connecting to the cluster")

	// Create Kubernetes client
	restConfig, err := config.GetConfig()
	Expect(err).NotTo(HaveOccurred())
	err = greenhousev1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	centralClusterClient, err = client.New(restConfig, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())

	// Create & fill kubeconfig data
	if os.Getenv("TEST_E2E_KUBECONFIG_INTERNAL_DOCKER_NETWORK") != "" {
		centralClusterKubeconfigData, err = os.ReadFile(os.Getenv("TEST_E2E_KUBECONFIG_INTERNAL_DOCKER_NETWORK"))
		Expect(err).NotTo(HaveOccurred())
	} else {
		centralClusterKubeconfigData, err = createKubeconfigFileForRestConfig(*restConfig)
		Expect(err).NotTo(HaveOccurred())
		Expect(centralClusterKubeconfigData).Should(Not(BeEmpty()))
	}

	By("Cluster version compatibility")
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	Expect(err).NotTo(HaveOccurred())
	version, err := discoveryClient.ServerVersion()
	Expect(err).NotTo(HaveOccurred())
	Expect(version.Major).Should(Equal("1"))
	Expect(version.Minor).Should(Equal("29")) // TODO(onur): list and check for supported versions

	if os.Getenv(centralClusterNamespaceEnvVar) != "" {
		centralClusterNamespace = os.Getenv(centralClusterNamespaceEnvVar)
	}

})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
})

func createKubeconfigFileForRestConfig(restConfig rest.Config) ([]byte, error) {

	clusters := make(map[string]*clientcmdapi.Cluster)
	clusters["default-cluster"] = &clientcmdapi.Cluster{
		Server:                   restConfig.Host,
		CertificateAuthorityData: restConfig.CAData,
	}
	contexts := make(map[string]*clientcmdapi.Context)
	contexts["default-context"] = &clientcmdapi.Context{
		Cluster:  "default-cluster",
		AuthInfo: "default-user",
	}
	authinfos := make(map[string]*clientcmdapi.AuthInfo)
	authinfos["default-user"] = &clientcmdapi.AuthInfo{
		ClientCertificateData: restConfig.CertData,
		ClientKeyData:         restConfig.KeyData,
	}
	clientConfig := clientcmdapi.Config{
		Kind:           "Config",
		APIVersion:     "v1",
		Clusters:       clusters,
		Contexts:       contexts,
		CurrentContext: "default-context",
		AuthInfos:      authinfos,
	}
	return clientcmd.Write(clientConfig)
}
