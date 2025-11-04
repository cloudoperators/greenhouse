// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package scenarios

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

// ExecuteCPDFailScenario executes the ClusterPluginDefinition scenario - Fail Path
func (s *scenario) ExecuteCPDFailScenario(ctx context.Context, namespace string) {
	GinkgoHelper()
	s.catalog.SetNamespace(namespace)
	err := s.createCatalogIfNotExists(ctx)
	Expect(err).ToNot(HaveOccurred(), "there should be no error creating the Catalog for cluster scenario")
	for _, source := range s.catalog.Spec.Sources {
		groupKey, err := getSourceGroupHash(source, s.catalog.Name)
		Expect(err).ToNot(HaveOccurred(), "there should be no error getting the source group hash")
		Eventually(func(g Gomega) {
			g.Expect(s.k8sClient.Get(ctx, client.ObjectKeyFromObject(s.catalog), s.catalog)).ToNot(HaveOccurred(), "there should be no error getting the Catalog")
			inventory := s.catalog.Status.Inventory
			g.Expect(inventory).ToNot(BeNil(), "the Catalog status inventory should not be nil")
			g.Expect(inventory[groupKey]).To(HaveLen(4), "the Catalog status inventory for the source should have 4 entries")
		}).Should(Succeed(), "the Catalog status inventory should be populated for the source")
		s.expectGitRepositoryReady(ctx, groupKey)
		s.expectGeneratorReady(ctx, groupKey)
		s.expectExternalArtifactReady(ctx, groupKey)
		s.expectKustomizationFailed(ctx, groupKey)
	}
	By("checking if Catalog has Ready=False condition")
	Eventually(func(g Gomega) {
		catalog := &greenhousev1alpha1.Catalog{}
		g.Expect(s.k8sClient.Get(ctx, client.ObjectKeyFromObject(s.catalog), catalog)).ToNot(HaveOccurred(), "there should be no error getting the Catalog")
		inventory := catalog.Status.Inventory
		g.Expect(inventory).ToNot(BeNil(), "the Catalog status inventory should not be nil")
		g.Expect(inventory).To(HaveLen(len(catalog.Spec.Sources)), "number of Catalog status inventory map should be equal to catalog sources length")
		for groupKey, items := range inventory {
			g.Expect(items).To(HaveLen(4), "each Catalog status inventory entry should have 4 items")
			s.expectStatusPropagationInCatalogInventory(ctx, groupKey, false)
		}
		catalogReady := catalog.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
		g.Expect(catalogReady).ToNot(BeNil(), "the Catalog should have a Ready condition")
		g.Expect(catalogReady.Status).To(Equal(metav1.ConditionFalse), "the Ready condition status should be False")
		g.Expect(catalogReady.Reason).To(Equal(greenhousev1alpha1.CatalogNotReadyReason), "the Ready condition reason should be CatalogNotReady")
	}).Should(Succeed(), "the Catalog should have a Ready=False condition")
	s.deleteCatalog(ctx)
}

