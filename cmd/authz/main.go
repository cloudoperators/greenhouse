// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
	"github.com/cloudoperators/greenhouse/internal/version"
)

const (
	authzTLS = "AUTHZ_TLS"
)

var (
	scheme          = runtime.NewScheme()
	setupLog        = ctrl.Log.WithName("setup")
	metricsAddr     string
	healthzAddr     string
	webhookPort     int
	webhookCertDir  string
	webhookClientCA string // filename in CertDir (e.g. "ca.crt")
)

func init() {
	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339TimeEncoder,
	}
	setupLog = zap.New(zap.UseFlagOptions(&opts))
	ctrl.SetLogger(setupLog)

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(greenhousev1alpha1.AddToScheme(scheme))
	utilruntime.Must(greenhousev1alpha2.AddToScheme(scheme))

	flag.StringVar(&metricsAddr, "metrics-addr", ":6543", "bind address for metrics")
	flag.StringVar(&healthzAddr, "healthz-addr", ":8081", "bind address for health checks")

	flag.IntVar(&webhookPort, "webhook-port", 9443, "Webhook server port")
	flag.StringVar(&webhookCertDir, "webhook-cert-dir",
		"/tmp/k8s-webhook-server/serving-certs", "Webhook cert directory")
	flag.StringVar(&webhookClientCA, "webhook-client-ca-name", "ca.crt",
		"Client CA filename in cert dir used to validate client certs (mtls only)")
}

func main() {
	setupLog.Info("Authorization Webhook", "version", version.GitCommit, "build_date", version.BuildDate, "go", version.GoVersion)

	flag.Parse()

	authzTLSValue := os.Getenv(authzTLS)
	secure := true
	var err error
	if strings.TrimSpace(authzTLSValue) != "" {
		secure, err = strconv.ParseBool(authzTLSValue)
		handleError(err, "unable to parse "+authzTLS)
	}

	var webhookServer webhook.Server

	if secure {
		// By default it takes tls.crt and tls.key from CertDir.
		webhookServer = webhook.NewServer(webhook.Options{
			Port:         webhookPort,
			CertDir:      webhookCertDir,
			ClientCAName: webhookClientCA, // enables client cert verification for mTLS
		})
	}

	metricsServerOptions := metricsserver.Options{
		BindAddress: metricsAddr,
	}

	mgrOptions := ctrl.Options{
		Logger:                 setupLog,
		Scheme:                 scheme,
		Metrics:                metricsServerOptions,
		HealthProbeBindAddress: healthzAddr,
		LeaderElection:         false,
	}

	if secure {
		mgrOptions.WebhookServer = webhookServer
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), mgrOptions)
	handleError(err, "Failed to create manager")

	// Register the authorizer webhook.
	client := mgr.GetClient()
	mapper := mgr.GetRESTMapper()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleAuthorize(w, r, client, mapper)
	})

	if secure {
		mgr.GetWebhookServer().Register("/authorize", handler)
	} else {
		setupLog.Info("Setting up insecure HTTP server")
		handleError(addInsecureWebhookServer(mgr, webhookPort, "/authorize", handler), "Failed to add insecure webhook server")
	}

	handleError(mgr.AddHealthzCheck("healthz", healthz.Ping), "Failed to set up health check")
	handleError(mgr.AddReadyzCheck("readyz", healthz.Ping), "Failed to set up ready check")

	handleError(mgr.Start(ctrl.SetupSignalHandler()), "Failed to start manager")
}

func handleError(err error, msg string) {
	if err != nil {
		setupLog.Error(err, msg)
		os.Exit(1)
	}
}

func addInsecureWebhookServer(mgr manager.Manager, port int, path string, handler http.Handler) error {
	return mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
		mux := http.NewServeMux()
		mux.Handle(path, handler)

		addr := ":" + strconv.Itoa(port)
		srv := &http.Server{Addr: addr, Handler: mux}

		errCh := make(chan error, 1)
		go func() { errCh <- srv.ListenAndServe() }()

		select {
		case <-ctx.Done():
			// stop with manager
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = srv.Shutdown(shutdownCtx)
			return nil
		case err := <-errCh:
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			return err // notify manager to exit
		}
	}))
}
