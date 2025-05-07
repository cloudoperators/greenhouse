// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	goflag "flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	helmcontroller "github.com/fluxcd/helm-controller/api/v2"
	sourcecontroller "github.com/fluxcd/source-controller/api/v1"

	flag "github.com/spf13/pflag"
	"go.uber.org/zap/zapcore"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	greenhousesapv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/common"
	dexapi "github.com/cloudoperators/greenhouse/pkg/dex/api"
	"github.com/cloudoperators/greenhouse/pkg/features"
	"github.com/cloudoperators/greenhouse/pkg/helm"
	"github.com/cloudoperators/greenhouse/pkg/version"
)

type managerMode int

const (
	// regularMode starts Manager with registered Controllers and all Webhooks
	regularMode managerMode = iota
	// webhookOnlyMode starts the Manager with all Webhooks and no Controllers
	webhookOnlyMode
	// controllerOnlyMode starts the Manager with registered Controllers and no Webhooks
	controllerOnlyMode
)

const (
	defaultRemoteClusterBearerTokenValidity   = 24 * time.Hour
	defaultRenewRemoteClusterBearerTokenAfter = 20 * time.Hour
	disableControllersEnv                     = "WEBHOOK_ONLY"             // used to deploy the operator in webhook only mode no controllers will run in this mode.
	disableWebhookEnv                         = "CONTROLLERS_ONLY"         // used to disable webhooks when running locally or in debug mode.
	podNamespaceEnv                           = "POD_NAMESPACE"            // used to read the pod namespace from the environment.
	defaultPodNamespace                       = "greenhouse"               // default pod namespace.
	featureFlagsEnv                           = "FEATURE_FLAGS"            // used to read the feature flags configMap name from the environment.
	defaultFeatureFlagConfigMapName           = "greenhouse-feature-flags" // default feature flags configMap name.
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")

	enabledControllers []string
	remoteClusterBearerTokenValidity,
	renewRemoteClusterBearerTokenAfter time.Duration
	kubeClientOpts clientutil.RuntimeOptions
	featureFlags   *features.Features
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(greenhousesapv1alpha1.AddToScheme(scheme))
	utilruntime.Must(dexapi.AddToScheme(scheme))
	utilruntime.Must(sourcecontroller.AddToScheme(scheme))
	utilruntime.Must(helmcontroller.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	flag.BoolVar(&helm.IsHelmDebug, "helm-debug", false,
		"Enable debug logging for underlying Helm client.")
	flag.StringSliceVar(&enabledControllers, "controllers", knownControllersNames(),
		"A list of controllers to enable.")

	var metricsAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080",
		"The address the metric endpoint binds to.")

	var probeAddr string
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081",
		"The address the probe endpoint binds to.")

	flag.DurationVar(&remoteClusterBearerTokenValidity, "remote-cluster-bearer-token-validity", defaultRemoteClusterBearerTokenValidity,
		"Validity of the bearer token we request to access the remote clusters")

	flag.DurationVar(&renewRemoteClusterBearerTokenAfter, "renew-remote-cluster-bearer-token-after", defaultRenewRemoteClusterBearerTokenAfter,
		"Renew the bearer token we requested for remote clusters after this duration")

	flag.StringVar(&common.DNSDomain, "dns-domain", "",
		"The DNS domain to use for the Greenhouse central cluster")

	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339TimeEncoder,
	}

	goFlagSet := goflag.CommandLine
	opts.BindFlags(goFlagSet)
	flag.CommandLine.AddGoFlagSet(goFlagSet)
	kubeClientOpts.BindFlags(flag.CommandLine)
	flag.Parse()

	version.ShowVersionAndExit("greenhouse")

	ctrl.SetLogger(zap.New(
		zap.UseFlagOptions(&opts)),
	)

	if common.DNSDomain == "" {
		handleError(errors.New("--dns-domain must not be empty"), "unable to start controller")
	}

	// Disable leader election if not run within a cluster.
	isEnableLeaderElection := true
	if _, ok := os.LookupEnv(podNamespaceEnv); !ok {
		isEnableLeaderElection = false
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress:        probeAddr,
		LeaderElection:                isEnableLeaderElection,
		LeaderElectionID:              "operator.greenhouse.sap",
		LeaderElectionReleaseOnCancel: true,
	})
	handleError(err, "unable to start manager")

	// extract the manager API Client Reader
	// Note: mgr.GetClient() will fail here because the cache is not ready yet
	k8sClient := mgr.GetAPIReader()
	// Initialize the feature gates from feature-flags config map
	featureFlags, err = features.NewFeatures(
		context.TODO(),
		k8sClient,
		clientutil.GetEnvOrDefault(featureFlagsEnv, defaultFeatureFlagConfigMapName),
		clientutil.GetEnvOrDefault(podNamespaceEnv, defaultPodNamespace),
	)
	if err != nil {
		handleError(err, "unable to get features")
	}

	mode, err := calculateManagerMode()
	if err != nil {
		handleError(err, "unable to calculate manager mode")
	}

	// Register controllers.
	if mode != webhookOnlyMode {
		for controllerName, hookFunc := range knownControllers {
			if !isControllerEnabled(controllerName) {
				setupLog.Info("skipping controller", "name", controllerName)
				continue
			}
			setupLog.Info("registering controller", "name", controllerName)
			handleError(hookFunc(controllerName, mgr), "unable to create controller", "name", controllerName)
			continue
		}
	}

	// Register webhooks.
	if mode != controllerOnlyMode {
		for webhookName, hookFunc := range knownWebhooks {
			setupLog.Info("registering webhook", "name", webhookName)
			handleError(hookFunc(mgr), "unable to create webhook", "name", webhookName)
		}
	}
	//+kubebuilder:scaffold:builder

	handleError(mgr.AddHealthzCheck("healthz", healthz.Ping), "unable to set up health check")
	handleError(mgr.AddReadyzCheck("readyz", healthz.Ping), "unable to set up ready check")

	setupLog.Info("starting manager")
	handleError(mgr.Start(ctrl.SetupSignalHandler()), "problem running manager")
}

func handleError(err error, msg string, keysAndValues ...interface{}) {
	if err != nil {
		setupLog.Error(err, msg, keysAndValues...)
		os.Exit(1)
	}
}

// calculateManagerMode - calculates in which mode the manager should run.
func calculateManagerMode() (managerMode, error) {
	webhookOnlyEnv := os.Getenv(disableControllersEnv)
	controllersOnlyEnv := os.Getenv(disableWebhookEnv)

	var webhookOnly, controllersOnly bool
	var err error

	if strings.TrimSpace(webhookOnlyEnv) != "" {
		webhookOnly, err = strconv.ParseBool(webhookOnlyEnv)
		if err != nil {
			return -1, fmt.Errorf("unable to parse %s: %w", disableControllersEnv, err)
		}
	}

	if strings.TrimSpace(controllersOnlyEnv) != "" {
		controllersOnly, err = strconv.ParseBool(controllersOnlyEnv)
		if err != nil {
			return -1, fmt.Errorf("unable to parse %s: %w", disableWebhookEnv, err)
		}
	}

	if webhookOnly && controllersOnly {
		return -1, errors.New("you can have only one of WEBHOOK_ONLY or CONTROLLERS_ONLY env be set to true")
	}

	if webhookOnly {
		return webhookOnlyMode, nil
	}

	if controllersOnly {
		return controllerOnlyMode, nil
	}

	return regularMode, nil
}
