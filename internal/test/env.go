// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	sourcev2 "github.com/fluxcd/source-watcher/api/v2/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	dexapi "github.com/cloudoperators/greenhouse/internal/dex/api"

	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func init() {
	RegisterFailHandler(Fail)
}

type (
	registerWebhookFunc    func(mgr ctrl.Manager) error
	registerControllerFunc func(controllerName string, mgr ctrl.Manager) error
)

var (
	allRegisterControllerFuncs   = make(map[string]registerControllerFunc, 0)
	allRegisterWebhookFuncs      = make(map[string]registerWebhookFunc, 0)
	useExistingGreenhouseCluster = clientutil.GetEnvOrDefault("USE_EXISTING_CLUSTER", "false") == "true"
)

// RegisterController registers a controller for the testbed.
// A currently running testbed is not affected.
func RegisterController(controllerName string, f registerControllerFunc) {
	if _, ok := allRegisterControllerFuncs[controllerName]; !ok {
		allRegisterControllerFuncs[controllerName] = f
	}
}

// UnregisterController removes a controller from the testbed.
// A currently running testbed is not affected.
func UnregisterController(controllerName string) {
	delete(allRegisterControllerFuncs, controllerName)
}

// RegisterWebhook registers a webhook for the testbed.
// A currently running testbed is not affected.
func RegisterWebhook(webhookName string, f registerWebhookFunc) {
	if _, ok := allRegisterWebhookFuncs[webhookName]; !ok {
		allRegisterWebhookFuncs[webhookName] = f
	}
}

// UnregisterWebhook removes a webhook from the testbed.
// A currently running testbed is not affected.
func UnregisterWebhook(webhookName string) {
	delete(allRegisterWebhookFuncs, webhookName)
}

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

const (
	// TestNamespace is the namespace used for testing. Name reflects it represents a greenhouse org.
	TestNamespace           = "test-org"
	TestGreenhouseNamespace = "greenhouse"
)

