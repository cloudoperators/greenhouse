// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package scenarios

import (
	"context"
	"os"
	"slices"
	"strings"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	sourcev2 "github.com/fluxcd/source-watcher/api/v2/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/e2e/shared"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

type IScenario interface {
	ExecuteSuccessScenario(ctx context.Context, namespace string)
	ExecuteCPDFailScenario(ctx context.Context, namespace string)
	ExecuteArtifactFailScenario(ctx context.Context, namespace string)
	ExecuteGitAuthFailScenario(ctx context.Context, namespace string)
	ExecuteOptionsOverrideScenario(ctx context.Context, namespace string)
}

type scenario struct {
	k8sClient  client.Client
	catalog    *greenhousev1alpha1.Catalog
	secretName string
}

func NewScenario(adminClient client.Client, catalogYamlPath, secretName string, skipTestData bool) IScenario {
	GinkgoHelper()
	catalog := &greenhousev1alpha1.Catalog{}
	if !skipTestData {
		catalogBytes, err := os.ReadFile(catalogYamlPath)
		Expect(err).ToNot(HaveOccurred(), "there should be no error reading the catalog yaml file for branch scenario")
		err = shared.FromYamlToK8sObject(string(catalogBytes), catalog)
		Expect(err).ToNot(HaveOccurred(), "there should be no error converting catalog yaml to k8s object for branch scenario")
	}
	return &scenario{
		k8sClient:  adminClient,
		catalog:    catalog,
		secretName: secretName,
	}
}

func (s *scenario) createCatalogIfNotExists(ctx context.Context) (err error) {
	GinkgoHelper()
	err = s.k8sClient.Create(ctx, s.catalog)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			catalog := &greenhousev1alpha1.Catalog{}
			catalog.SetName(s.catalog.Name)
			catalog.SetNamespace(s.catalog.Namespace)
			err = s.k8sClient.Get(ctx, client.ObjectKeyFromObject(catalog), catalog)
			if err != nil {
				return
			}
			catalog.Spec = s.catalog.Spec
			return s.k8sClient.Update(ctx, catalog)
		}
		return
	}
	return
}

func (s *scenario) deleteCatalog(ctx context.Context) {
	GinkgoHelper()
	GinkgoWriter.Printf("Deleting Catalog %s/%s\n", s.catalog.Namespace, s.catalog.Name)
	Expect(s.k8sClient.Delete(ctx, s.catalog)).ToNot(HaveOccurred(), "there should be no error deleting the Catalog")
	for _, source := range s.catalog.Spec.Sources {
		groupKey, err := getSourceGroupHash(source, s.catalog.Name)
		Expect(err).ToNot(HaveOccurred(), "there should be no error getting the source group hash")
		s.expectGitRepositoryNotFound(ctx, groupKey)
		s.expectGeneratorNotFound(ctx, groupKey)
		s.expectExternalArtifactNotFound(ctx, groupKey)
		s.expectKustomizationNotFound(ctx, groupKey)
	}
}

func (s *scenario) expectGitRepositoryReady(ctx context.Context, groupKey string) {
	GinkgoHelper()
	By("checking if GitRepository has Ready=True condition")
	Eventually(func(g Gomega) {
		gitRepository := s.getGitRepositoryObject(groupKey)
		g.Expect(s.k8sClient.Get(ctx, client.ObjectKeyFromObject(gitRepository), gitRepository)).To(Succeed(), "there should be no error getting the GitRepository")
		readyCond := meta.FindStatusCondition(gitRepository.GetConditions(), fluxmeta.ReadyCondition)
		g.Expect(readyCond).ToNot(BeNil(), "the GitRepository should have a Ready condition")
		g.Expect(readyCond.Status).To(Equal(metav1.ConditionTrue), "the GitRepository Ready condition status should be True - "+gitRepository.Name)
	}).Should(Succeed(), "flux GitRepository should be created for the Catalog source")
}

func (s *scenario) expectGitRepositoryNotFound(ctx context.Context, groupKey string) {
	GinkgoHelper()
	By("checking if GitRepository does not exist")
	Eventually(func(g Gomega) {
		gitRepository := s.getGitRepositoryObject(groupKey)
		err := s.k8sClient.Get(ctx, client.ObjectKeyFromObject(gitRepository), gitRepository)
		g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "the GitRepository should be not found - "+gitRepository.Name)
	}).Should(Succeed(), "flux GitRepository should not exist for the Catalog source")
}

