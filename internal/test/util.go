// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

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
)

func UpdateClusterWithDeletionAnnotation(ctx context.Context, c client.Client, id client.ObjectKey) *greenhousev1alpha1.Cluster {
	GinkgoHelper()
	schedule, err := clientutil.ParseDateTime(time.Now().Add(-1 * time.Minute))
	Expect(err).ToNot(HaveOccurred(), "there should be no error parsing the time")
	cluster := &greenhousev1alpha1.Cluster{}
	Eventually(func(g Gomega) {
		g.Expect(c.Get(ctx, id, cluster)).
			To(Succeed(), "there must be no error getting the cluster")
		baseCluster := cluster.DeepCopy()
		cluster.SetAnnotations(map[string]string{
			greenhouseapis.MarkClusterDeletionAnnotation:     "true",
			greenhouseapis.ScheduleClusterDeletionAnnotation: schedule.Format(time.DateTime),
		})
		g.Expect(c.Patch(ctx, cluster, client.MergeFrom(baseCluster))).To(Succeed(), "there must be no error updating the cluster")
	}).Should(Succeed(), "there should be no error setting the cluster deletion annotation")
	return cluster
}

func RemoveDeletionProtection(ctx context.Context, c client.Client, id client.ObjectKey) *greenhousev1alpha1.PluginPreset {
	GinkgoHelper()
	pluginPreset := &greenhousev1alpha1.PluginPreset{}
	Eventually(func(g Gomega) {
		g.Expect(c.Get(ctx, id, pluginPreset)).
			To(Succeed(), "there must be no error getting the plugin preset")
		base := pluginPreset.DeepCopy()
		annotations := pluginPreset.GetAnnotations()
		delete(annotations, greenhousev1alpha1.PreventDeletionAnnotation)
		pluginPreset.SetAnnotations(annotations)
		g.Expect(c.Patch(ctx, pluginPreset, client.MergeFrom(base))).To(Succeed(), "there must be no error updating the pluginpreset")
	}).Should(Succeed(), "there should be no error removing the deletion projection")
	return pluginPreset
}

// MustDeleteCluster is used in the test context only and removes a cluster by namespaced name.
func MustDeleteCluster(ctx context.Context, c client.Client, id client.ObjectKey) {
	GinkgoHelper()

	cluster := UpdateClusterWithDeletionAnnotation(ctx, c, id)
	Expect(c.Delete(ctx, cluster)).
		To(Succeed(), "there must be no error deleting object", "key", client.ObjectKeyFromObject(cluster))

	Eventually(func() bool {
		err := c.Get(ctx, client.ObjectKeyFromObject(cluster), cluster)
		return apierrors.IsNotFound(err)
	}).Should(BeTrue(), "the object should be deleted eventually")
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
