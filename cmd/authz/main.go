// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"crypto/tls"
	"flag"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/certwatcher"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
	"github.com/cloudoperators/greenhouse/internal/version"
)

var setupLog logr.Logger

func main() {
	var webhookCertPath, webhookCertName, webhookCertKey string
	var metricsAddr, healthzAddr string

	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339TimeEncoder,
	}
	setupLog = zap.New(zap.UseFlagOptions(&opts))

	setupLog.Info("Authorization Webhook", "version", version.GitCommit, "build_date", version.BuildDate, "go", version.GoVersion)

	flag.StringVar(&webhookCertPath, "webhook-cert-path", "ssl", "path to the webhook certificate")
	flag.StringVar(&webhookCertName, "webhook-cert-name", "tls.crt", "name of the webhook certificate")
	flag.StringVar(&webhookCertKey, "webhook-cert-key", "tls.key", "key of the webhook certificate")

	flag.StringVar(&metricsAddr, "metrics-addr", ":6543", "bind address for metrics")
	flag.StringVar(&healthzAddr, "healthz-addr", ":8081", "bind address for health checks")
	flag.Parse()

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(greenhousev1alpha1.AddToScheme(scheme))
	utilruntime.Must(greenhousev1alpha2.AddToScheme(scheme))

	metricsServerOptions := metricsserver.Options{
		BindAddress: metricsAddr,
	}

	// Initial webhook TLS options
	webhookTLSOpts := []func(*tls.Config){}
	var webhookCertWatcher *certwatcher.CertWatcher
	// TODO: env var to turn off the TLS

	if webhookCertPath != "" {
		setupLog.Info("Initializing webhook certificate watcher using provided certificates",
			"webhook-cert-path", webhookCertPath, "webhook-cert-name", webhookCertName, "webhook-cert-key", webhookCertKey)

		var err error
		// certwatcher is a helper for reloading Certificates from disk to be used with tls servers.
		webhookCertWatcher, err = certwatcher.New(
			filepath.Join(webhookCertPath, webhookCertName),
			filepath.Join(webhookCertPath, webhookCertKey),
		)
		if err != nil {
			setupLog.Error(err, "Failed to initialize webhook certificate watcher")
			os.Exit(1)
		}

		webhookTLSOpts = []func(*tls.Config){func(config *tls.Config) {
			config.GetCertificate = webhookCertWatcher.GetCertificate
		}}
	}

	webhookServer := webhook.NewServer(webhook.Options{
		TLSOpts: webhookTLSOpts,
	})

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Logger:                 setupLog,
		Scheme:                 scheme,
		Metrics:                metricsServerOptions,
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: healthzAddr,
		LeaderElection:         false,
	})
	handleError(err, "Failed to create manager")

	// Register the authorizer webhook.
	setupLog.Info("Registering authorization webhook", "path", "/authorize")
	mgr.GetWebhookServer().Register("/authorize", http.HandlerFunc(handleAuthorizeDummy))

	// dynClient, err := dynamic.NewForConfig(mgr.GetConfig())
	// if err != nil {
	// 	handleError(err, "unable to create dynamic client")
	// }
	// mgr.GetWebhookServer().Register("/authorize", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 	handleAuthorize(w, r, dynClient)
	// }))
	setupLog.Info("Health probe addr", "addr", healthzAddr)

	handleError(mgr.AddHealthzCheck("healthz", healthz.Ping), "Failed to set up health check")
	handleError(mgr.AddReadyzCheck("readyz", healthz.Ping), "Failed to set up ready check")

	setupLog.Info("starting manager")
	handleError(mgr.Start(ctrl.SetupSignalHandler()), "Failed to start manager")
}

func handleError(err error, msg string, keysAndValues ...any) {
	if err != nil {
		setupLog.Error(err, msg, keysAndValues...)
		os.Exit(1)
	}
}