func (s *scenario) expectGitRepositoryFailedAuth(ctx context.Context, groupKey string) {
	GinkgoHelper()
	By("checking if GitRepository has Ready=False condition due to auth failure")
	Eventually(func(g Gomega) {
		gitRepository := s.getGitRepositoryObject(groupKey)
		g.Expect(s.k8sClient.Get(ctx, client.ObjectKeyFromObject(gitRepository), gitRepository)).To(Succeed(), "there should be no error getting the GitRepository")
		readyCond := meta.FindStatusCondition(gitRepository.GetConditions(), fluxmeta.ReadyCondition)
		g.Expect(readyCond).ToNot(BeNil(), "the GitRepository should have a Ready condition")
		g.Expect(readyCond.Status).To(Equal(metav1.ConditionFalse), "the GitRepository Ready condition status should be False - "+gitRepository.Name)
		g.Expect(readyCond.Reason).To(Equal(sourcev1.GitOperationFailedReason), "the GitRepository Ready condition reason should be GitOperationFailed - "+gitRepository.Name)
		g.Expect(readyCond.Message).To(ContainSubstring("Invalid username or token"), "git repository failure message should contain 'authentication failed' error - "+gitRepository.Name)
	}).Should(Succeed(), "flux GitRepository should have Ready=False condition for the Catalog source")
}

func (s *scenario) expectGeneratorReady(ctx context.Context, groupKey string) {
	GinkgoHelper()
	By("checking if ArtifactGenerator has Ready=True condition")
	Eventually(func(g Gomega) {
		artifactGenerator := s.getGeneratorObject(groupKey)
		g.Expect(s.k8sClient.Get(ctx, client.ObjectKeyFromObject(artifactGenerator), artifactGenerator)).To(Succeed(), "there should be no error getting the ArtifactGenerator")
		readyCond := meta.FindStatusCondition(artifactGenerator.GetConditions(), fluxmeta.ReadyCondition)
		g.Expect(readyCond).ToNot(BeNil(), "the ArtifactGenerator should have a Ready condition")
		g.Expect(readyCond.Status).To(Equal(metav1.ConditionTrue), "the ArtifactGenerator Ready condition status should be True - "+artifactGenerator.Name)
	}).Should(Succeed(), "flux ArtifactGenerator should be created for the Catalog source")
}

func (s *scenario) expectGeneratorNotReady(ctx context.Context, groupKey string) {
	GinkgoHelper()
	By("checking if ArtifactGenerator has Ready=False condition")
	Eventually(func(g Gomega) {
		artifactGenerator := s.getGeneratorObject(groupKey)
		g.Expect(s.k8sClient.Get(ctx, client.ObjectKeyFromObject(artifactGenerator), artifactGenerator)).To(Succeed(), "there should be no error getting the ArtifactGenerator")
		readyCond := meta.FindStatusCondition(artifactGenerator.GetConditions(), fluxmeta.ReadyCondition)
		g.Expect(readyCond).ToNot(BeNil(), "the ArtifactGenerator should have a Ready condition")
		g.Expect(readyCond.Status).To(Equal(metav1.ConditionFalse), "the ArtifactGenerator Ready condition status should be False - "+artifactGenerator.Name)
	}).Should(Succeed(), "flux ArtifactGenerator should be created and not Ready for the Catalog source")
}

func (s *scenario) expectGeneratorNotFound(ctx context.Context, groupKey string) {
	GinkgoHelper()
	By("checking if ArtifactGenerator does not exist")
	Eventually(func(g Gomega) {
		artifactGenerator := s.getGeneratorObject(groupKey)
		err := s.k8sClient.Get(ctx, client.ObjectKeyFromObject(artifactGenerator), artifactGenerator)
		g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "the ArtifactGenerator should be not found - "+artifactGenerator.Name)
	}).Should(Succeed(), "flux ArtifactGenerator should not exist for the Catalog source")
}

