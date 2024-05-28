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
	"k8s.io/klog/v2"
	kubernetesClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/wait"

	"sigs.k8s.io/e2e-framework/support/kind"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

	klog.Info("========== CONFIGURATION ==========")
	klog.Infof("Cluster name: %s", kindClusterName)
	klog.Infof("Greenhouse manager container image: %s:%s", dockerImageRepository, dockerImageTag)
	if dockerImageBuildSkip {
		klog.Infof("Docker build will be skipped..")
	}
	klog.Infof("kubeconfig will be exported to the files: %s %s", kubeconfigName, kubeconfigInternalName)
	klog.Info("========== CONFIGURATION ==========")

}

func main() {

	ctx := context.Background()

	// Create cluster
	cluster := kind.NewCluster(kindClusterName)
	cluster.SetDefaults()
	kubeconfig, err := cluster.Create(ctx)
	if err != nil {
		klog.Fatal("Error during cluster creation:", err)
	}
	klog.Info("cluster created successfully")

	// Export kubeconfig
	f, err := os.Create(kubeconfigName)
	if err != nil {
		klog.Fatal("Error during file creation:", err)
	}
	args := []string{"export", "kubeconfig", "--name", kindClusterName, "--kubeconfig", f.Name()}
	cmd := exec.Command("kind", args...)
	_, err = cmd.Output()
	if err != nil {
		klog.Fatal("Error during kubeconfig export:", err)
	}
	f, err = os.Create(kubeconfigInternalName)
	if err != nil {
		klog.Fatal("Error during file creation:", err)
	}
	args = []string{"export", "kubeconfig", "--name", kindClusterName, "--internal", "--kubeconfig", f.Name()}
	cmd = exec.Command("kind", args...)
	_, err = cmd.Output()
	if err != nil {
		klog.Fatal("Error during kubeconfig export:", err)
	}

	klog.Info("kubeconfig exported successfully")

	// Build image
	image := fmt.Sprintf("%s:%s", dockerImageRepository, dockerImageTag)
	if !dockerImageBuildSkip {
		err = dockerImageBuild("./../../../", image)
		if err != nil {
			klog.Fatal("Error during build image: ", err)
		}
		klog.Info("Docker image built successfully")
	}

	// Load image
	err = cluster.LoadImage(ctx, image)
	if err != nil {
		klog.Fatal("Error during load image: ", err)
	}
	klog.Info("Docker image loaded to the cluster successfully")

	// Create Kubernetes client
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigName)
	if err != nil {
		klog.Fatal("Error during creating config:", err)
	}
	k8sClient, err := clientutil.NewK8sClient(config)
	if err != nil {
		klog.Fatal("Error during creating Kubernetes client:", err)
	}

	// Deploy Greenhouse manager
	err = installChart("./../../../charts/manager", "greenhouse", kubeconfig, "greenhouse")
	if err != nil {
		klog.Fatal(err)
	}
	klog.Infof("Greenhouse manager is deployed successfully")

	// Deploy test org
	err = deployTestOrganization(context.TODO(), k8sClient)
	if err != nil {
		klog.Fatal(err)
	}
	klog.Infof("Test organization is deployed and checked successfully")

}

func installChart(dir, release, kubeconfig string, namespace string) error {
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
		klog.V(10).Infof, //TODO(onur) debug log
	); err != nil {
		return err
	}

	//TODO(onur): make a separate function to edit values
	globalDNS := map[string]interface{}{"dnsDomain": "greenhouse.cloudoperators"}
	chart.Values["global"] = globalDNS

	controllerManagerImage := map[string]interface{}{"repository": dockerImageRepository, "tag": dockerImageTag}

	controllerManager, ok := chart.Values["controllerManager"].(map[string]interface{})
	if !ok {
		klog.Fatal("failed in value merge")
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
				return err
			}
		}
	} else {
		client := action.NewUpgrade(actionConfig)
		client.Namespace = namespace
		client.Wait = true
		client.Timeout = TEST_TIMEOUT
		if _, err := client.Run(release, chart, chart.Values); err != nil {
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
		klog.Info("Organization already exists")
	} else {
		err := client.Create(ctx, org)
		if err != nil {
			return err
		}
	}

	if err := wait.For(
		func(context.Context) (done bool, err error) {
			namespace := &corev1.Namespace{}
			err = client.Get(ctx, kubernetesTypes.NamespacedName{Name: greenhouseOrganizationName}, namespace)
			if apierrors.IsNotFound(err) {
				klog.Info("Waiting for namespace creation for organization...")
				return false, nil
			} else if err != nil {
				return false, err
			}
			klog.Info("Namespace is created automatically for organization")
			return true, nil
		},
		wait.WithTimeout(TEST_TIMEOUT),
		wait.WithInterval(TEST_RETRY_INTERVAL),
	); err != nil {
		return err
	}
	return nil
}