var (
	// Cfg is the rest.Config to access the cluster the tests are running against.
	Cfg *rest.Config
	// RestClientGetter is the clientutil.RestClientGetter to access the cluster the tests are running against.
	RestClientGetter *clientutil.RestClientGetter
	// K8sClient is the client.Client to access the cluster the tests are running against.
	K8sClient client.Client
	// K8sManager is the ctrl.Manager the controllers are run by.
	K8sManager ctrl.Manager
	// KubeConfig is the raw kubeconfig to access the cluster the tests are running against.
	KubeConfig []byte
	// Ctx is the context to use for the tests.
	Ctx context.Context
	// IsUseExistingCluster is true if the tests are running against an existing cluster.
	IsUseExistingCluster = useExistingGreenhouseCluster
	testEnv              *envtest.Environment
	cancel               context.CancelFunc
	pollInterval         = 1 * time.Second
	updateTimeout        = 30 * time.Second

	persistedKubeconfig = os.Getenv("KUBECONFIG")

	// TestBeforeSuite configures the test suite.
	TestBeforeSuite = func() {
		logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

		SetDefaultEventuallyPollingInterval(1 * time.Second)
		SetDefaultEventuallyTimeout(1 * time.Minute)

		installCRDs := clientutil.GetEnvOrDefault("TEST_INSTALL_CRDS", "true") == "true"
		installWebhooks := len(allRegisterWebhookFuncs) > 0 && os.Getenv("TEST_INSTALL_WEBHOOKS") != "false"
		if useExistingGreenhouseCluster {
			// we are making use of https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest#pkg-constants to prevent starting a new control plane
			e2eKubeconfig := os.Getenv("TEST_KUBECONFIG")
			Expect(e2eKubeconfig).NotTo(BeEmpty(), "the environment variable TEST_KUBECONFIG must be set to run the tests against a remote cluster")
			// we overwrite the KUBECONFIG env var expected by envtest to make sure tests are not accidentally running against existing k8s context
			os.Setenv("KUBECONFIG", e2eKubeconfig)
			fmt.Printf("Running tests against existing cluster with kubeconfig: %s\n", e2eKubeconfig)
			installCRDs = false
			installWebhooks = false
		} else {
			// ensure envtest is setup correctly, also see Makefile --> make envtest
			_, isHasEnvKubebuilderAssets := os.LookupEnv("KUBEBUILDER_ASSETS")
			Expect(isHasEnvKubebuilderAssets).
				To(BeTrue(), "the environment variable KUBEBUILDER_ASSETS must be set to run the tests against local envtest")
		}

		Cfg, K8sClient, testEnv, KubeConfig = StartControlPlane("", installCRDs, installWebhooks)
		_ = K8sClient
		// use the TestNamespace for the ClientGetter to ensure the Helm Actions are executed in the correct namespace
		RestClientGetter = clientutil.NewRestClientGetterFromRestConfig(Cfg, TestNamespace, clientutil.WithPersistentConfig())
		Expect(RestClientGetter).ToNot(BeNil(), "the RestClientGetter should not be nil")

		Ctx, cancel = context.WithCancel(context.TODO())
		// Only start the local manager and webhook server if we are not using an existing cluster
		if !useExistingGreenhouseCluster {
			//+kubebuilder:scaffold:scheme
			var err error
			K8sManager, err = ctrl.NewManager(Cfg, ctrl.Options{
				Scheme: testEnv.Scheme,
				Metrics: metricsserver.Options{
					BindAddress: "0",
				},
				WebhookServer: webhook.NewServer(webhook.Options{
					Host:    testEnv.WebhookInstallOptions.LocalServingHost,
					Port:    testEnv.WebhookInstallOptions.LocalServingPort,
					CertDir: testEnv.WebhookInstallOptions.LocalServingCertDir,
				}),
				LeaderElection: false,
			})
			Expect(err).
				ToNot(HaveOccurred(), "there must be no error creating a manager")
			Expect(K8sManager).
				NotTo(BeNil(), "the manager must not be nil")

			// Register webhooks.
			for webhookName, registerFunc := range allRegisterWebhookFuncs {
				logf.FromContext(Ctx, "message", "registering webhook", "name", webhookName)
				Expect(registerFunc(K8sManager)).To(Succeed(), "there must be no error registering the webhook", "name", webhookName)
			}

			// Register controllers.
			for controllerName, registerFunc := range allRegisterControllerFuncs {
				Expect(registerFunc(controllerName, K8sManager)).
					To(Succeed(), "there must be no error registering the controller", "name", controllerName)
			}

			// Start Manager
			go func() {
				defer GinkgoRecover()
				err = K8sManager.Start(Ctx)
				Expect(err).
					ToNot(HaveOccurred(), "there must be no error starting the manager")
			}()

			if len(allRegisterWebhookFuncs) > 0 {
				// wait for the webhook server to get ready
				dialer := &net.Dialer{Timeout: time.Second}
				addrPort := fmt.Sprintf("%s:%d", testEnv.WebhookInstallOptions.LocalServingHost, testEnv.WebhookInstallOptions.LocalServingPort)
				Eventually(func() error {
					conn, err := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true}) //nolint:gosec
					if err != nil {
						return err
					}
					conn.Close()
					return nil
				}, updateTimeout, pollInterval).Should(Succeed(), "there should be no error dialing the webhook server")
			}
		}
		// Create test namespace and thereby test connection to the cluster
		err := K8sClient.Create(Ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: TestNamespace}})
		Expect(err).NotTo(HaveOccurred(), "there should be no error creating the test namespace")
		err = K8sClient.Create(Ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: TestGreenhouseNamespace}})
		Expect(err).NotTo(HaveOccurred(), "there should be no error creating the greenhouse namespace")
	}

	// TestAfterSuite configures the test suite.
	TestAfterSuite = func() {
		// By deleting the test-namespace, this is especially for USE_EXISTING_GREENHOUSE_CLUSTER=true
		err := K8sClient.Delete(Ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: TestNamespace}})
		Expect(err).NotTo(HaveOccurred(), "there should be no error deleting the test namespace")
		cancel()
		By("tearing down the test environment")
		Eventually(func() error {
			return testEnv.Stop()
		}).Should(Succeed(), "there should be no error stopping the test environment")

		if useExistingGreenhouseCluster {
			// we reset the KUBECONFIG env var to its original value
			os.Setenv("KUBECONFIG", persistedKubeconfig)
		}
	}
)

