// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package expect

import (
	"context"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

func CatalogReady(ctx context.Context, c client.Client, namespace, name string) {
	Eventually(func(g Gomega) {
		catalog := &greenhousev1alpha1.Catalog{}
		err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, catalog)
		g.Expect(err).NotTo(HaveOccurred(), "there should be no error fetching the Catalog resource")
		g.Expect(catalog.Status.Conditions).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Type":   Equal(greenhousemetav1alpha1.ReadyCondition),
			"Status": Equal(metav1.ConditionTrue),
		})), "the Catalog resource should have a Ready condition set to True")
	}).Should(Succeed(), "the Catalog resource should eventually be ready")
}

func CatalogDeleted(ctx context.Context, c client.Client, namespace, name string) {
	catalog := &greenhousev1alpha1.Catalog{}
	err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, catalog)
	Expect(err).NotTo(HaveOccurred(), "there should be no error fetching the Catalog resource")
	err = c.Delete(ctx, catalog)
	Expect(err).NotTo(HaveOccurred(), "there should be no error deleting the Catalog resource")
	Eventually(func(g Gomega) {
		err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, catalog)
		g.Expect(err).To(HaveOccurred(), "there should be no error fetching the Catalog resource")
		g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "the Catalog resource should not be found")
	}).Should(Succeed(), "the Catalog resource should eventually be deleted")
}
