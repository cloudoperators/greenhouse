// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package expect

import (
	"context"

	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

const (
	catalogE2ELabel = "greenhouse.sap/managed-by"
)

func AllCatalogDeleted(ctx context.Context, c client.Client) {
	catalogs := &greenhousev1alpha1.CatalogList{}
	err := c.List(ctx, catalogs, client.MatchingLabels{catalogE2ELabel: "e2e"})
	Expect(err).NotTo(HaveOccurred(), "there should be no error listing the Catalog resources")
	for _, catalog := range catalogs.Items {
		err = c.Delete(ctx, &catalog)
		Expect(err).NotTo(HaveOccurred(), "there should be no error deleting the Catalog resource")
		Eventually(func(g Gomega) {
			err := c.Get(ctx, client.ObjectKeyFromObject(&catalog), &catalog)
			g.Expect(err).To(HaveOccurred(), "there should be no error fetching the Catalog resource")
			g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "the Catalog resource should not be found")
		}).Should(Succeed(), "the Catalog resource should eventually be deleted")
	}
}
