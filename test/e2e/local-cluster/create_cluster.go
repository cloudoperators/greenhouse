// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	"gopkg.in/yaml.v3"
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
	dockerImagePlatform   string

	dockerImageBuildSkip = false

	greenhouseControllerManagerNamespace string
	greenhouseControllerManagerRelease   string

	greenhouseControllerManagerValuesFilename string

	idProxyNamespace      string
	idProxyRelease        string
	idProxyValuesFilename string
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
	flag.StringVar(&dockerImagePlatform, "dockerImagePlatform", "linux/amd64", "Docker image platform for Greenhouse manager")
	flag.BoolVar(&dockerImageBuildSkip, "dockerImageBuildSkip", false, "Skip building the docker image for Greenhouse manager")

	flag.StringVar(&greenhouseControllerManagerNamespace, "greenhouseControllerManagerNamespace", "greenhouse", "Namespace for deploying Greenhouse manager")
	flag.StringVar(&greenhouseControllerManagerRelease, "greenhouseControllerManagerRelease", "greenhouse", "Helm release name for deploying Greenhouse manager")

	flag.StringVar(&greenhouseControllerManagerValuesFilename, "greenhouseControllerManagerValuesFile", "./manager.values.yaml", "path to the values file for greenhouse controller manager")

	flag.StringVar(&idProxyNamespace, "gidProxyNamespace", "greenhouse", "Namespace for deploying idproxy")
	flag.StringVar(&idProxyRelease, "idProxyRelease", "idproxy", "Helm release name for deploying idproxy")
	flag.StringVar(&idProxyValuesFilename, "idProxyValuesFilename", "./idproxy.values.yaml", "path to the values file for idproxy")

	flag.Parse()

	l.Info("configuration loaded", "kindClusterName", kindClusterName, "dockerImageRepository", dockerImageRepository, "dockerImageTag", dockerImageTag, "dockerImageBuildSkip", dockerImageBuildSkip, "kubeconfigName", kubeconfigName, "kubeconfigInternalName", kubeconfigInternalName)

}

func main() {

	ctx := context.Background()
	l := log.FromContext(ctx)

	// Create cluster
	l.Info("[START] Create kind cluster")
	cluster := kind.NewCluster(kindClusterName)
	cluster.SetDefaults()
	kubeconfig, err := cluster.Create(ctx)
	if err != nil {
		l.Error(err, "Failed in cluster creation")
		os.Exit(1)
	}
	l.Info("[SUCCESS] Create kind cluster")

	// Export kubeconfig
	l.Info("[START] Export kubeconfig files")
	f, err := os.Create(kubeconfigName)
	if err != nil {
		l.Error(err, "Failed in kubeconfig creation")
		os.Exit(1)
	}
	args := []string{"export", "kubeconfig", "--name", kindClusterName, "--kubeconfig", f.Name()}
	cmd := exec.Command("kind", args...)
	_, err = cmd.Output()
	if err != nil {
		l.Error(err, "Failed in kubeconfig export")
		os.Exit(1)
	}
	f, err = os.Create(kubeconfigInternalName)
	if err != nil {
		l.Error(err, "Failed in kubeconfig creation")
		os.Exit(1)
	}
	args = []string{"export", "kubeconfig", "--name", kindClusterName, "--internal", "--kubeconfig", f.Name()}
	cmd = exec.Command("kind", args...)
	_, err = cmd.Output()
	if err != nil {
		l.Error(err, "Failed in kubeconfig export")
		os.Exit(1)
	}

	l.Info("[SUCCESS] Export kubeconfig files")

	// Build image
	l.Info("[START] Docker image build")
	image := fmt.Sprintf("%s:%s", dockerImageRepository, dockerImageTag)
	if !dockerImageBuildSkip {
		cmdArgs := []string{"build", "-t", fmt.Sprintf("%s:%s", dockerImageRepository, dockerImageTag), "--platform", dockerImagePlatform, "--no-cache", "./../../../"}
		cmd := exec.Command("docker", cmdArgs...)
		stderr, err := cmd.StderrPipe()
		if err != nil {
			l.Error(err, "Failed in docker image build")
			os.Exit(1)
		}

		err = cmd.Start()
		if err != nil {
			l.Error(err, "Failed in docker image build")
			os.Exit(1)
		}

		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			m := scanner.Text()
			l.Info("[DOCKER BUILD]", m)
		}
		cmd.Wait()

		if err != nil {
			l.Error(err, "Failed in docker image build")
			os.Exit(1)
		}
		l.Info("[SUCCESS] Docker image build")
	} else {
		l.Info("[SKIP] Docker image build")
	}

	// Load image
	l.Info("[START] Docker image load")
	err = cluster.LoadImage(ctx, image)
	if err != nil {
		l.Error(err, "Failed in image load")
		os.Exit(1)
	}
	l.Info("[SUCCESS] Docker image load")

	// Deploy idproxy chart
	l.Info("[START] Deploy idproxy chart")
	err = installChart(ctx, "./../../../charts/idproxy", idProxyRelease, kubeconfig, idProxyNamespace, idProxyValuesFilename)
	if err != nil {
		l.Error(err, "Failed in deploy idproxy")
		os.Exit(1)
	}
	l.Info("[SUCCESS] Deploy idproxy chart")

	// Deploy Greenhouse manager
	l.Info("[START] Deploy greenhouse manager chart")
	err = installChart(ctx, "./../../../charts/manager", greenhouseControllerManagerRelease, kubeconfig, greenhouseControllerManagerNamespace, greenhouseControllerManagerValuesFilename)
	if err != nil {
		l.Error(err, "Failed in deploy greenhouse")
		os.Exit(1)
	}
	l.Info("[SUCCESS] Deploy greenhouse manager chart")

}

func installChart(ctx context.Context, dir, release, kubeconfig string, namespace string, valuesFilename string) error {

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

	values := map[string]interface{}{}
	valuesFile, err := os.ReadFile(valuesFilename)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(valuesFile, &values)
	if err != nil {
		return err
	}

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

			if _, err := client.RunWithContext(ctx, chart, values); err != nil {
				l.Error(err, "chart install")
				return err
			}
		}
	} else {
		client := action.NewUpgrade(actionConfig)
		client.Namespace = namespace
		client.Wait = true
		client.Timeout = TEST_TIMEOUT
		if _, err := client.RunWithContext(ctx, release, chart, values); err != nil {
			l.Error(err, "chart upgrade")
			return err
		}
	}

	return nil
}
