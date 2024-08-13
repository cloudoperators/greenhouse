// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	gomegaTypes "github.com/onsi/gomega/types"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

// MustDeleteCluster is used in the test context only and removes a cluster by namespaced name.
func MustDeleteCluster(ctx context.Context, c client.Client, id client.ObjectKey) {
	GinkgoHelper()
	var cluster = new(greenhousev1alpha1.Cluster)
	Expect(c.Get(ctx, id, cluster)).
		To(Succeed(), "there must be no error getting the cluster")

	Expect(c.Delete(ctx, cluster)).
		To(Succeed(), "there must be no error deleting object", "key", client.ObjectKeyFromObject(cluster))

	Eventually(func() bool {
		err := c.Get(ctx, client.ObjectKeyFromObject(cluster), cluster)
		return apierrors.IsNotFound(err)
	}).
		Should(BeFalse(), "the object should be deleted eventually")
}

// MustReturnJSONFor marshals val to JSON and returns an apiextensionsv1.JSON.
func MustReturnJSONFor(val any) *apiextensionsv1.JSON {
	raw, err := json.Marshal(val)
	Expect(err).ShouldNot(HaveOccurred(), "there should be no error marshalling the value")
	return &apiextensionsv1.JSON{Raw: raw}
}

var ClientObjectMatcherByName = func(name string) gomegaTypes.GomegaMatcher {
	return gstruct.MatchFields(
		gstruct.IgnoreExtras, gstruct.Fields{"ObjectMeta": gstruct.MatchFields(
			gstruct.IgnoreExtras, gstruct.Fields{"Name": Equal(
				name)})})
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
		return nil, fmt.Errorf("kubeconfig path is empty")
	}
	kubeconfig, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return kubeconfig, nil
}
