// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/common-nighthawk/go-figure"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap/zapcore"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/cloudoperators/greenhouse/internal/proxy"
	"github.com/cloudoperators/greenhouse/internal/version"
)

var (
	listenAddr  string
	metricsAddr string
)

func main() {
	flag.StringVar(&listenAddr, "listen-addr", ":8080", "proxy listen address")
	flag.StringVar(&metricsAddr, "metrics-addr", ":6543", "metrics listen address")
	flag.Parse()

	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339TimeEncoder,
	}
	logger := zap.New(zap.UseFlagOptions(&opts))
	ctrl.SetLogger(logger)

	figure.NewColorFigure("Greenhouse Service Proxy", "doom", "green", true).Print()
	figure.NewColorFigure("Version: "+version.GitCommit+" ("+version.BuildDate+")", "term", "yellow", true).Print()
	logger.Info("Service-proxy", "version", version.GitCommit, "build_date", version.BuildDate, "go", version.GoVersion)

	// Signal-canceled context drives the informer cache.
	ctx := ctrl.SetupSignalHandler()

	pm, err := proxy.NewProxyManager(ctx, logger)
	if err != nil {
		logger.Error(err, "failed to create proxy manager")
		os.Exit(1)
	}

	// Start the cache in the background; the proxy serves once it is warm.
	go func() {
		if err := pm.Start(ctx); err != nil {
			logger.Error(err, "cache stopped")
			os.Exit(1)
		}
	}()

	if err := pm.WaitForCacheSync(ctx); err != nil {
		logger.Error(err, "failed to sync cache")
		os.Exit(1)
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(middleware.Recover())

	registry := prometheus.NewRegistry()
	pm.RegisterRoutes(e, registry)

	metrics := echo.New()
	metrics.HideBanner = true
	metrics.HidePort = true
	metrics.GET("/metrics", echo.WrapHandler(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})))
	metrics.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	metrics.GET("/readyz", func(c echo.Context) error {
		if !pm.Ready() {
			return c.String(http.StatusServiceUnavailable, "informers not synced yet")
		}
		return c.String(http.StatusOK, "ok")
	})

	go func() {
		if err := metrics.Start(metricsAddr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error(err, "metrics server exited unclean")
			os.Exit(1)
		}
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = e.Shutdown(shutdownCtx)
		_ = metrics.Shutdown(shutdownCtx)
	}()

	if err := e.Start(listenAddr); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error(err, "exited unclean")
		os.Exit(1)
	}
}
