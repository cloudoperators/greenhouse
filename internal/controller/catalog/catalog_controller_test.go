// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cl "sigs.k8s.io/controller-runtime/pkg/client"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/test"
)

func mockKustomizationReady(kustomization *kustomizev1.Kustomization) error {
	GinkgoHelper()
	newKustomization := &kustomizev1.Kustomization{}
	Expect(test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(kustomization), kustomization)).To(Succeed(), "there should be no error getting the Kustomization")
	*newKustomization = *kustomization
	kustomizationReadyCondition := metav1.Condition{
		Type:               string(greenhousemetav1alpha1.ReadyCondition),
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "Succeeded",
		Message:            "",
	}
	newKustomization.Status.Conditions = []metav1.Condition{kustomizationReadyCondition}
	return patchStatus(kustomization, newKustomization)
}

func mockGitRepositoryReady(gitRepository *sourcev1.GitRepository) error {
	GinkgoHelper()
	newGitRepository := &sourcev1.GitRepository{}
	Expect(test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(gitRepository), gitRepository)).To(Succeed(), "there should be no error getting the GitRepository")
	*newGitRepository = *gitRepository
	gitRepositoryReadyCondition := metav1.Condition{
		Type:               string(greenhousemetav1alpha1.ReadyCondition),
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "Succeeded",
		Message:            "",
	}
	newGitRepository.Status.Conditions = []metav1.Condition{gitRepositoryReadyCondition}
	return patchStatus(gitRepository, newGitRepository)
}

func patchStatus(oldObj, newObj cl.Object) error {
	GinkgoHelper()
	return test.K8sClient.Status().Patch(test.Ctx, newObj, cl.MergeFrom(oldObj))
}