func (s *scenario) expectGeneratorFailed(ctx context.Context, groupKey string) {
	GinkgoHelper()
	By("checking if ArtifactGenerator has Ready=False condition")
	Eventually(func(g Gomega) {
		artifactGenerator := s.getGeneratorObject(groupKey)
		g.Expect(s.k8sClient.Get(ctx, client.ObjectKeyFromObject(artifactGenerator), artifactGenerator)).To(Succeed(), "there should be no error getting the ArtifactGenerator")
		readyCond := meta.FindStatusCondition(artifactGenerator.GetConditions(), fluxmeta.ReadyCondition)
		g.Expect(readyCond).ToNot(BeNil(), "the ArtifactGenerator should have a Ready condition")
		g.Expect(readyCond.Status).To(Equal(metav1.ConditionFalse), "the ArtifactGenerator Ready condition status should be False - "+artifactGenerator.Name)
		g.Expect(readyCond.Reason).To(Equal(fluxmeta.BuildFailedReason), "the ArtifactGenerator Ready condition reason should be BuildFailed - "+artifactGenerator.Name)
		g.Expect(readyCond.Message).To(ContainSubstring("build failed"), "artifact generator failure message should contain 'build failed' error - "+artifactGenerator.Name)
	}).Should(Succeed(), "flux ArtifactGenerator should have Ready=False condition for the Catalog source")
}

func (s *scenario) expectExternalArtifactReady(ctx context.Context, groupKey string) {
	GinkgoHelper()
	By("checking if ExternalArtifact has Ready=True condition")
	Eventually(func(g Gomega) {
		externalArtifact := s.getExternalArtifactObject(groupKey)
		g.Expect(s.k8sClient.Get(ctx, client.ObjectKeyFromObject(externalArtifact), externalArtifact)).To(Succeed(), "there should be no error getting the ExternalArtifact")
		readyCond := meta.FindStatusCondition(externalArtifact.GetConditions(), fluxmeta.ReadyCondition)
		g.Expect(readyCond).ToNot(BeNil(), "the ExternalArtifact should have a Ready condition")
		g.Expect(readyCond.Status).To(Equal(metav1.ConditionTrue), "the ExternalArtifact Ready condition status should be True - "+externalArtifact.Name)
	}).Should(Succeed(), "flux ExternalArtifact should be created for the Catalog source")
}

func (s *scenario) expectExternalArtifactNotFound(ctx context.Context, groupKey string) {
	GinkgoHelper()
	By("checking if ExternalArtifact does not exist")
	Eventually(func(g Gomega) {
		externalArtifact := s.getExternalArtifactObject(groupKey)
		err := s.k8sClient.Get(ctx, client.ObjectKeyFromObject(externalArtifact), externalArtifact)
		g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "the ExternalArtifact should be not found - "+externalArtifact.Name)
	}).Should(Succeed(), "flux ExternalArtifact should not exist for the Catalog source")
}

func (s *scenario) expectKustomizationReady(ctx context.Context, groupKey string, overrides []greenhousev1alpha1.CatalogOverrides) {
	GinkgoHelper()
	By("checking if Kustomization has Ready=True condition")
	kustomization := s.getKustomizationObject(groupKey)
	var inventory *kustomizev1.ResourceInventory
	Eventually(func(g Gomega) {
		g.Expect(s.k8sClient.Get(ctx, client.ObjectKeyFromObject(kustomization), kustomization)).To(Succeed(), "there should be no error getting the Kustomization")
		readyCond := meta.FindStatusCondition(kustomization.GetConditions(), fluxmeta.ReadyCondition)
		g.Expect(readyCond).ToNot(BeNil(), "the Kustomization should have a Ready condition")
		g.Expect(readyCond.Status).To(Equal(metav1.ConditionTrue), "the Kustomization Ready condition status should be True - "+kustomization.Name)
		inventory = kustomization.Status.Inventory
		g.Expect(inventory).ToNot(BeNil(), "the Kustomization status inventory should not be nil")
	}).Should(Succeed(), "flux Kustomization should be created for the Catalog source")
	s.verifyPluginDefinitions(ctx, inventory, overrides)
}

func (s *scenario) expectKustomizationNotFound(ctx context.Context, groupKey string) {
	GinkgoHelper()
	By("checking if Kustomization does not exist")
	Eventually(func(g Gomega) {
		kustomization := s.getKustomizationObject(groupKey)
		err := s.k8sClient.Get(ctx, client.ObjectKeyFromObject(kustomization), kustomization)
		g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "the Kustomization should be not found - "+kustomization.Name)
	}).Should(Succeed(), "flux Kustomization should not exist for the Catalog source")
}

