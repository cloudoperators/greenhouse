// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/support/kind"
	"sigs.k8s.io/e2e-framework/support/utils"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/sirupsen/logrus"
)

const (
	TEST_TIMEOUT        = 3 * time.Minute
	TEST_RETRY_INTERVAL = 3 * time.Second
)

var (
	testEnv env.Environment
)

var (
	centralCluster          *kind.Cluster
	centralClusterName      string
	centralClusterK8sClient client.Client
)

var (
	greenhouseControllerManagerNamespace  = "greenhouse"
	greenhouseControllerManagerRelease    = "greenhouse"
	greenhouseControllerManagerDeployment = "greenhouse-controller-manager"
	greenhouseOrganizationName            = "e2e-org"
)

var (
	dockerImageBuildPath  = "./../../"
	dockerImageRepository = "greenhouse"
	dockerImageTag        = "e2e-latest"
	skipDockerImageBuild  = flag.Bool("skip-docker-image-build", false, "skip docker image build")
)

func TestMain(m *testing.M) {
	flag.Parse()
	testEnv = env.New()

	centralClusterName = envconf.RandomName("central-cluster", 0)
	centralCluster = kind.NewCluster(centralClusterName)

	testEnv.Setup(
		envfuncs.CreateCluster(centralCluster, centralClusterName),
		envfuncs.CreateNamespace(greenhouseControllerManagerNamespace),
		initTestEnvironmentVariables,
		buildDockerImage,
		envfuncs.LoadImageToCluster(centralClusterName, fmt.Sprintf("%s:%s", dockerImageRepository, dockerImageTag)),
		installGreenhouseControllerManager,
		deployTestOrganization,
	)

	testEnv.Finish(
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			return ctx, nil
		},
		envfuncs.DeleteNamespace(greenhouseControllerManagerNamespace),
		envfuncs.DestroyCluster(centralClusterName),
	)

	// Launch the test
	os.Exit(testEnv.Run(m))
}

func buildDockerImage(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	if !(*skipDockerImageBuild) {
		logrus.Infof("Building docker image: %s:%s", dockerImageRepository, dockerImageTag)
		command := fmt.Sprintf("docker build --platform linux/amd64 -t %s:%s %s ", dockerImageRepository, dockerImageTag, dockerImageBuildPath)
		p := utils.RunCommand(
			command,
		)
		err := p.Err()
		if err != nil {
			return ctx, p.Err()
		}
		logrus.Infof("Docker image built: %s:%s", dockerImageRepository, dockerImageTag)
	}
	return ctx, nil
}

func initTestEnvironmentVariables(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	logrus.Info("Initializing the test environment variables")
	var err error
	centralClusterK8sClient, err = clientutil.NewK8sClient(cfg.Client().RESTConfig())
	if err != nil {
		return ctx, err
	}

	return ctx, nil
}

func installGreenhouseControllerManager(ctx context.Context, cfg *envconf.Config) (context.Context, error) {

	logrus.Info("Installing Greenhouse Controller Manager to the central cluster")

	helmcfg := &action.Configuration{}
	restClientGetter := clientutil.NewRestClientGetterFromRestConfig(cfg.Client().RESTConfig(), cfg.Namespace())
	err := helmcfg.Init(restClientGetter, cfg.Namespace(), "secrets", func(format string, v ...interface{}) {
		logrus.Debugf(format, v...)
	})
	if err != nil {
		logrus.Error("Error initializing helm config: ", err)
		return ctx, err
	}

	installAction := action.NewInstall(helmcfg)
	installAction.ReleaseName = greenhouseControllerManagerRelease
	installAction.Namespace = cfg.Namespace()

	installAction.CreateNamespace = false
	installAction.DependencyUpdate = true
	chartsBasePath, err := clientutil.FindDirUpwards(".", "charts", 10)
	if err != nil {
		return ctx, err
	}
	chartPath := filepath.Join(chartsBasePath, "manager")
	chart, err := loader.Load(chartPath)
	if err != nil {
		return ctx, err
	}

	valuesYAML := fmt.Sprintf(
		`
global:
  dnsDomain: greenhouse.cloudoperators
controllerManager:
  replicas: 1
  image:
    repository: %s
    tag: %s
`, dockerImageRepository, dockerImageTag)

	values := make(map[string]interface{})
	err = yaml.Unmarshal([]byte(valuesYAML), values)
	if err != nil {
		return ctx, err
	}
	_, err = installAction.RunWithContext(ctx, chart, values)
	if err != nil {
		return ctx, err
	}

	if err := wait.For(
		conditions.New(cfg.Client().Resources()).DeploymentAvailable(greenhouseControllerManagerDeployment, cfg.Namespace()),
		wait.WithTimeout(TEST_TIMEOUT),
		wait.WithInterval(TEST_RETRY_INTERVAL),
	); err != nil {
		return ctx, err
	}

	logrus.Info("Installing Greenhouse Controller Manager to the central cluster: Done!")
	return ctx, nil
}

func deployTestOrganization(ctx context.Context, cfg *envconf.Config) (context.Context, error) {

	logrus.Info("Deploying organization resource to the central cluster")
	err := centralClusterK8sClient.Create(ctx,
		&greenhousev1alpha1.Organization{
			ObjectMeta: metav1.ObjectMeta{
				Name:      greenhouseOrganizationName,
				Namespace: "default",
			},
			Spec: greenhousev1alpha1.OrganizationSpec{
				DisplayName: greenhouseOrganizationName,
				Description: "Organization created for the e2e tests",
			},
		},
	)
	if err != nil {
		return ctx, err
	}

	if err := wait.For(
		func(context.Context) (done bool, err error) {
			namespace := &corev1.Namespace{}
			err = centralClusterK8sClient.Get(ctx, types.NamespacedName{Name: greenhouseOrganizationName}, namespace)
			if apierrors.IsNotFound(err) {
				return false, nil
			} else if err != nil {
				return false, err
			}
			logrus.Info("Namespace is created automatically for organization")
			return true, nil
		},
		wait.WithTimeout(TEST_TIMEOUT),
		wait.WithInterval(TEST_RETRY_INTERVAL),
	); err != nil {
		return ctx, err
	}
	return ctx, nil

}
