// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"os"
	"time"

	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/common"
)

func UpdateClusterWithDeletionAnnotation(ctx context.Context, c client.Client, cluster *greenhousev1alpha1.Cluster) *greenhousev1alpha1.Cluster {
	GinkgoHelper()
	schedule, err := clientutil.ParseDateTime(time.Now().Add(-1 * time.Minute))
	Expect(err).ToNot(HaveOccurred(), "there should be no error parsing the time")
	MustSetAnnotations(ctx, c, cluster, map[string]string{
		greenhouseapis.MarkClusterDeletionAnnotation:     "true",
		greenhouseapis.ScheduleClusterDeletionAnnotation: schedule.Format(time.DateTime)})
	return cluster
}

func MustSetAnnotation(ctx context.Context, c client.Client, o client.Object, key, value string) {
	GinkgoHelper()
	MustSetAnnotations(ctx, c, o, map[string]string{key: value})
}

func MustSetAnnotations(ctx context.Context, c client.Client, o client.Object, annotations map[string]string) {
	GinkgoHelper()
	Eventually(func(g Gomega) {
		base := o.DeepCopyObject().(client.Object)
		g.Expect(c.Get(ctx, client.ObjectKeyFromObject(o), o)).To(Succeed(), "there must be no error getting the object")
		if o.GetAnnotations() == nil {
			o.SetAnnotations(annotations)
		} else {
			maps.Copy(o.GetAnnotations(), annotations)
		}
		g.Expect(c.Patch(ctx, o, client.MergeFrom(base))).To(Succeed(), "there must be no error updating the object")
	}).Should(Succeed(), "there should be no error setting the annotation")
}

func MustRemoveAnnotation(ctx context.Context, c client.Client, o client.Object, key string) {
	GinkgoHelper()
	Eventually(func(g Gomega) {
		g.Expect(c.Get(ctx, client.ObjectKeyFromObject(o), o)).To(Succeed(), "there must be no error getting the object")
		delete(o.GetAnnotations(), key)
		g.Expect(c.Update(ctx, o)).To(Succeed(), "there must be no error updating the object")
	}).Should(Succeed(), "there should be no error removing the annotation")
}

// MustDeleteCluster is used in the test context only and removes a cluster by namespaced name.
func MustDeleteCluster(ctx context.Context, c client.Client, cluster *greenhousev1alpha1.Cluster) {
	GinkgoHelper()
	UpdateClusterWithDeletionAnnotation(ctx, c, cluster)

	// Retry delete until the cluster is gone - handles conflicts and waits for deletion to complete
	Eventually(func() bool {
		err := c.Get(ctx, client.ObjectKeyFromObject(cluster), cluster)
		if err != nil {
			return apierrors.IsNotFound(err)
		}

		err = c.Delete(ctx, cluster)
		if err != nil && !apierrors.IsNotFound(err) {
			return false // Delete failed, will retry
		}

		// Make sure that the cluster is gone
		err = c.Get(ctx, client.ObjectKeyFromObject(cluster), cluster)
		return apierrors.IsNotFound(err)
	}).Should(BeTrue(), "the cluster should be deleted eventually", "key", client.ObjectKeyFromObject(cluster))
}

// SetClusterReadyCondition sets the ready condition of the cluster resource.
func SetClusterReadyCondition(ctx context.Context, c client.Client, cluster *greenhousev1alpha1.Cluster, readyStatus metav1.ConditionStatus) error {
	_, err := clientutil.PatchStatus(ctx, c, cluster, func() error {
		cluster.Status.SetConditions(greenhousemetav1alpha1.NewCondition(
			greenhousemetav1alpha1.ReadyCondition,
			readyStatus,
			"",
			"",
		))
		return nil
	})
	return err
}

// MustReturnJSONFor marshals val to JSON and returns an apiextensionsv1.JSON.
func MustReturnJSONFor(val any) *apiextensionsv1.JSON {
	GinkgoHelper()
	raw, err := json.Marshal(val)
	Expect(err).ShouldNot(HaveOccurred(), "there should be no error marshalling the value")
	return &apiextensionsv1.JSON{Raw: raw}
}

// GreenhouseV1Alpha1Scheme returns a new runtime.Scheme with the Greenhouse v1alpha1 scheme added.
func GreenhouseV1Alpha1Scheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(greenhousev1alpha1.AddToScheme(scheme))
	return scheme
}

// KubeconfigFromEnvVar returns the kubeconfig []byte from the path specified in the environment variable
func KubeconfigFromEnvVar(envVar string) ([]byte, error) {
	kubeconfigPath := os.Getenv(envVar)
	if kubeconfigPath == "" {
		return nil, errors.New("kubeconfig path is empty")
	}
	kubeconfig, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return kubeconfig, nil
}

// MockHelmChartReady mocks the HelmChart status for a PluginDefinition as ready.
// This is useful in tests where the Flux source controller is not running.
// The function uses Eventually to wait for the HelmChart to be created and then patches its status.
// Works with both ClusterPluginDefinition and PluginDefinition via the common.GenericPluginDefinition interface.
func MockHelmChartReady(ctx context.Context, k8sClient client.Client, pluginDefinition common.GenericPluginDefinition, helmChartNamespace string) {
	GinkgoHelper()
	Eventually(func(g Gomega) {
		helmChart := &sourcev1.HelmChart{}
		helmChart.SetName(pluginDefinition.FluxHelmChartResourceName())
		helmChart.SetNamespace(helmChartNamespace)
		err := k8sClient.Get(ctx, client.ObjectKeyFromObject(helmChart), helmChart)
		g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the HelmChart")
		newHelmChart := &sourcev1.HelmChart{}
		*newHelmChart = *helmChart
		helmChartReadyCondition := metav1.Condition{
			Type:               fluxmeta.ReadyCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             "Succeeded",
			Message:            "Helm chart is ready",
		}
		newHelmChart.Status.Conditions = []metav1.Condition{helmChartReadyCondition}
		g.Expect(k8sClient.Status().Patch(ctx, newHelmChart, client.MergeFrom(helmChart))).To(Succeed(), "there should be no error patching HelmChart status")
	}).Should(Succeed(), "HelmChart should be mocked as ready")
}
