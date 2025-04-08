// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/oklog/run"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	greenhousehealthz "github.com/cloudoperators/greenhouse/internal/healthz"
	"github.com/cloudoperators/greenhouse/internal/version"
)

func main() {
	var kubecontext, kubenamespace string
	var listenAddr, metricsAddr, healthzAddr string

	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339TimeEncoder,
	}
	logger := zap.New(zap.UseFlagOptions(&opts))

	logger.Info("Service-proxy", "version", version.GitCommit, "build_date", version.BuildDate, "go", version.GoVersion)

	var failWithError = func(err error, message string) {
		logger.Error(err, message)
		os.Exit(1)
	}
	// --kubeconfig in ctrl package
	flag.StringVar(&kubecontext, "kubecontext", os.Getenv("KUBECONTEXT"), "Use context from kubeconfig")
	flag.StringVar(&kubenamespace, "kubenamespace", os.Getenv("KUBENAMESPACE"), "Use namespace")
	flag.StringVar(&listenAddr, "listen-addr", ":8080", "proxy listen address")
	flag.StringVar(&metricsAddr, "metrics-addr", ":6543", "bind address for metrics")
	flag.StringVar(&healthzAddr, "healz-addr", ":8081", "bind address for health checks")
	flag.Parse()

	k8sConfig, err := ctrlconfig.GetConfigWithContext(kubecontext)
	if err != nil {
		failWithError(err, "Failed to create k8s config")
	}
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(greenhousev1alpha1.AddToScheme(scheme))

	mgr, err := ctrl.NewManager(k8sConfig, ctrl.Options{
		Logger: logger,
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: healthzAddr,
		LeaderElection:         false,
		Cache: cache.Options{
			Scheme:            scheme,
			DefaultNamespaces: map[string]cache.Config{kubenamespace: {}},
		},
	})

	if err != nil {
		failWithError(err, "Failed to create manager")
	}
	if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
		failWithError(err, "Failed to add ping healthz check to manager")
	}
	if err := mgr.AddReadyzCheck("informer-sync", greenhousehealthz.NewCacheSyncHealthz(mgr.GetCache())); err != nil {
		failWithError(err, "Failed to add readiness healthz check to manager")
	}

	pm := NewProxyManager()

	if err := pm.SetupWithManager("proxymanager", mgr); err != nil {
		failWithError(err, "Failed to setup proxy manager")
	}

	var g run.Group

	// Add signal handler
	g.Add(run.SignalHandler(context.Background(), syscall.SIGINT, syscall.SIGTERM))

	ctx, cancelMgr := context.WithCancel(context.Background())
	g.Add(
		func() error {
			return mgr.Start(ctx)
		},
		func(_ error) {
			cancelMgr()
		})

	frontend := http.Server{
		Addr:    listenAddr,
		Handler: InstrumentHandler(pm, metrics.Registry),
	}

	g.Add(
		func() error {
			logger.Info("starting listener", "addr", frontend.Addr)
			return frontend.ListenAndServe()
		},
		func(_ error) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			frontend.Shutdown(ctx) //nolint:errcheck
			cancel()
		})

	err = g.Run()
	var signalErr run.SignalError
	if ok := errors.As(err, &signalErr); ok {
		return
	}
	failWithError(err, "Exited unclean")
}
