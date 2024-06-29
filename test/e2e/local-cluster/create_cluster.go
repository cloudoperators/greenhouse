// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"sigs.k8s.io/e2e-framework/support/kind"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	kindClusterName string

	kubeconfigName         string
	kubeconfigInternalName string

	dockerImageRepository string
	dockerImageTag        string

	dockerImageBuildSkip = false

	greenhouseControllerManagerNamespace string
	greenhouseControllerManagerRelease   string
)

const (
	TEST_TIMEOUT        = 3 * time.Minute
	TEST_RETRY_INTERVAL = 3 * time.Second
)

func init() {

	l := log.FromContext(context.Background())

	flag.StringVar(&kindClusterName, "kindClusterName", "greenhouse-e2e", "Cluster name for creating a new kind cluster")

	flag.StringVar(&kubeconfigName, "kubeconfigName", "e2e.kubeconfig", "kubeconfig file name for connecting to the e2e clusters")
	flag.StringVar(&kubeconfigInternalName, "kubeconfigInternalName", "e2e.internal.kubeconfig", "kubeconfig file name for connecting to the e2e clusters from the same Docker network")

	flag.StringVar(&dockerImageRepository, "dockerImageRepository", "greenhouse", "Docker image repository  for Greenhouse manager")
	flag.StringVar(&dockerImageTag, "dockerImageTag", "e2e-latest", "Docker image tag for Greenhouse manager")
	flag.BoolVar(&dockerImageBuildSkip, "dockerImageBuildSkip", false, "Skip building the docker image for Greenhouse manager")

	flag.StringVar(&greenhouseControllerManagerNamespace, "greenhouseControllerManagerNamespace", "greenhouse", "Namespace for deploying Greenhouse manager")
	flag.StringVar(&greenhouseControllerManagerRelease, "greenhouseControllerManagerRelease", "greenhouse", "Helm release name for deploying Greenhouse manager")

	flag.Parse()

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
		l.Error(err, "cluster creation")
		os.Exit(1)
	}
	l.Info("cluster created successfully")

	// Export kubeconfig
	f, err := os.Create(kubeconfigName)
	if err != nil {
		l.Error(err, "kubeconfig creation")
		os.Exit(1)
	}
	args := []string{"export", "kubeconfig", "--name", kindClusterName, "--kubeconfig", f.Name()}
	cmd := exec.Command("kind", args...)
	_, err = cmd.Output()
	if err != nil {
		l.Error(err, "kubeconfig export")
		os.Exit(1)
	}
	f, err = os.Create(kubeconfigInternalName)
	if err != nil {
		l.Error(err, "kubeconfig creation")
		os.Exit(1)
	}
	args = []string{"export", "kubeconfig", "--name", kindClusterName, "--internal", "--kubeconfig", f.Name()}
	cmd = exec.Command("kind", args...)
	_, err = cmd.Output()
	if err != nil {
		l.Error(err, "kubeconfig export")
		os.Exit(1)
	}

	l.Info("kubeconfig exported successfully")

	// Build image
	image := fmt.Sprintf("%s:%s", dockerImageRepository, dockerImageTag)
	if !dockerImageBuildSkip {
		err = dockerImageBuild("./../../../", image)
		if err != nil {
			l.Error(err, "image build")
			os.Exit(1)
		}
		l.Info("Docker image built successfully")
	}

	// Load image
	err = cluster.LoadImage(ctx, image)
	if err != nil {
		l.Error(err, "image load")
		os.Exit(1)
	}
	l.Info("Docker image loaded to the cluster successfully")

	// Deploy Greenhouse manager
	err = installChart(ctx, "./../../../charts/manager", greenhouseControllerManagerRelease, kubeconfig, greenhouseControllerManagerNamespace)
	if err != nil {
		l.Error(err, "deploy greenhouse")
		os.Exit(1)
	}
	l.Info("Greenhouse manager is deployed successfully")

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

	// These values are the minimal set of required values to deploy manager helm chart
	globalDNS := map[string]interface{}{"dnsDomain": "greenhouse.cloudoperators"}
	chart.Values["global"] = globalDNS

	controllerManagerImage := map[string]interface{}{"repository": dockerImageRepository, "tag": dockerImageTag}
	controllerManager, ok := chart.Values["controllerManager"].(map[string]interface{})
	if !ok {
		l.Error(err, "value merge")
		os.Exit(1)
	}
	controllerManager["image"] = controllerManagerImage
	controllerManager["replicas"] = "1"
	chart.Values["controllerManager"] = controllerManager
	// These values are the minimal set of required values to deploy manager helm chart

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
				l.Error(err, "chart install")
				return err
			}
		}
	} else {
		client := action.NewUpgrade(actionConfig)
		client.Namespace = namespace
		client.Wait = true
		client.Timeout = TEST_TIMEOUT
		if _, err := client.Run(release, chart, chart.Values); err != nil {
			l.Error(err, "chart upgrade")
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