func (s *scenario) expectKustomizationFailed(ctx context.Context, groupKey string) {
	GinkgoHelper()
	By("checking if Kustomization has Ready=False condition")
	Eventually(func(g Gomega) {
		kustomization := s.getKustomizationObject(groupKey)
		g.Expect(s.k8sClient.Get(ctx, client.ObjectKeyFromObject(kustomization), kustomization)).To(Succeed(), "there should be no error getting the Kustomization")
		readyCond := meta.FindStatusCondition(kustomization.GetConditions(), fluxmeta.ReadyCondition)
		g.Expect(readyCond).ToNot(BeNil(), "the Kustomization should have a Ready condition")
		g.Expect(readyCond.Status).To(Equal(metav1.ConditionFalse), "the Kustomization Ready condition status should be False - "+kustomization.Name)
		g.Expect(readyCond.Reason).To(Equal(fluxmeta.ReconciliationFailedReason), "the Kustomization Ready condition reason should be ReconciliationFailed - "+kustomization.Name)
		g.Expect(readyCond.Message).To(ContainSubstring("forbidden"), "kustomization failure message should contain 'forbidden' error - "+kustomization.Name)
	}).Should(Succeed(), "flux Kustomization should have Ready=False condition for the Catalog source")
}

func (s *scenario) getGitRepositoryObject(groupKey string) *sourcev1.GitRepository {
	inventory := s.catalog.Status.Inventory[groupKey]
	idx := getKindIndexFromInventory(inventory, sourcev1.GitRepositoryKind)
	if idx == -1 {
		return nil
	}
	gitRepo := &sourcev1.GitRepository{}
	gitRepo.SetName(inventory[idx].Name)
	gitRepo.SetNamespace(s.catalog.Namespace)
	return gitRepo
}

func (s *scenario) getGeneratorObject(groupKey string) *sourcev2.ArtifactGenerator {
	inventory := s.catalog.Status.Inventory[groupKey]
	idx := getKindIndexFromInventory(inventory, sourcev2.ArtifactGeneratorKind)
	if idx == -1 {
		return nil
	}
	generator := &sourcev2.ArtifactGenerator{}
	generator.SetName(inventory[idx].Name)
	generator.SetNamespace(s.catalog.Namespace)
	return generator
}

func (s *scenario) getExternalArtifactObject(groupKey string) *sourcev1.ExternalArtifact {
	inventory := s.catalog.Status.Inventory[groupKey]
	idx := getKindIndexFromInventory(inventory, sourcev1.ExternalArtifactKind)
	if idx == -1 {
		return nil
	}
	artifact := &sourcev1.ExternalArtifact{}
	artifact.SetName(inventory[idx].Name)
	artifact.SetNamespace(s.catalog.Namespace)
	return artifact
}

func (s *scenario) getKustomizationObject(groupKey string) *kustomizev1.Kustomization {
	inventory := s.catalog.Status.Inventory[groupKey]
	idx := getKindIndexFromInventory(inventory, kustomizev1.KustomizationKind)
	if idx == -1 {
		return nil
	}
	kustomization := &kustomizev1.Kustomization{}
	kustomization.SetName(inventory[idx].Name)
	kustomization.SetNamespace(s.catalog.Namespace)
	return kustomization
}

func getKindIndexFromInventory(inventory []greenhousev1alpha1.SourceStatus, kind string) int {
	return slices.IndexFunc(inventory, func(item greenhousev1alpha1.SourceStatus) bool {
		return item.Kind == kind
	})
}

func (s *scenario) verifyPluginDefinitions(ctx context.Context, kuzInventory *kustomizev1.ResourceInventory, overrides []greenhousev1alpha1.CatalogOverrides) {
	GinkgoHelper()
	By("checking if PluginDefinitions are created")
	for _, resource := range kuzInventory.Entries {
		resourceMeta := strings.Split(resource.ID, "_")
		if len(resourceMeta) == 4 && resourceMeta[0] != "" && resourceMeta[0] == s.catalog.Namespace {
			key := types.NamespacedName{
				Namespace: resourceMeta[0],
				Name:      resourceMeta[1],
			}
			pluginDef := checkIfPDExists(ctx, s.k8sClient, key)
			if len(overrides) > 0 {
				checkIfRepositoryIsOverridden(pluginDef.Spec, overrides, pluginDef.Name)
			}
		} else {
			key := types.NamespacedName{
				Name: resourceMeta[1],
			}
			clusterPluginDef := checkIfCPDExists(ctx, s.k8sClient, key)
			checkIfRepositoryIsOverridden(clusterPluginDef.Spec, overrides, clusterPluginDef.Name)
		}
	}
}

