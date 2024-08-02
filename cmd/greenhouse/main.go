// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	goflag "flag"
	"fmt"
	"os"
	"time"

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
	"github.com/cloudoperators/greenhouse/pkg/helm"
	"github.com/cloudoperators/greenhouse/pkg/version"
)

const (
	defaultRemoteClusterBearerTokenValidity   = 24 * time.Hour
	defaultRenewRemoteClusterBearerTokenAfter = 20 * time.Hour
)
const (
	scimBasicAuthUserEnvKey string = "SCIM_BASIC_AUTH_USER"
	scimBasicAuthPwEnvKey   string = "SCIM_BASIC_AUTH_PW" //nolint:gosec
	scimBaseURLEnvKey       string = "SCIM_BASE_URL"      //nolint:gosec
	namespaceEnvKey         string = "NAMESPACE"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")

	enabledControllers []string
	headscaleAPIURL,
	headscaleAPIKey,
	tailscaleProxy string
	remoteClusterBearerTokenValidity,
	renewRemoteClusterBearerTokenAfter time.Duration
	kubeClientOpts clientutil.RuntimeOptions
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(greenhousesapv1alpha1.AddToScheme(scheme))
	utilruntime.Must(dexapi.AddToScheme(scheme))
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

	flag.StringVar(&headscaleAPIURL, "headscale-api-url", clientutil.GetEnvOrDefault("HEADSCALE_API_URL", ""),
		"Headscale API URL.(format https://<url>) Can be set via HEADSCALE_API_URL env var")

	flag.StringVar(&headscaleAPIKey, "headscale-api-key", clientutil.GetEnvOrDefault("HEADSCALE_API_KEY", ""),
		"Headscale API KEY. Can be set via HEADSCALE_API_KEY env var")

	flag.StringVar(&tailscaleProxy, "tailscale-proxy", clientutil.GetEnvOrDefault("TAILSCALE_PROXY", ""),
		"Tailscale proxy to be used by Greenhouse in case of type the communication is not direct. Can be set via TAILSCALE_PROXY env var")

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
		handleError(fmt.Errorf("--dns-domain must not be empty"), "unable to start controller")
	}

	// Disable leader election if not run within a cluster.
	isEnableLeaderElection := true
	if _, ok := os.LookupEnv("POD_NAMESPACE"); !ok {
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

	// Register controllers.
	for controllerName, hookFunc := range knownControllers {
		if !isControllerEnabled(controllerName) {
			setupLog.Info("skipping controller", "name", controllerName)
			continue
		}
		setupLog.Info("registering controller", "name", controllerName)
		handleError(hookFunc(controllerName, mgr), "unable to create controller", "name", controllerName)
		continue
	}

	// Register webhooks.
	for webhookName, hookFunc := range knownWebhooks {
		setupLog.Info("registering webhook", "name", webhookName)
		handleError(hookFunc(mgr), "unable to create webhook", "name", webhookName)
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
