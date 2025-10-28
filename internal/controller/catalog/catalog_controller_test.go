// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"fmt"
	"maps"
	"slices"

	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcev2 "github.com/fluxcd/source-watcher/api/v2/v1beta1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"

	cl "sigs.k8s.io/controller-runtime/pkg/client"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
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

func mockArtifactGeneratorReady(artifact *sourcev2.ArtifactGenerator) error {
	GinkgoHelper()
	newArtifact := &sourcev2.ArtifactGenerator{}
	Expect(test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(artifact), artifact)).To(Succeed(), "there should be no error getting the ArtifactGenerator")
	*newArtifact = *artifact
	artifactReadyCondition := metav1.Condition{
		Type:               string(greenhousemetav1alpha1.ReadyCondition),
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "Succeeded",
		Message:            "",
	}
	newArtifact.Status.Conditions = []metav1.Condition{artifactReadyCondition}
	return patchStatus(artifact, newArtifact)
}

func mockExternalArtifactReady(artifact *sourcev1.ExternalArtifact) error {
	GinkgoHelper()
	err := test.K8sClient.Create(test.Ctx, artifact)
	Expect(cl.IgnoreAlreadyExists(err)).ToNot(HaveOccurred(), "there should be no error creating the external artifact")
	newArtifact := &sourcev1.ExternalArtifact{}
	Expect(test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(artifact), artifact)).To(Succeed(), "there should be no error getting the ExternalArtifact")
	*newArtifact = *artifact
	artifactReadyCondition := metav1.Condition{
		Type:               string(greenhousemetav1alpha1.ReadyCondition),
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "Succeeded",
		Message:            "",
	}
	newArtifact.Status.Conditions = []metav1.Condition{artifactReadyCondition}
	newArtifact.Status.Artifact = &fluxmeta.Artifact{
		LastUpdateTime: metav1.Now(),
		Path:           "externalartifact/greenhouse/artifact-hash/123456.tar.gz",
		Digest:         "sha256:dummyhash",
		Size:           ptr.To[int64](1234),
		URL:            "http://source-watcher.flux-system.svc.cluster.local./externalartifact/greenhouse/artifact-hash/123456.tar.gz",
		Revision:       "latest@sha256:dummyhash",
	}
	return patchStatus(artifact, newArtifact)
}

