// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cl "sigs.k8s.io/controller-runtime/pkg/client"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/flux"
	"github.com/cloudoperators/greenhouse/internal/rbac"
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

var _ = Describe("Catalog controller", Ordered, func() {
	const (
		catalogName = "greenhouse-extensions-catalog"
		catalogTest = "catalog-controller-test"
	)
	var (
		setup           *test.TestSetup
		catalog         *greenhousev1alpha1.Catalog
		defaultInterval = metav1.Duration{Duration: flux.DefaultInterval}
	)
	BeforeAll(func() {
		setup = test.NewTestSetup(test.Ctx, test.K8sClient, catalogTest)
		catalog = setup.CreateCatalog(
			test.Ctx,
			catalogName,
			test.WithRepositoryURL("https://github.com/cloudoperators/greenhouse-extensions"),
			test.WithRepositoryBranch("main"),
		)
	})
	Context("When creating or updating a Plugin Definition Catalog", Ordered, func() {

		It("should successfully create a flux git repository and kustomization from plugin definition catalog", func() {
			By("checking if the catalog repository is created")
			gitRepository := &sourcev1.GitRepository{}
			gitRepository.SetName(catalog.Name)
			gitRepository.SetNamespace(catalog.Namespace)
			Eventually(func(g Gomega) {
				g.Expect(test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(catalog), catalog)).To(Succeed(), "there should be no error getting the Catalog")
				gitSource := catalog.GetCatalogSource()
				g.Expect(test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(gitRepository), gitRepository)).To(Succeed(), "there should be no error getting the GitRepository")
				g.Expect(gitRepository.Spec.URL).To(Equal(gitSource.URL), "Flux git repository URL should match the catalog source URL")
				g.Expect(gitRepository.Spec.Reference.Branch).To(Equal(*gitSource.Ref.Branch), "Flux git repository branch should match the catalog source branch")
				g.Expect(gitRepository.Spec.Interval).To(Equal(defaultInterval), "Flux git repository interval should match the catalog interval")
			}).Should(Succeed(), "Flux GitRepository should be created for the Catalog")

			By("checking if the catalog kustomization is created")
			kustomization := &kustomizev1.Kustomization{}
			kustomization.SetName(catalog.Name)
			kustomization.SetNamespace(catalog.Namespace)
			expectedServiceAccountName := rbac.OrgCatalogServiceAccountName(catalog.Namespace)
			Eventually(func(g Gomega) {
				g.Expect(test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(kustomization), kustomization)).To(Succeed(), "there should be no error getting the Kustomization")
				sourceRef := kustomization.Spec.SourceRef
				g.Expect(sourceRef.Kind).To(Equal(sourcev1.GitRepositoryKind), "Flux Kustomization SourceRef kind should be flux git repository kind")
				g.Expect(sourceRef.Name).To(Equal(catalog.Name), "Flux Kustomization SourceRef name should be the flux git repository name")
				g.Expect(kustomization.Spec.Interval).To(Equal(defaultInterval), "Flux Kustomization interval should match the catalog interval")
				g.Expect(kustomization.Spec.Path).To(Equal(catalog.ResourcePath()), "Flux Kustomization path should match the catalog source path")
				g.Expect(kustomization.Spec.ServiceAccountName).To(Equal(expectedServiceAccountName), "Flux Kustomization should reference the organization's ServiceAccount")
			}).Should(Succeed(), "Flux Kustomization should be created for the Catalog")
		})

		It("should reach Ready=True for catalog status when flux git repository and kustomization is ready", func() {
			By("mocking flux git repository Ready=True condition")
			gitRepository := &sourcev1.GitRepository{}
			gitRepository.SetName(catalog.Name)
			gitRepository.SetNamespace(catalog.Namespace)
			Expect(mockGitRepositoryReady(gitRepository)).To(Succeed(), "there should be no error mocking the GitRepository ready condition")

			By("mocking flux kustomization Ready=True condition")
			kustomization := &kustomizev1.Kustomization{}
			kustomization.SetName(catalog.Name)
			kustomization.SetNamespace(catalog.Namespace)
			Expect(mockKustomizationReady(kustomization)).To(Succeed(), "there should be no error mocking the Kustomization ready condition")

			By("verifying catalog status has Ready=True condition")
			Eventually(func(g Gomega) {
				g.Expect(test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(catalog), catalog)).To(Succeed(), "there should be no error getting the Catalog")
				catalogReady := catalog.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
				g.Expect(catalogReady).ToNot(BeNil(), "the Catalog should have a Ready condition")
				g.Expect(catalogReady.Status).To(Equal(metav1.ConditionTrue), "the Ready condition status should be True")
			}).Should(Succeed(), "catalog status should be ready when flux kustomization is ready")
		})

		It("should successfully update flux kustomization patch when catalog has overrides", func() {
			catalog = setup.UpdateCatalog(test.Ctx, catalogName, test.WithOverrides([]greenhousev1alpha1.CatalogOverrides{
				{
					Alias: "new-name",
					Name:  "name",
				},
			}))
			patches, err := flux.PrepareKustomizePatches(catalog.Spec.Overrides, greenhousev1alpha1.GroupVersion.Group)
			Expect(err).NotTo(HaveOccurred(), "there should be no error preparing kustomize patches for the catalog overrides")
			Expect(patches).To(HaveLen(1), "there should be one kustomize patch for the catalog overrides")

			Eventually(func(g Gomega) {
				g.Expect(test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(catalog), catalog)).To(Succeed(), "there should be no error getting the Catalog")
				kustomization := &kustomizev1.Kustomization{}
				kustomization.SetName(catalog.Name)
				kustomization.SetNamespace(catalog.Namespace)
				g.Expect(test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(kustomization), kustomization)).To(Succeed(), "there should be no error getting the Kustomization")
				g.Expect(kustomization.Spec.Patches).To(HaveLen(1), "Flux Kustomization should have one patch for the override")
				g.Expect(kustomization.Spec.Patches[0].Patch).To(Equal(patches[0].Patch), "Flux Kustomization patch in spec should match the catalog generated patches")
			}).Should(Succeed(), "Flux Kustomization .spec.patches should be updated with the catalog overrides")
		})
	})
})