func checkIfPDExists(ctx context.Context, k8sClient client.Client, namespacedName types.NamespacedName) *greenhousev1alpha1.PluginDefinition {
	GinkgoHelper()
	pluginDef := &greenhousev1alpha1.PluginDefinition{}
	err := k8sClient.Get(ctx, namespacedName, pluginDef)
	Expect(err).ToNot(HaveOccurred(), "there should be no error getting the PluginDefinition")
	return pluginDef
}

func checkIfCPDExists(ctx context.Context, k8sClient client.Client, namespacedName types.NamespacedName) *greenhousev1alpha1.ClusterPluginDefinition {
	GinkgoHelper()
	clusterPluginDef := &greenhousev1alpha1.ClusterPluginDefinition{}
	err := k8sClient.Get(ctx, namespacedName, clusterPluginDef)
	Expect(err).ToNot(HaveOccurred(), "there should be no error getting the ClusterPluginDefinition")
	return clusterPluginDef
}

func checkIfRepositoryIsOverridden(spec greenhousev1alpha1.PluginDefinitionSpec, overrides []greenhousev1alpha1.CatalogOverrides, pdName string) {
	GinkgoHelper()
	repoOverrideIdx := slices.IndexFunc(overrides, func(override greenhousev1alpha1.CatalogOverrides) bool {
		return override.Alias == pdName && override.Repository != ""
	})
	if repoOverrideIdx != -1 {
		By("checking if PluginDefinition repository is overridden by Catalog override for " + pdName)
		overriddenRepository := overrides[repoOverrideIdx].Repository
		Expect(overriddenRepository).To(Equal(spec.HelmChart.Repository), "the PluginDefinition repository should be overridden by the Catalog override")
	}
}

func (s *scenario) expectStatusPropagationInCatalogInventory(ctx context.Context, groupKey string, ignoreNotFound bool) {
	GinkgoHelper()
	catalog := &greenhousev1alpha1.Catalog{}
	Expect(s.k8sClient.Get(ctx, client.ObjectKeyFromObject(s.catalog), catalog)).ToNot(HaveOccurred(), "there should be no error getting the Catalog")
	groupInventory := catalog.Status.Inventory[groupKey]
	Expect(groupInventory).ToNot(BeEmpty(), "the Catalog status inventory for the source should not be empty")
	for _, resource := range groupInventory {
		var fluxObj lifecycle.CatalogObject
		switch resource.Kind {
		case sourcev1.GitRepositoryKind:
			fluxObj = s.getGitRepositoryObject(groupKey)
		case sourcev2.ArtifactGeneratorKind:
			fluxObj = s.getGeneratorObject(groupKey)
		case sourcev1.ExternalArtifactKind:
			fluxObj = s.getExternalArtifactObject(groupKey)
		case kustomizev1.KustomizationKind:
			fluxObj = s.getKustomizationObject(groupKey)
		default:
			Fail("unsupported kind for propagation check: " + resource.Kind)
		}
		By("checking if Catalog inventory contains the flux resource status for " + resource.Kind + "/" + resource.Name)
		Eventually(func(g Gomega) {
			err := s.k8sClient.Get(ctx, client.ObjectKeyFromObject(fluxObj), fluxObj)
			if apierrors.IsNotFound(err) && ignoreNotFound {
				// ignore not found errors
				return
			}
			g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the catalog flux resource: "+fluxObj.GetName())
			fluxCondition := meta.FindStatusCondition(fluxObj.GetConditions(), fluxmeta.ReadyCondition)
			g.Expect(fluxCondition).ToNot(BeNil(), "the underlying resource should have a Ready condition: "+fluxObj.GetName())

			freshCatalog := &greenhousev1alpha1.Catalog{}
			g.Expect(s.k8sClient.Get(ctx, client.ObjectKeyFromObject(s.catalog), freshCatalog)).ToNot(HaveOccurred(), "there should be no error getting the Catalog")
			freshGroupInventory := freshCatalog.Status.Inventory[groupKey]
			idx := getKindIndexFromInventory(freshGroupInventory, resource.Kind)
			g.Expect(idx).ToNot(Equal(-1), "the Catalog inventory should contain the resource kind: "+resource.Kind)
			kindInventoryStatus := freshGroupInventory[idx].Ready

			g.Expect(kindInventoryStatus).To(Equal(fluxCondition.Status), "the Catalog inventory status should contain the flux resource condition status")
		}).Should(Succeed(), "the flux resource condition should be propagated to the Catalog inventory status")
	}
}
