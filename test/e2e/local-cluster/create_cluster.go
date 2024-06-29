// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	kubernetesTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
	kubernetesClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/wait"

	"sigs.k8s.io/e2e-framework/support/kind"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

var (
	kindClusterNameEnvVar = "TEST_E2E_KIND_CLUSTER_NAME"
	kindClusterName       = "greenhouse-e2e"

	kubeconfigNameEnvVar = "TEST_E2E_KUBECONFIG"
	kubeconfigName       = "e2e.kubeconfig"

	kubeconfigInternalNameEnvVar = "TEST_E2E_DOCKER_INTERNAL_NETWORK_KUBECONFIG"
	kubeconfigInternalName       = "e2e.internal.kubeconfig"

	dockerImageRepositoryEnvVar = "TEST_E2E_DOCKER_IMAGE_REPOSITORY"
	dockerImageRepository       = "greenhouse"

	dockerImageTagEnvVar = "TEST_E2E_DOCKER_IMAGE_TAG"
	dockerImageTag       = "e2e-latest"

	dockerImageBuildSkipEnvVar = "TEST_E2E_DOCKER_IMAGE_BUILD_SKIP"
	dockerImageBuildSkip       = false

	greenhouseControllerManagerNamespace = "greenhouse"
	greenhouseControllerManagerRelease   = "greenhouse"
	greenhouseOrganizationName           = "e2e-org"
)

const (
	TEST_TIMEOUT        = 3 * time.Minute
	TEST_RETRY_INTERVAL = 3 * time.Second
)

func init() {

	l := log.FromContext(context.Background())

	if os.Getenv(kindClusterNameEnvVar) != "" {
		kindClusterName = os.Getenv(kindClusterNameEnvVar)
	}

	if os.Getenv(dockerImageRepositoryEnvVar) != "" {
		dockerImageRepository = os.Getenv(dockerImageRepositoryEnvVar)
	}

	if os.Getenv(dockerImageTagEnvVar) != "" {
		dockerImageTag = os.Getenv(dockerImageTagEnvVar)
	}

	if os.Getenv(dockerImageBuildSkipEnvVar) != "" {
		dockerImageBuildSkip = true
	}
	if os.Getenv(kubeconfigNameEnvVar) != "" {
		kubeconfigName = os.Getenv(kubeconfigNameEnvVar)
	}

	if os.Getenv(kubeconfigInternalNameEnvVar) != "" {
		kubeconfigInternalName = os.Getenv(kubeconfigInternalNameEnvVar)
	}

	l.Info("configuration loaded", "kindClusterName", kindClusterName, "dockerImageRepository", dockerImageRepository, "dockerImageTag", dockerImageTag, "dockerImageBuildSkip", dockerImageBuildSkip, "kubeconfigName", kubeconfigName, "kubeconfigInternalName", kubeconfigInternalName)

}

func main() {

	ctx := context.Background()
	l := log.FromContext(ctx)

	// Create cluster
	cluster := kind.NewCluster(kindClusterName)
	cluster.SetDefaults()
	kubeconfig, err := cluster.Create(ctx)
	if err != nil {
		l.Error(err, "state", "cluster creation")
		os.Exit(1)
	}
	l.Info("cluster created successfully")

	// Export kubeconfig
	f, err := os.Create(kubeconfigName)
	if err != nil {
		l.Error(err, "state", "kubeconfig creation")
		os.Exit(1)
	}
	args := []string{"export", "kubeconfig", "--name", kindClusterName, "--kubeconfig", f.Name()}
	cmd := exec.Command("kind", args...)
	_, err = cmd.Output()
	if err != nil {
		l.Error(err, "state", "kubeconfig export")
		os.Exit(1)
	}
	f, err = os.Create(kubeconfigInternalName)
	if err != nil {
		l.Error(err, "state", "kubeconfig creation")
		os.Exit(1)
	}
	args = []string{"export", "kubeconfig", "--name", kindClusterName, "--internal", "--kubeconfig", f.Name()}
	cmd = exec.Command("kind", args...)
	_, err = cmd.Output()
	if err != nil {
		l.Error(err, "state", "kubeconfig export")
		os.Exit(1)
	}

	l.Info("kubeconfig exported successfully")

	// Build image
	image := fmt.Sprintf("%s:%s", dockerImageRepository, dockerImageTag)
	if !dockerImageBuildSkip {
		err = dockerImageBuild("./../../../", image)
		if err != nil {
			l.Error(err, "state", "image build")
			os.Exit(1)
		}
		l.Info("Docker image built successfully")
	}

	// Load image
	err = cluster.LoadImage(ctx, image)
	if err != nil {
		l.Error(err, "state", "image load")
		os.Exit(1)
	}
	l.Info("Docker image loaded to the cluster successfully")

	// Create Kubernetes client
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigName)
	if err != nil {
		l.Error(err, "state", "client config")
		os.Exit(1)
	}
	k8sClient, err := clientutil.NewK8sClient(config)
	if err != nil {
		l.Error(err, "state", "k8s client")
		os.Exit(1)
	}

	// Deploy Greenhouse manager
	err = installChart(ctx, "./../../../charts/manager", greenhouseControllerManagerRelease, kubeconfig, greenhouseControllerManagerNamespace)
	if err != nil {
		l.Error(err, "state", "deploy greenhouse")
		os.Exit(1)
	}
	l.Info("Greenhouse manager is deployed successfully")

	// Deploy test org
	err = deployTestOrganization(context.TODO(), k8sClient)
	if err != nil {
		l.Error(err, "state", "deploy test org")
		os.Exit(1)
	}
	l.Info("Test organization is deployed and checked successfully")

}

