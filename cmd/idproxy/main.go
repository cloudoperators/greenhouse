// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/dexidp/dex/server"
	"github.com/go-logr/logr"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	flag "github.com/spf13/pflag"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client/config"
	logk "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/dex"
	dexstore "github.com/cloudoperators/greenhouse/pkg/dex/store"
	"github.com/cloudoperators/greenhouse/pkg/dex/web"
	"github.com/cloudoperators/greenhouse/pkg/features"
)

func main() {
	var issuer string
	var idTokenValidity time.Duration
	var listenAddr, metricsAddr string
	var allowedOrigins []string
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	// set default logger to be used by log
	slog.SetDefault(logger)
	// set default deferred logger to be used by controller-runtime
	logk.SetLogger(logr.FromSlogHandler(logger.Handler()))

	flag.StringVar(&issuer, "issuer", "", "Issuer URL")
	flag.StringVar(&listenAddr, "listen-addr", ":8080", "oidc listen address")
	flag.StringVar(&metricsAddr, "metrics-addr", ":6543", "bind address for metrics")
	flag.StringSliceVar(&allowedOrigins, "allowed-origins", []string{"*"}, "list of allowed origins for CORS requests on discovery, token and keys endpoint")
	flag.DurationVar(&idTokenValidity, "id-token-validity", 1*time.Hour, "Maximum validity of issued id tokens")
	flag.Parse()

	if issuer == "" {
		log.Fatal("No --issuer given")
	}

	// ctrl.GetConfigOrDie() is used to get the k8s client config depending on the environment
	// In Cluster config is used when running in a k8s cluster else uses the kubeconfig file specified by the KUBECONFIG env variable
	restCfg := ctrl.GetConfigOrDie()
	ctx := context.TODO()
	k8sClient, err := clientutil.NewK8sClient(restCfg)
	if err != nil {
		log.Fatalf("failed to create k8s client: %s", err)
	}
	// default to kubernetes storage backend
	backend := clientutil.Ptr("kubernetes")
	ghFeatures, err := features.NewFeatures(ctx, k8sClient)
	if err != nil {
		log.Fatalf("failed to get greenhouse features: %s", err)
	}
	if ghFeatures != nil {
		backend = ghFeatures.GetDexStorageType(ctx)
	}
	// initialize dex storage adapter interface
	dexter, err := dexstore.NewDexStorageFactory(logger.With("component", "storage"), *backend)
	if err != nil {
		log.Fatalf("failed to create dex storage interface: %s", err)
	}
	logger.Info("using dex storage - ", "type", *backend)
	// get the underlying dex storage interface
	dexStorage := dexter.GetStorage()

	refreshPolicy, err := server.NewRefreshTokenPolicy(logger.With("component", "refreshtokenpolicy"), true, "24h", "24h", "5s")
	if err != nil {
		log.Fatalf("Failed to setup refresh token policy: %s", err)
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	registry.MustRegister(collectors.NewGoCollector())

	config := server.Config{
		Issuer:             issuer,
		SkipApprovalScreen: true,
		Logger:             logger.With("component", "server"),
		Storage:            dexStorage,
		AllowedOrigins:     allowedOrigins,
		IDTokensValidFor:   idTokenValidity,
		RefreshTokenPolicy: refreshPolicy,
		PrometheusRegistry: registry,
		Web: server.WebConfig{
			WebFS: web.FS(),
		},
	}

	server.ConnectorsConfig["greenhouse-oidc"] = func() server.ConnectorConfig {
		oidcConfig := new(dex.OIDCConfig)
		oidcConfig.AddClient(k8sClient)
		oidcConfig.AddRedirectURI(issuer + "/callback")

		return oidcConfig
	}

	var g run.Group

	// Add signal handler
	g.Add(run.SignalHandler(context.Background(), syscall.SIGINT, syscall.SIGTERM))

	// oidc server
	ctx, cancel := context.WithCancel(context.Background())
	serv, err := server.NewServer(ctx, config)
	if err != nil {
		log.Fatalf("OIDC server setup failed: %s", err)
	}
	s := &http.Server{Handler: serv}
	g.Add(func() error {
		ln, err := net.Listen("tcp", listenAddr)
		if err != nil {
			return fmt.Errorf("failed to listen on %s: %w", listenAddr, err)
		}
		logger.Info("OIDC server listening ", "address", listenAddr)
		return s.Serve(ln)
	}, func(_ error) {
		cancel()
		timeoutCtx, c := context.WithTimeout(context.Background(), 5*time.Second)
		defer c()
		s.Shutdown(timeoutCtx) //nolint: errcheck
	})

	// metrics server
	ms := &http.Server{Handler: promhttp.HandlerFor(registry, promhttp.HandlerOpts{Registry: registry})}
	g.Add(func() error {
		ln, err := net.Listen("tcp", metricsAddr)
		if err != nil {
			return fmt.Errorf("failed to listen on %s: %w", metricsAddr, err)
		}
		logger.Info("Metrics server listing", "address", metricsAddr)
		return ms.Serve(ln)
	}, func(_ error) {
		ms.Close()
	})

	err = g.Run()
	var signalErr run.SignalError
	if ok := errors.As(err, &signalErr); ok {
		return
	}
	log.Fatal(err)
}