func mockGitRepositoryReady(gitRepository *sourcev1.GitRepository, status metav1.ConditionStatus) error {
	GinkgoHelper()
	newGitRepository := &sourcev1.GitRepository{}
	Expect(test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(gitRepository), gitRepository)).To(Succeed(), "there should be no error getting the GitRepository")
	*newGitRepository = *gitRepository
	gitRepositoryReadyCondition := metav1.Condition{
		Type:               string(greenhousemetav1alpha1.ReadyCondition),
		Status:             status,
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

const (
	catalogName = "greenhouse-extensions-catalog"
	catalogTest = "catalog-controller-test"
)

var _ = Describe("Catalog controller", Ordered, func() {
	var (
		setup   *test.TestSetup
		catalog *greenhousev1alpha1.Catalog
		// defaultInterval = metav1.Duration{Duration: flux.DefaultInterval}
	)
	BeforeAll(func() {
		setup = test.NewTestSetup(test.Ctx, test.K8sClient, catalogTest)
		catalog = setup.CreateCatalog(
			test.Ctx,
			catalogName,
		)
	})

	Context("When creating or updating a Plugin Definition Catalog", Ordered, func() {
		It("should have Ready=False status when no sources are defined", func() {
			By("creating a catalog with no sources")
			Eventually(func(g Gomega) {
				g.Expect(test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(catalog), catalog)).To(Succeed(), "there should be no error getting the Catalog")
				catalogReady := catalog.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
				g.Expect(catalogReady).ToNot(BeNil(), "the Catalog should have a Ready condition")
				g.Expect(catalogReady.Status).To(Equal(metav1.ConditionFalse), "the Ready condition status should be False")
				g.Expect(catalogReady.Reason).To(Equal(greenhousev1alpha1.CatalogNotReadyReason), "the Ready condition reason should be CatalogNotReady")
			}).Should(Succeed(), "catalog status should be not ready when no sources are defined")
		})

		It("should create Catalog internal resources when a source is defined", func() {
			By("adding a new source to the catalog")
			source := test.NewCatalogSource(
				test.WithRepository("https://github.com/cloudoperators/greenhouse-extensions"),
				test.WithRepositoryBranch("main"),
				test.WithCatalogResources([]string{
					"perses/plugindefinition.yaml",
				}),
			)
			By("updating the catalog with a new branch source")
			catalog = setup.UpdateCatalog(test.Ctx, catalogName, source)
			Expect(createSource(catalog, source)).To(Succeed(), "there should be no error in creating internal catalog resources")
			sourceGroupKey, _, err := getSourceGroupHash(source)
			Expect(err).NotTo(HaveOccurred(), "there should be no error getting source group hash for the source")

			newBranchSource := test.NewCatalogSource(
				test.WithRepository("https://github.com/cloudoperators/greenhouse-extensions"),
				test.WithRepositoryBranch("new-plugin"),
				test.WithCatalogResources([]string{
					"newplugin/plugindefinition.yaml",
				}),
			)
			By("updating the catalog with a new branch source")
			catalog = setup.UpdateCatalog(test.Ctx, catalogName, newBranchSource)
			Expect(createSource(catalog, newBranchSource)).To(Succeed(), "there should be no error in updating internal catalog resources for new branch source")
			newBranchSourceGroupKey, newBranchSourceHash, err := getSourceGroupHash(newBranchSource)
			Expect(err).NotTo(HaveOccurred(), "there should be no error getting source group hash for the new branch source")

			By("checking inventory has multiple sources")
			Eventually(func(g Gomega) {
				g.Expect(test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(catalog), catalog)).To(Succeed(), "there should be no error getting the Catalog")
				inventory := catalog.Status.Inventory
				g.Expect(inventory).ToNot(BeNil(), "the Catalog status inventory should not be nil")
				g.Expect(inventory).To(HaveLen(2), "the Catalog status inventory should contain two items")
				g.Expect(inventory).To(HaveKey(sourceGroupKey), "the Catalog status inventory should have the groupKey for source")
				g.Expect(inventory).To(HaveKey(newBranchSourceGroupKey), "the Catalog status inventory should have the groupKey for the new branch source")
			}).Should(Succeed(), "catalog status should have multiple sources in inventory")

			By("mocking git repository Ready=False condition for the new branch source")
			gitRepository := &sourcev1.GitRepository{}
			gitRepository.SetName(gitRepoArtifactPrefix + "-" + newBranchSourceHash)
			gitRepository.SetNamespace(catalog.Namespace)
			Expect(mockGitRepositoryReady(gitRepository, metav1.ConditionFalse)).To(Succeed(), "there should be no error mocking git repository ready condition to False")

			By("checking catalog status is not ready due to git repository not ready")
			Eventually(func(g Gomega) {
				g.Expect(test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(catalog), catalog)).To(Succeed(), "there should be no error getting the Catalog")
				catalogReady := catalog.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
				g.Expect(catalogReady).ToNot(BeNil(), "the Catalog should have a Ready condition")
				g.Expect(catalogReady.Status).To(Equal(metav1.ConditionFalse), "the Ready condition status should be False")
				g.Expect(catalogReady.Reason).To(Equal(greenhousev1alpha1.CatalogNotReadyReason), "the Ready condition reason should be CatalogNotReady")
			}).Should(Succeed(), "catalog status should be not ready when the sources git repository is not ready")

			By("removing the new branch source from the catalog")
			catalog.Spec.Sources = slices.DeleteFunc(catalog.Spec.Sources, func(catalogSource greenhousev1alpha1.CatalogSource) bool {
				return catalogSource.Repository == newBranchSource.Repository && catalogSource.GetRefValue() == newBranchSource.GetRefValue()
			})
			Expect(test.K8sClient.Update(test.Ctx, catalog)).NotTo(HaveOccurred(), "there should be no error updating the catalog to remove the new branch source")
			By("checking inventory has only one source")
			Eventually(func(g Gomega) {
				g.Expect(test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(catalog), catalog)).To(Succeed(), "there should be no error getting the Catalog")
				inventory := catalog.Status.Inventory
				g.Expect(inventory).ToNot(BeNil(), "the Catalog status inventory should not be nil")
				g.Expect(err).NotTo(HaveOccurred(), "there should be no error getting source group hash for the first source")
				g.Expect(err).NotTo(HaveOccurred(), "there should be no error getting source group hash for the new branch source")
				g.Expect(inventory).To(HaveKey(sourceGroupKey), "the Catalog status inventory should have the first source key")
				g.Expect(inventory[sourceGroupKey]).To(HaveLen(4), "the Catalog status inventory for source should have 4 entries")
				g.Expect(inventory).ToNot(HaveKey(newBranchSourceGroupKey), "the Catalog status inventory should not have the new branch source key")
				g.Expect(inventory).To(HaveLen(1), "the Catalog status inventory should contain one item")
			}).Should(Succeed(), "catalog status should have only the first source in inventory")
		})

		It("should suspend/resume flux resources when catalog is suspended/resumed", func() {
			By("suspending the catalog")
			a := catalog.Annotations
			if a == nil {
				a = make(map[string]string)
			}
			a[lifecycle.SuspendAnnotation] = "true"
			catalog.SetAnnotations(a)
			Expect(test.K8sClient.Update(test.Ctx, catalog)).To(Succeed(), "failed to update Catalog with suspend annotation")

			By("checking if the catalog git repository, kustomization & artifact generator are suspended")
			actSourceStatus := []greenhousev1alpha1.SourceStatus{}
			Eventually(func(g Gomega) {
				g.Expect(test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(catalog), catalog)).To(Succeed(), "there should be no error getting the Catalog")
				g.Expect(catalog.Spec.Sources).To(HaveLen(1), "there should be one source in the catalog")
				sourceGroupKey, _, err := getSourceGroupHash(catalog.Spec.Sources[0])
				g.Expect(err).NotTo(HaveOccurred(), "there should be no error getting source group hash for the source")
				inventory := catalog.Status.Inventory
				g.Expect(inventory).To(HaveKey(sourceGroupKey), "the Catalog status inventory should have the groupKey for source")
				actSourceStatus = inventory[sourceGroupKey]
			}).Should(Succeed(), "there should be no error getting the Catalog after adding suspend annotation")

			var gitRepoName, kustomizationName, artifactGeneratorName string
			for _, sourceStatus := range actSourceStatus {
				switch sourceStatus.Kind {
				case sourcev1.GitRepositoryKind:
					gitRepoName = sourceStatus.Name
				case kustomizev1.KustomizationKind:
					kustomizationName = sourceStatus.Name
				case sourcev2.ArtifactGeneratorKind:
					artifactGeneratorName = sourceStatus.Name
				}
			}

			gitRepository := &sourcev1.GitRepository{}
			gitRepository.SetName(gitRepoName)
			gitRepository.SetNamespace(catalog.Namespace)
			Eventually(func(g Gomega) {
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(gitRepository), gitRepository)
				g.Expect(err).ToNot(HaveOccurred(), "failed to get GitRepository")
				g.Expect(gitRepository.Spec.Suspend).To(BeTrue(), "GitRepository should be suspended")
			}).Should(Succeed(), "GitRepository should be suspended after catalog is suspended")

			kustomization := &kustomizev1.Kustomization{}
			kustomization.SetName(kustomizationName)
			kustomization.SetNamespace(catalog.Namespace)
			Eventually(func(g Gomega) {
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(kustomization), kustomization)
				g.Expect(err).ToNot(HaveOccurred(), "failed to get Kustomization")
				g.Expect(kustomization.Spec.Suspend).To(BeTrue(), "Kustomization should be suspended")
			}).Should(Succeed(), "Kustomization should be suspended after catalog is suspended")

			artifactGenerator := &sourcev2.ArtifactGenerator{}
			artifactGenerator.SetName(artifactGeneratorName)
			artifactGenerator.SetNamespace(catalog.Namespace)
			Eventually(func(g Gomega) {
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(artifactGenerator), artifactGenerator)
				g.Expect(err).ToNot(HaveOccurred(), "failed to get ArtifactGenerator")
				g.Expect(artifactGenerator.Annotations).To(HaveKeyWithValue(sourcev2.ReconcileAnnotation, sourcev2.DisabledValue), "ArtifactGenerator should be suspended")
			}).Should(Succeed(), "ArtifactGenerator should be suspended after catalog is suspended")

			By("resuming the catalog")
			maps.DeleteFunc(catalog.Annotations, func(key, value string) bool {
				return key == lifecycle.SuspendAnnotation
			})
			Expect(test.K8sClient.Update(test.Ctx, catalog)).To(Succeed(), "failed to remove suspend annotation from Catalog")

			By("checking if the catalog git repository and kustomization are resumed")
			Eventually(func(g Gomega) {
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(gitRepository), gitRepository)
				g.Expect(err).ToNot(HaveOccurred(), "failed to get GitRepository")
				g.Expect(gitRepository.Spec.Suspend).To(BeFalse(), "GitRepository should not be suspended")
			}).Should(Succeed(), "GitRepository should be resumed after catalog is resumed")

			Eventually(func(g Gomega) {
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(kustomization), kustomization)
				g.Expect(err).ToNot(HaveOccurred(), "failed to get Kustomization")
				g.Expect(kustomization.Spec.Suspend).To(BeFalse(), "Kustomization should not be suspended")
			}).Should(Succeed(), "Kustomization should be resumed after catalog is resumed")

			Eventually(func(g Gomega) {
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(artifactGenerator), artifactGenerator)
				g.Expect(err).ToNot(HaveOccurred(), "failed to get ArtifactGenerator")
				g.Expect(artifactGenerator.Annotations).ToNot(HaveKey(sourcev2.ReconcileAnnotation), "ArtifactGenerator should be resumed")
			}).Should(Succeed(), "ArtifactGenerator should be resumed after catalog is resumed")
		})
	})
})