func installChart(ctx context.Context, dir, release, kubeconfig string, namespace string) error {

	l := log.FromContext(ctx)

	chart, err := loader.Load(dir)
	if err != nil {
		return err
	}

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(
		&genericclioptions.ConfigFlags{
			KubeConfig: &kubeconfig,
			Namespace:  &namespace,
		},
		namespace,
		"secret",
		l.V(10).Info,
	); err != nil {
		return err
	}

	//TODO(onur): make a separate function to edit values
	globalDNS := map[string]interface{}{"dnsDomain": "greenhouse.cloudoperators"}
	chart.Values["global"] = globalDNS

	controllerManagerImage := map[string]interface{}{"repository": dockerImageRepository, "tag": dockerImageTag}

	controllerManager, ok := chart.Values["controllerManager"].(map[string]interface{})
	if !ok {
		l.Error(err, "state", "value merge")
		os.Exit(1)
	}
	controllerManager["image"] = controllerManagerImage
	controllerManager["replicas"] = "1"
	chart.Values["controllerManager"] = controllerManager

	get := action.NewGet(actionConfig)
	_, err = get.Run(release)
	if err != nil {
		if err.Error() == "release: not found" {
			client := action.NewInstall(actionConfig)
			client.ReleaseName = release
			client.Namespace = namespace
			client.CreateNamespace = true
			client.Wait = true
			client.Timeout = TEST_TIMEOUT

			if _, err := client.Run(chart, chart.Values); err != nil {
				l.Error(err, "state", "chart install")
				return err
			}
		}
	} else {
		client := action.NewUpgrade(actionConfig)
		client.Namespace = namespace
		client.Wait = true
		client.Timeout = TEST_TIMEOUT
		if _, err := client.Run(release, chart, chart.Values); err != nil {
			l.Error(err, "state", "chart upgrade")
			return err
		}
	}

	return nil
}

func dockerImageBuild(path string, repoAndtag string) error {

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*120)
	defer cancel()

	tar, err := archive.TarWithOptions(path, &archive.TarOptions{})
	if err != nil {
		return err
	}

	opts := types.ImageBuildOptions{
		Dockerfile:     "Dockerfile",
		Tags:           []string{repoAndtag},
		Version:        types.BuilderBuildKit,
		Platform:       "linux/amd64",
		SuppressOutput: true,
	}
	_, err = dockerClient.ImageBuild(ctx, tar, opts)
	return err
}

func deployTestOrganization(ctx context.Context, client kubernetesClient.Client) error {

	l := log.FromContext(ctx)

	org := &greenhousev1alpha1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			Name:      greenhouseOrganizationName,
			Namespace: "default",
		},
		Spec: greenhousev1alpha1.OrganizationSpec{
			DisplayName: greenhouseOrganizationName,
			Description: "Organization created for the e2e tests",
		},
	}
	err := client.Get(ctx, kubernetesTypes.NamespacedName{Namespace: org.Namespace, Name: org.Name}, org)
	if err == nil { //it exists already
		l.Info("Organization already exists")
	} else {
		err := client.Create(ctx, org)
		if err != nil {
			l.Error(err, "state", "organization creation")
			return err
		}
	}

	if err := wait.For(
		func(context.Context) (done bool, err error) {
			namespace := &corev1.Namespace{}
			err = client.Get(ctx, kubernetesTypes.NamespacedName{Name: greenhouseOrganizationName}, namespace)
			if apierrors.IsNotFound(err) {
				l.Info("Waiting for namespace creation for organization...")
				return false, nil
			} else if err != nil {
				l.Error(err, "state", "namespace creation")
				return false, err
			}
			l.Info("Namespace is created automatically for organization")
			return true, nil
		},
		wait.WithTimeout(TEST_TIMEOUT),
		wait.WithInterval(TEST_RETRY_INTERVAL),
	); err != nil {
		return err
	}
	return nil
}
