// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/exec"
	"testing"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

var (
	remotClusterKubeconfigFilePath = flag.String("remoteClusterKubeConfig", "", "path to the kubeconfig file for the remote cluster")
)

func TestClusterBootstrap(t *testing.T) {

	feature := features.New("Cluster onboarding")
	feature.Setup(
		func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {

			if *remotClusterKubeconfigFilePath == "" {
				logrus.Info("Creating temporary file for central cluster kubeconfig (with external access)")
				f, err := os.CreateTemp("", "greenhouse-central")
				if err != nil {
					t.Fatal(err)
				}
				defer os.Remove(f.Name())

				logrus.Info("Exporting kubeconfig file for central cluster")
				args := []string{"export", "kubeconfig", "--name", centralClusterName, "--internal", "--kubeconfig", f.Name()}
				cmd := exec.Command("kind", args...)
				output, err := cmd.Output()
				if err != nil {
					logrus.Error("Error during kind export kubeconfig:", output)
					t.Fatal(err)
				}
				*remotClusterKubeconfigFilePath = f.Name()
			}
			kubeconfigFileContents, err := os.ReadFile(*remotClusterKubeconfigFilePath)
			if err != nil {
				t.Fatal(err)
			}

			logrus.Info("Creating kubeconfig secret for cluster")
			err = centralClusterK8sClient.Create(ctx,
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      centralClusterName,
						Namespace: greenhouseOrganizationName,
					},
					Type: greenhouseapis.SecretTypeKubeConfig,
					Data: map[string][]byte{
						greenhouseapis.KubeConfigKey: kubeconfigFileContents,
					},
				},
			)
			if err != nil {
				t.Fatal(err)
			}
			return ctx
		})

	feature.Assess("Cluster with ready status", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		// wait for the cluster to be ready
		if err := wait.For(
			func(context.Context) (done bool, err error) {
				cluster := &greenhousev1alpha1.Cluster{}
				err = centralClusterK8sClient.Get(ctx, types.NamespacedName{Namespace: greenhouseOrganizationName, Name: centralClusterName}, cluster)
				if apierrors.IsNotFound(err) {
					return false, nil
				} else if err != nil {
					return false, err
				}
				clusterReady := cluster.Status.GetConditionByType(greenhousev1alpha1.ReadyCondition)
				if clusterReady == nil {
					return false, nil
				}
				if clusterReady.IsTrue() {
					logrus.Info("Cluster status is ready")
					return true, nil
				}
				return false, errors.New(clusterReady.Message)
			},
			wait.WithTimeout(TEST_TIMEOUT),
			wait.WithInterval(TEST_RETRY_INTERVAL),
		); err != nil {
			t.Fatal(err)
		}
		return ctx
	},
	)

	// submit the feature to be tested
	testEnv.Test(t, feature.Feature())
}
