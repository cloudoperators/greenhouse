// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	extensionsgreenhousesapv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/extensions.greenhouse/v1alpha1"
	greenhousesapv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
)

func init() {
	RegisterFailHandler(Fail)
}

type (
	registerWebhookFunc    func(mgr ctrl.Manager) error
	registerControllerFunc func(controllerName string, mgr ctrl.Manager) error
)

var (
	allRegisterControllerFuncs = make(map[string]registerControllerFunc, 0)
	allRegisterWebhookFuncs    = make(map[string]registerWebhookFunc, 0)
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
	TestNamespace = "test-org"
)

var (
	Cfg              *rest.Config
	RestClientGetter *clientutil.RestClientGetter
	K8sClient        client.Client
	K8sManager       ctrl.Manager
	KubeConfig       []byte
	testEnv          *envtest.Environment
	Ctx              context.Context
	cancel           context.CancelFunc
	pollInterval     = 1 * time.Second
	updateTimeout    = 30 * time.Second

	// TestBeforeSuite configures the test suite.
	TestBeforeSuite = func() {
		logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

		SetDefaultEventuallyPollingInterval(1 * time.Second)
		SetDefaultEventuallyTimeout(1 * time.Minute)

		installCRDs := clientutil.GetEnvOrDefault("TEST_INSTALL_CRDS", "true") == "true"
		installWebhooks := len(allRegisterWebhookFuncs) > 0 && os.Getenv("TEST_INSTALL_WEBHOOKS") != "false"

		Cfg, K8sClient, testEnv, KubeConfig = StartControlPlane("", installCRDs, installWebhooks)
		_ = K8sClient
		// use the TestNamespace for the ClientGetter to ensure the Helm Actions are executed in the correct namespace
		RestClientGetter = clientutil.NewRestClientGetterFromRestConfig(Cfg, TestNamespace, clientutil.WithPersistentConfig())
		Expect(RestClientGetter).ToNot(BeNil(), "the RestClientGetter should not be nil")

		//+kubebuilder:scaffold:scheme
		var err error
		K8sManager, err = ctrl.NewManager(Cfg, ctrl.Options{
			Scheme: scheme.Scheme,
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
		Ctx, cancel = context.WithCancel(context.TODO())
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

		// Create test namespace
		err = K8sClient.Create(Ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: TestNamespace}})
		Expect(err).NotTo(HaveOccurred(), "there should be no error creating the test namespace")
	}

	// TestAfterSuite configures the test suite.
	TestAfterSuite = func() {
		cancel()
		By("tearing down the test environment")
		Eventually(func() error {
			return testEnv.Stop()
		}).Should(Succeed(), "there should be no error stopping the test environment")
	}
)

// Starts a envTest control plane and returns the config, client, envtest.Environment and raw kubeconfig.
func StartControlPlane(port string, installCRDs, installWebhooks bool) (*rest.Config, client.Client, *envtest.Environment, []byte) {
	// ensure envtest is setup correctly, also see Makefile --> make envtest
	_, isHasEnvKubebuilderAssets := os.LookupEnv("KUBEBUILDER_ASSETS")
	Expect(isHasEnvKubebuilderAssets).
		To(BeTrue(), "the environment variable KUBEBUILDER_ASSETS must be set")

	// Configure control plane
	var testEnv = &envtest.Environment{}
	absPathConfigBasePath, err := clientutil.FindDirUpwards(".", "charts", 10)
	Expect(err).
		NotTo(HaveOccurred(), "there must be no error finding the config directory")
	if installCRDs {
		crdPaths := []string{filepath.Join(absPathConfigBasePath, "manager", "crds")}
		testEnv.CRDDirectoryPaths = crdPaths
		testEnv.ErrorIfCRDPathMissing = true
	}
	if installWebhooks {
		webhookInstallOptions := envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join(absPathConfigBasePath, "manager", "templates", "webhooks.yaml")},
		}
		testEnv.WebhookInstallOptions = webhookInstallOptions
	}
	testEnv.ControlPlane.GetAPIServer().SecureServing.ListenAddr.Port = port
	testEnv.ControlPlane.GetAPIServer().Configure().Append("enable-admission-plugins", "MutatingAdmissionWebhook", "ValidatingAdmissionWebhook")

	// Start control plane
	cfg, err := testEnv.Start()
	Expect(err).
		NotTo(HaveOccurred(), "there must be no error starting the test environment")
	Expect(cfg).
		NotTo(BeNil(), "the configuration of the test environment must not be nil")
	Expect(greenhousesapv1alpha1.AddToScheme(scheme.Scheme)).
		To(Succeed(), "there must be no error adding the greenhouse api to the scheme")
	Expect(extensionsgreenhousesapv1alpha1.AddToScheme(scheme.Scheme)).
		To(Succeed(), "there must be no error adding the extensions.greenhouse api to the scheme")
	Expect(clientgoscheme.AddToScheme(scheme.Scheme)).
		To(Succeed(), "there must no error adding the clientgo api to the scheme")
	Expect(apiextensionsv1.AddToScheme(scheme.Scheme)).
		To(Succeed(), "there must be no error adding the apiextensions api to the scheme")

	// Create k8s client
	k8sClient, err := clientutil.NewK8sClient(cfg)
	Expect(err).
		NotTo(HaveOccurred(), "there must be no error creating a new client")
	Expect(k8sClient).
		NotTo(BeNil(), "the kubernetes client must not be nil")

	// create raw kubeconfig
	user, err := testEnv.ControlPlane.AddUser(envtest.User{
		Name:   "test-admin",
		Groups: []string{"system:masters"},
	}, nil)
	Expect(err).NotTo(HaveOccurred())
	kubeConfig, err := user.KubeConfig()
	Expect(err).NotTo(HaveOccurred())

	// utility to export kubeconfig and use it e.g. on a breakpoint to inspect resources during testing
	if os.Getenv("TEST_EXPORT_KUBECONFIG") == "true" {
		dir := GinkgoT().TempDir()
		kubeCfgFile, err := os.CreateTemp(dir, "*-kubeconfig.yaml")
		Expect(err).NotTo(HaveOccurred())
		_, err = kubeCfgFile.Write(kubeConfig)
		Expect(err).NotTo(HaveOccurred())
		fmt.Printf("export KUBECONFIG=%s\n", kubeCfgFile.Name())
		Expect(err).NotTo(HaveOccurred())
	}

	return cfg, k8sClient, testEnv, kubeConfig
}