var _ = Describe("PluginDefinitionCatalog controller", Ordered, func() {
	const (
		catalogName = "greenhouse-extensions-catalog"
		catalogTest = "catalog-controller-test"
	)
	var (
		setup   *test.TestSetup
		catalog *greenhousev1alpha1.PluginDefinitionCatalog
	)
	BeforeAll(func() {
		setup = test.NewTestSetup(test.Ctx, test.K8sClient, catalogTest)
		catalog = setup.CreatePluginDefinitionCatalog(
			test.Ctx,
			catalogName,
			test.WithRepositoryURL("https://github.com/cloudoperators/greenhouse-extensions"),
			test.WithRepositoryBranch("main"),
		)
	})
	Context("When creating or updating a Plugin Definition Catalog", Ordered, func() {
		When("the catalog is created", Ordered, func() {
			It("should successfully create a flux git repository from plugin definition catalog", func() {
				By("checking if the catalog repository is created")
				gitRepository := &sourcev1.GitRepository{}
				gitRepository.SetName(catalog.Name + gitCatalogSourceSuffix)
				gitRepository.SetNamespace(catalog.Namespace)
				Eventually(func(g Gomega) {
					g.Expect(test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(catalog), catalog)).To(Succeed(), "there should be no error getting the PluginDefinitionCatalog")
					gitSource := catalog.GetCatalogSource()
					g.Expect(test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(gitRepository), gitRepository)).To(Succeed(), "there should be no error getting the GitRepository")
					g.Expect(gitRepository.Spec.URL).To(Equal(gitSource.URL), "Flux git repository URL should match the catalog source URL")
					g.Expect(gitRepository.Spec.Reference.Branch).To(Equal(*gitSource.Ref.Branch), "Flux git repository branch should match the catalog source branch")
					g.Expect(gitRepository.Spec.Interval).To(Equal(catalog.Interval()), "Flux git repository interval should match the catalog interval")
					g.Expect(gitRepository.Spec.Timeout).To(Equal(catalog.Timeout()), "Git repository timeout should match the catalog timeout")
				}).Should(Succeed(), "Flux GitRepository should be created for the PluginDefinitionCatalog")
				By("checking if the catalog kustomization is not created yet")
				kustomization := &kustomizev1.Kustomization{}
				kustomization.SetName(catalog.Name + kustomizeCatalogSuffix)
				kustomization.SetNamespace(catalog.Namespace)
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(kustomization), kustomization)
				Expect(err).To(HaveOccurred(), "there should be an error getting the Kustomization")
				Expect(errors.IsNotFound(err)).To(BeTrue(), "there should be a not found error when getting the Flux Kustomization resource")
			})

			It("should update the catalog repository status when flux git repository is ready", func() {
				By("mocking a flux git repository ready condition")
				gitRepository := &sourcev1.GitRepository{}
				gitRepository.SetName(catalog.Name + gitCatalogSourceSuffix)
				gitRepository.SetNamespace(catalog.Namespace)
				Expect(mockGitRepositoryReady(gitRepository)).To(Succeed(), "there should be no error mocking the GitRepository ready condition")
				Eventually(func(g Gomega) {
					g.Expect(test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(catalog), catalog)).To(Succeed(), "there should be no error getting the PluginDefinitionCatalog")
					gitReady := catalog.Status.GetConditionByType(greenhousev1alpha1.GitRepositoryReady)
					g.Expect(gitReady).ToNot(BeNil(), "the PluginDefinitionCatalog should have a Git Repository Ready condition")
					g.Expect(gitReady.Status).To(Equal(metav1.ConditionTrue), "the Ready condition status should be True")
					catalogReady := catalog.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
					g.Expect(catalogReady).ToNot(BeNil(), "the PluginDefinitionCatalog should have a Ready condition")
					g.Expect(catalogReady.Status).To(Equal(metav1.ConditionFalse), "the Ready condition status should be True")
					g.Expect(catalogReady.Reason).To(Equal(greenhousev1alpha1.CatalogNotReadyReason))
				}).Should(Succeed(), "catalog repository status should be ready when flux git repository is ready")
			})

			It("should successfully create a flux kustomization from plugin definition catalog", func() {
				kustomization := &kustomizev1.Kustomization{}
				kustomization.SetName(catalog.Name + kustomizeCatalogSuffix)
				kustomization.SetNamespace(catalog.Namespace)
				Eventually(func(g Gomega) {
					g.Expect(test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(kustomization), kustomization)).To(Succeed(), "there should be no error getting the Kustomization")
					sourceRef := kustomization.Spec.SourceRef
					g.Expect(sourceRef.Kind).To(Equal(sourcev1.GitRepositoryKind), "Flux Kustomization SourceRef kind should be flux git repository kind")
					g.Expect(sourceRef.Name).To(Equal(catalog.Name+gitCatalogSourceSuffix), "Flux Kustomization SourceRef name should be the flux git repository name")
					g.Expect(kustomization.Spec.Interval).To(Equal(catalog.Interval()), "Flux Kustomization interval should match the catalog interval")
					g.Expect(kustomization.Spec.Timeout).To(Equal(catalog.Timeout()), "Flux Kustomization timeout should match the catalog timeout")
					g.Expect(kustomization.Spec.Suspend).To(Equal(catalog.IsSuspended()), "Flux Kustomization suspend should match the catalog suspend status")
					g.Expect(kustomization.Spec.Path).To(Equal(catalog.ResourcePath()), "Flux Kustomization path should match the catalog source path")
					g.Expect(kustomization.Spec.Prune).To(BeFalse(), "Flux Kustomization should have prune to be false")
					g.Expect(kustomization.Spec.Force).To(BeFalse(), "Flux Kustomization should have force to be false")
				}).Should(Succeed(), "Flux Kustomization should be created for the PluginDefinitionCatalog")
			})

			It("should update the catalog kustomization status when flux kustomization is ready", func() {
				By("mocking a flux kustomization ready condition")
				kustomization := &kustomizev1.Kustomization{}
				kustomization.SetName(catalog.Name + kustomizeCatalogSuffix)
				kustomization.SetNamespace(catalog.Namespace)
				Expect(mockKustomizationReady(kustomization)).To(Succeed(), "there should be no error mocking the Kustomization ready condition")
				Eventually(func(g Gomega) {
					g.Expect(test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(catalog), catalog)).To(Succeed(), "there should be no error getting the PluginDefinitionCatalog")
					kustomizeReady := catalog.Status.GetConditionByType(greenhousev1alpha1.KustomizationReady)
					g.Expect(kustomizeReady).ToNot(BeNil(), "the PluginDefinitionCatalog should have a Kustomization Ready condition")
					g.Expect(kustomizeReady.Status).To(Equal(metav1.ConditionTrue), "the Ready condition status should be True")
					catalogReady := catalog.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
					g.Expect(catalogReady).ToNot(BeNil(), "the PluginDefinitionCatalog should have a Ready condition")
					g.Expect(catalogReady.Status).To(Equal(metav1.ConditionTrue), "the Ready condition status should be True")
					g.Expect(catalogReady.Reason).To(Equal(greenhousev1alpha1.CatalogReadyReason))
				}).Should(Succeed(), "catalog status should be ready when flux kustomization is ready")
			})
		})

		When("the catalog is suspended", func() {
			It("should suspend flux git repository and kustomization to suspend when the catalog is suspended", func() {
				catalog = setup.UpdatePluginDefinitionCatalog(test.Ctx, catalogName, test.WithSuspend(true))
				Eventually(func(g Gomega) {
					g.Expect(test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(catalog), catalog)).To(Succeed(), "there should be no error getting the PluginDefinitionCatalog")

					gitRepository := &sourcev1.GitRepository{}
					gitRepository.SetName(catalog.Name + gitCatalogSourceSuffix)
					gitRepository.SetNamespace(catalog.Namespace)
					g.Expect(test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(gitRepository), gitRepository)).To(Succeed(), "there should be no error getting the GitRepository")
					g.Expect(gitRepository.Spec.Suspend).To(BeTrue(), "Flux GitRepository should be suspended when the catalog is suspended")
					gitReady := catalog.Status.GetConditionByType(greenhousev1alpha1.GitRepositoryReady)
					g.Expect(gitReady.Status).To(Equal(metav1.ConditionUnknown), "the GitRepository Ready condition status should be Unknown when the catalog is suspended")
					g.Expect(gitReady.Reason).To(Equal(greenhousev1alpha1.CatalogSuspendedReason), "the GitRepository Ready condition reason should be suspended")

					kustomization := &kustomizev1.Kustomization{}
					kustomization.SetName(catalog.Name + kustomizeCatalogSuffix)
					kustomization.SetNamespace(catalog.Namespace)
					g.Expect(test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(kustomization), kustomization)).To(Succeed(), "there should be no error getting the Kustomization")
					g.Expect(kustomization.Spec.Suspend).To(BeTrue(), "Flux Kustomization should be suspended when the catalog is suspended")
					kustomizeReady := catalog.Status.GetConditionByType(greenhousev1alpha1.KustomizationReady)
					g.Expect(kustomizeReady.Status).To(Equal(metav1.ConditionUnknown), "the Kustomization Ready condition status should be Unknown when the catalog is suspended")
					g.Expect(kustomizeReady.Reason).To(Equal(greenhousev1alpha1.CatalogSuspendedReason), "the Kustomization Ready condition reason should be suspended")

					catalogReady := catalog.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
					g.Expect(catalogReady.Status).To(Equal(metav1.ConditionFalse), "the Ready condition status should be False when the catalog is suspended")
					g.Expect(catalogReady.Reason).To(Equal(greenhousev1alpha1.CatalogNotReadyReason), "the Ready condition reason should be suspended")
				})
			})
		})
	})
})
