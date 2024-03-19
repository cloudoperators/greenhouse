// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"encoding/json"
	"time"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	gomegaTypes "github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
)

// LabelRemoveAllFinalizersOnDeletion is used in the test context only to indicate a resource must be deleted even with finalizers present.
const LabelRemoveAllFinalizersOnDeletion = "greenhouse.test/removeAllFinalizersOnDeletion"

// MustDeleteCluster is used in the test context only and removes all clusters.
func MustDeleteCluster(ctx context.Context, c client.Client, id client.ObjectKey) {
	var cluster = new(greenhousev1alpha1.Cluster)
	Expect(c.Get(ctx, id, cluster)).
		To(Succeed(), "there must be no error getting the cluster")
	Expect(c.Delete(ctx, cluster)).
		To(Succeed(), "there must be no error deleting cluster", "key", client.ObjectKeyFromObject(cluster))
	if isHasLabel(cluster, LabelRemoveAllFinalizersOnDeletion) {
		Expect(removeAllFinalizersFromObject(ctx, c, cluster)).
			To(Succeed(), "there must be no error removing all finalizers from the cluster",
				"key", client.ObjectKeyFromObject(cluster))
	}

	Eventually(func() bool {
		err := c.Get(ctx, id, cluster)
		return apierrors.IsNotFound(err)
	}, time.Minute, time.Second).
		Should(BeFalse(), "the cluster should be deleted eventually")
}

// MustDeleteSecretWithLabel is used in the test context only and removes all secrets.
func MustDeleteSecretWithLabel(ctx context.Context, c client.Client, l string) {
	var secretList = new(corev1.SecretList)
	listOpts := &client.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set{"greenhouse/test": l})}
	Expect(c.List(ctx, secretList, listOpts)).
		To(Succeed(), "there must be no error listing all secrets matching the label")
	for _, secret := range secretList.Items {
		cp := secret.DeepCopy()
		Expect(c.Delete(ctx, cp)).
			To(Succeed(), "there must be no error deleting secret", "key", client.ObjectKeyFromObject(cp))
	}
	Eventually(func() []corev1.Secret {
		Expect(c.List(ctx, secretList, listOpts)).
			To(Succeed(), "there must be no error listing all secrets matching the label")
		return secretList.Items
	}, time.Minute, time.Second).
		Should(BeEmpty(), "there should be no secret left")
}

func isHasLabel(o client.Object, labelKey string) bool {
	lbls := o.GetLabels()
	if lbls == nil {
		return false
	}
	_, ok := lbls[labelKey]
	return ok
}

func removeAllFinalizersFromObject(ctx context.Context, c client.Client, o client.Object) error {
	_, err := clientutil.Patch(ctx, c, o, func() error {
		o.SetFinalizers(nil)
		return nil
	})
	return err
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
