// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package scenarios

import (
	"context"
	"fmt"
	"strings"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	catalogcontroller "github.com/cloudoperators/greenhouse/internal/controller/catalog"
)

func (s *scenario) ExecuteSuccessScenario(ctx context.Context, namespace string) {
	GinkgoHelper()
	s.catalog.SetNamespace(namespace)
	err := s.createCatalogIfNotExists(ctx)
	Expect(err).ToNot(HaveOccurred(), "there should be no error creating the Catalog for multi-source scenario")
	s.verifySuccess(ctx)
	By("cleaning up Catalog")
	for _, source := range s.catalog.Spec.Sources {
		groupKey, err := getSourceGroupHash(source, s.catalog.Name)
		Expect(err).ToNot(HaveOccurred(), "there should be no error getting the source group hash")
		s.deletePluginDefinitions(ctx, groupKey)
	}
	s.deleteCatalog(ctx)
}

func (s *scenario) verifySuccess(ctx context.Context) {
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
		s.expectKustomizationReady(ctx, groupKey, source.Overrides)
	}

	By("checking if Catalog has Ready=True condition")
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
		g.Expect(catalogReady.Status).To(Equal(metav1.ConditionTrue), "the Ready condition status should be True")
		g.Expect(catalogReady.Reason).To(Equal(greenhousev1alpha1.CatalogReadyReason), "the Ready condition reason should be CatalogReady")
	}).Should(Succeed(), "the Catalog should have a Ready=True condition")
}

func getSourceGroupHash(source greenhousev1alpha1.CatalogSource, catalogName string) (groupKey string, err error) {
	var host, owner, repo string
	host, owner, repo, err = catalogcontroller.GetOwnerRepoInfo(source.Repository)
	if err != nil {
		return
	}
	ref := source.GetRefValue()
	hash, err := catalogcontroller.HashValue(fmt.Sprintf("%s-%s-%s-%s-%s", catalogName, host, owner, repo, ref))
	if err != nil {
		return
	}
	groupKey = fmt.Sprintf("%s-%s-%s-%s-%s", host, owner, repo, ref, hash)
	return
}

func (s *scenario) deletePluginDefinitions(ctx context.Context, groupKey string) {
	GinkgoHelper()
	kustomization := &kustomizev1.Kustomization{}
	err := s.k8sClient.Get(ctx, client.ObjectKeyFromObject(s.getKustomizationObject(groupKey)), kustomization)
	Expect(err).ToNot(HaveOccurred(), "there should be no error getting the Kustomization")
	kuzInventory := kustomization.Status.Inventory
	Expect(kuzInventory).ToNot(BeNil(), "the Kustomization status inventory should not be nil")

	By("deleting PluginDefinitions created by kustomization")
	for _, resource := range kuzInventory.Entries {
		resourceMeta := strings.Split(resource.ID, "_")
		if len(resourceMeta) == 4 && resourceMeta[0] != "" && resourceMeta[0] == s.catalog.Namespace {
			pd := &greenhousev1alpha1.PluginDefinition{}
			pd.SetName(resourceMeta[1])
			pd.SetNamespace(resourceMeta[0])
			err = s.k8sClient.Delete(ctx, pd)
			Expect(err).ToNot(HaveOccurred(), "there should be no error deleting the PluginDefinition")
		} else {
			cpd := &greenhousev1alpha1.ClusterPluginDefinition{}
			cpd.SetName(resourceMeta[1])
			err = s.k8sClient.Delete(ctx, cpd)
			Expect(err).ToNot(HaveOccurred(), "there should be no error deleting the ClusterPluginDefinition")
		}
	}
}