func (s *scenario) ExecuteArtifactFailScenario(ctx context.Context, namespace string) {
	GinkgoHelper()
	s.catalog.SetNamespace(namespace)
	Expect(s.createCatalogIfNotExists(ctx)).To(Succeed(), "there should be no error creating the Catalog for cluster scenario")
	for _, source := range s.catalog.Spec.Sources {
		groupKey, err := getSourceGroupHash(source, s.catalog.Name)
		Expect(err).ToNot(HaveOccurred(), "there should be no error getting the source group hash")
		Eventually(func(g Gomega) {
			g.Expect(s.k8sClient.Get(ctx, client.ObjectKeyFromObject(s.catalog), s.catalog)).ToNot(HaveOccurred(), "there should be no error getting the Catalog")
			inventory := s.catalog.Status.Inventory
			g.Expect(inventory).ToNot(BeNil(), "the Catalog status inventory should not be nil")
			g.Expect(inventory[groupKey]).To(HaveLen(4), "the Catalog status inventory for the source should have 4 entries")
		}).Should(Succeed(), "the Catalog status inventory should be populated for the source")
		s.expectGitRepositoryReady(ctx, groupKey)
		s.expectGeneratorFailed(ctx, groupKey)
		s.expectExternalArtifactNotFound(ctx, groupKey)
		s.expectKustomizationNotFound(ctx, groupKey)
	}
	By("checking if Catalog has Ready=False condition")
	Eventually(func(g Gomega) {
		catalog := &greenhousev1alpha1.Catalog{}
		g.Expect(s.k8sClient.Get(ctx, client.ObjectKeyFromObject(s.catalog), catalog)).ToNot(HaveOccurred(), "there should be no error getting the Catalog")
		inventory := catalog.Status.Inventory
		g.Expect(inventory).ToNot(BeNil(), "the Catalog status inventory should not be nil")
		g.Expect(inventory).To(HaveLen(len(catalog.Spec.Sources)), "number of Catalog status inventory map should be equal to catalog sources length")
		for groupKey, items := range inventory {
			g.Expect(items).To(HaveLen(4), "each Catalog status inventory entry should have 4 items")
			s.expectStatusPropagationInCatalogInventory(ctx, groupKey, true)
		}
		catalogReady := catalog.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
		g.Expect(catalogReady).ToNot(BeNil(), "the Catalog should have a Ready condition")
		g.Expect(catalogReady.Status).To(Equal(metav1.ConditionFalse), "the Ready condition status should be False")
		g.Expect(catalogReady.Reason).To(Equal(greenhousev1alpha1.CatalogNotReadyReason), "the Ready condition reason should be CatalogNotReady")
	}).Should(Succeed(), "the Catalog should have a Ready=False condition")
	s.deleteCatalog(ctx)
}

func (s *scenario) ExecuteGitAuthFailScenario(ctx context.Context, namespace string) {
	GinkgoHelper()
	s.catalog.SetNamespace(namespace)
	err := s.createCatalogIfNotExists(ctx)
	Expect(err).ToNot(HaveOccurred(), "there should be no error creating the Catalog for cluster scenario")
	for _, source := range s.catalog.Spec.Sources {
		groupKey, err := getSourceGroupHash(source, s.catalog.Name)
		Expect(err).ToNot(HaveOccurred(), "there should be no error getting the source group hash")
		Eventually(func(g Gomega) {
			g.Expect(s.k8sClient.Get(ctx, client.ObjectKeyFromObject(s.catalog), s.catalog)).ToNot(HaveOccurred(), "there should be no error getting the Catalog")
			inventory := s.catalog.Status.Inventory
			g.Expect(inventory).ToNot(BeNil(), "the Catalog status inventory should not be nil")
			g.Expect(inventory[groupKey]).To(HaveLen(4), "the Catalog status inventory for the source should have 4 entries")
		}).Should(Succeed(), "the Catalog status inventory should be populated for the source")
		s.expectGitRepositoryFailedAuth(ctx, groupKey)
		s.expectGeneratorNotReady(ctx, groupKey)
		s.expectExternalArtifactNotFound(ctx, groupKey)
		s.expectKustomizationNotFound(ctx, groupKey)
	}
	By("checking if Catalog has Ready=False condition")
	Eventually(func(g Gomega) {
		catalog := &greenhousev1alpha1.Catalog{}
		g.Expect(s.k8sClient.Get(ctx, client.ObjectKeyFromObject(s.catalog), catalog)).ToNot(HaveOccurred(), "there should be no error getting the Catalog")
		inventory := catalog.Status.Inventory
		g.Expect(inventory).ToNot(BeNil(), "the Catalog status inventory should not be nil")
		g.Expect(inventory).To(HaveLen(len(catalog.Spec.Sources)), "number of Catalog status inventory map should be equal to catalog sources length")
		for groupKey, items := range inventory {
			g.Expect(items).To(HaveLen(4), "each Catalog status inventory entry should have 4 items")
			s.expectStatusPropagationInCatalogInventory(ctx, groupKey, true)
		}
		catalogReady := catalog.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
		g.Expect(catalogReady).ToNot(BeNil(), "the Catalog should have a Ready condition")
		g.Expect(catalogReady.Status).To(Equal(metav1.ConditionFalse), "the Ready condition status should be False")
		g.Expect(catalogReady.Reason).To(Equal(greenhousev1alpha1.CatalogNotReadyReason), "the Ready condition reason should be CatalogNotReady")
	}).Should(Succeed(), "the Catalog should have a Ready=False condition")
	s.deleteCatalog(ctx)
}