// Starts a envTest control plane and returns the config, client, envtest.Environment and raw kubeconfig.
func StartControlPlane(port string, installCRDs, installWebhooks bool) (*rest.Config, client.Client, *envtest.Environment, []byte) {
	// Configure control plane
	var testEnv = &envtest.Environment{}
	absPathConfigBasePath, err := clientutil.FindDirUpwards(".", "charts", 10)
	Expect(err).
		NotTo(HaveOccurred(), "there must be no error finding the config directory")
	binPathBasePath, err := clientutil.FindDirUpwards(".", "bin", 10)
	Expect(err).
		NotTo(HaveOccurred(), "there must be no error finding the flux crd directory")
	if installCRDs {
		crdPaths := []string{
			filepath.Join(absPathConfigBasePath, "manager", "crds"),
			filepath.Join(absPathConfigBasePath, "idproxy", "crds"),
			filepath.Join(binPathBasePath, "flux"),
		}
		testEnv.CRDDirectoryPaths = crdPaths
		testEnv.ErrorIfCRDPathMissing = true
	}
	if installWebhooks {
		webhookInstallOptions := envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join(absPathConfigBasePath, "manager", "templates", "webhook", "webhooks.yaml")},
		}
		testEnv.WebhookInstallOptions = webhookInstallOptions
	}
	testEnv.ControlPlane.GetAPIServer().Port = port
	testEnv.ControlPlane.GetAPIServer().Configure().Append("enable-admission-plugins", "MutatingAdmissionWebhook", "ValidatingAdmissionWebhook")

	testEnvScheme := runtime.NewScheme()
	Expect(greenhousev1alpha1.AddToScheme(testEnvScheme)).
		To(Succeed(), "there must be no error adding the greenhouse api v1alpha1 to the scheme")
	Expect(greenhousev1alpha2.AddToScheme(testEnvScheme)).
		To(Succeed(), "there must be no error adding the greenhouse api v1alpha2 to the scheme")
	Expect(clientgoscheme.AddToScheme(testEnvScheme)).
		To(Succeed(), "there must no error adding the clientgo api to the scheme")
	Expect(apiextensionsv1.AddToScheme(testEnvScheme)).
		To(Succeed(), "there must be no error adding the apiextensions api to the scheme")
	Expect(dexapi.AddToScheme(testEnvScheme)).
		To(Succeed(), "there must be no error adding the dex api to the scheme")
	Expect(sourcev1.AddToScheme(testEnvScheme)).To(Succeed(), "there must be no error adding the flux source api to the scheme")
	Expect(helmv2.AddToScheme(testEnvScheme)).To(Succeed(), "there must be no error adding the flux helm api to the scheme")
	Expect(kustomizev1.AddToScheme(testEnvScheme)).To(Succeed(), "there must be no error adding the flux kustomize api to the scheme")
	Expect(sourcev2.AddToScheme(testEnvScheme)).To(Succeed(), "there must be no error adding the flux source watcher api to the scheme")

	// Make sure all schemes are added before starting the envtest. This will enable conversion webhooks.
	testEnv.CRDInstallOptions = envtest.CRDInstallOptions{
		Scheme: testEnvScheme,
	}
	testEnv.Scheme = testEnvScheme

	// Start control plane
	cfg, err := testEnv.Start()
	Expect(err).
		NotTo(HaveOccurred(), "there must be no error starting the test environment")
	Expect(cfg).
		NotTo(BeNil(), "the configuration of the test environment must not be nil")

	// Create k8s client
	k8sClient, err := clientutil.NewK8sClient(cfg)
	Expect(err).
		NotTo(HaveOccurred(), "there must be no error creating a new client")
	Expect(k8sClient).
		NotTo(BeNil(), "the kubernetes client must not be nil")

	// create raw kubeconfig
	var kubeConfig []byte
	// we extract the kubeconfig from env var if we are using an existing cluster
	if useExistingGreenhouseCluster {
		kubeConfig, err = KubeconfigFromEnvVar("KUBECONFIG")
		Expect(err).NotTo(HaveOccurred())
	} else {
		// we add a user to the control plane to easily get a kubeconfig
		user, err := testEnv.ControlPlane.AddUser(envtest.User{
			Name:   "test-admin",
			Groups: []string{"system:masters"},
		}, nil)
		Expect(err).NotTo(HaveOccurred())
		kubeConfig, err = user.KubeConfig()
		Expect(err).NotTo(HaveOccurred())
	}

	// utility to export kubeconfig and use it e.g. on a breakpoint to inspect resources during testing

	dir := GinkgoT().TempDir()
	kubeCfgFile, err := os.CreateTemp(dir, "*-kubeconfig.yaml")
	Expect(err).NotTo(HaveOccurred())
	_, err = kubeCfgFile.Write(kubeConfig)
	Expect(err).NotTo(HaveOccurred())
	err = os.Setenv("KUBECONFIG", kubeCfgFile.Name())
	Expect(err).NotTo(HaveOccurred())
	fmt.Printf("export KUBECONFIG=%s\n", kubeCfgFile.Name())

	return cfg, k8sClient, testEnv, kubeConfig
}