func getSourceGroupHash(source greenhousev1alpha1.CatalogSource) (groupKey, hash string, err error) {
	var host, owner, repo string
	host, owner, repo, err = lifecycle.GetOwnerRepoInfo(source.Repository)
	if err != nil {
		return
	}
	ref := source.GetRefValue()
	hash, err = lifecycle.HashValue(fmt.Sprintf("%s-%s-%s-%s-%s", catalogName, host, owner, repo, ref))
	if err != nil {
		return
	}
	groupKey = fmt.Sprintf("%s-%s-%s-%s-%s", host, owner, repo, ref, hash)
	return
}

func createSource(catalog *greenhousev1alpha1.Catalog, source greenhousev1alpha1.CatalogSource) error {
	GinkgoHelper()
	groupKey, hash, err := getSourceGroupHash(source)
	if err != nil {
		return err
	}

	By("checking if the GitRepository is created")
	gitRepository := &sourcev1.GitRepository{}
	gitRepository.SetName(gitRepoArtifactPrefix + "-" + hash)
	gitRepository.SetNamespace(catalog.Namespace)
	Eventually(func(g Gomega) {
		g.Expect(test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(gitRepository), gitRepository)).To(Succeed(), "there should be no error getting the GitRepository")
	}).Should(Succeed(), "flux GitRepository should be created for the Catalog source")
	By("mocking git repository Ready=True condition")
	err = mockGitRepositoryReady(gitRepository, metav1.ConditionTrue)
	if err != nil {
		return err
	}

	By("checking if ArtifactGenerator is created")
	artifactGenerator := &sourcev2.ArtifactGenerator{}
	artifactGenerator.SetName(generatorArtifactPrefix + "-" + hash)
	artifactGenerator.SetNamespace(catalog.Namespace)
	Eventually(func(g Gomega) {
		g.Expect(test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(artifactGenerator), artifactGenerator)).To(Succeed(), "there should be no error getting the ArtifactGenerator")
	}).Should(Succeed(), "flux ArtifactGenerator should be created for the Catalog source")
	By("mocking artifact generator Ready=True condition")

	err = mockArtifactGeneratorReady(artifactGenerator)
	if err != nil {
		return err
	}

	By("mocking external artifact Ready=True condition")
	externalArtifact := &sourcev1.ExternalArtifact{}
	externalArtifact.SetName(externalArtifactPrefix + "-" + hash)
	externalArtifact.SetNamespace(catalog.Namespace)

	err = mockExternalArtifactReady(externalArtifact)
	if err != nil {
		return err
	}

	By("checking if Kustomization is created")
	kustomization := &kustomizev1.Kustomization{}
	kustomization.SetName(kustomizeArtifactPrefix + "-" + hash)
	kustomization.SetNamespace(catalog.Namespace)
	Eventually(func(g Gomega) {
		g.Expect(test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(kustomization), kustomization)).To(Succeed(), "there should be no error getting the Kustomization")
	}).Should(Succeed(), "flux Kustomization should be created for the Catalog source")
	By("mocking kustomization Ready=True condition")

	err = mockKustomizationReady(kustomization)
	if err != nil {
		return err
	}

	By("verifying catalog status has Ready=True condition")
	Eventually(func(g Gomega) {
		g.Expect(test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(catalog), catalog)).To(Succeed(), "there should be no error getting the Catalog")
		inventory := catalog.Status.Inventory
		g.Expect(inventory).ToNot(BeNil(), "the Catalog status inventory should not be nil")
		g.Expect(inventory[groupKey]).To(HaveLen(4), "the Catalog status inventory for the source should have 4 entries")
		catalogReady := catalog.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
		g.Expect(catalogReady).ToNot(BeNil(), "the Catalog should have a Ready condition")
		g.Expect(catalogReady.Status).To(Equal(metav1.ConditionTrue), "the Ready condition status should be True")
		g.Expect(catalogReady.Reason).To(Equal(greenhousev1alpha1.CatalogReadyReason), "the Ready condition reason should be CatalogReady")
	}).Should(Succeed(), "catalog status should be ready when flux kustomization is ready")

	return nil
}
