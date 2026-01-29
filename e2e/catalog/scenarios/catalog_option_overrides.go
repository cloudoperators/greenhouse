// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package scenarios

import (
	"context"
	"strings"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/test"
)

// baseCatalog returns a base Catalog with a single PluginDefinition resource with no overrides.
// the PluginDefinition applied from this Catalog will be used to compare diff against Catalog that applies
// option overrides on the same PluginDefinition with an alias.
func baseCatalog(name, namespace, secretName string) (*greenhousev1alpha1.Catalog, greenhousev1alpha1.CatalogSource) {
	GinkgoHelper()
	catalog := test.NewCatalog(name, namespace)
	source := test.NewCatalogSource(
		test.WithRepository("https://github.com/cloudoperators/extensions-e2e"),
		test.WithRepositoryBranch("main"),
		test.WithCatalogResources([]string{
			"plugindefinitions/pd-cert-manager.yaml",
		}),
	)
	source.SecretName = ptr.To[string](secretName)
	catalog.SetLabels(map[string]string{
		"greenhouse.sap/managed-by": "e2e",
	})
	return catalog, source
}

func overriddenCatalog(name, namespace, secretName string) (*greenhousev1alpha1.Catalog, greenhousev1alpha1.CatalogSource) {
	GinkgoHelper()
	catalog, source := baseCatalog(name, namespace, secretName)
	source.Ref.Branch = ptr.To[string]("dev")
	source.Overrides = append(source.Overrides, greenhousev1alpha1.CatalogOverrides{
		Name:       "cert-manager",
		Alias:      "cert-manager-override",
		Repository: "oci://quay.io/jetstack/charts/cert-manager",
		OptionsOverride: []greenhousev1alpha1.OptionsOverride{
			{
				Name:  "cert-manager.installCRDs",
				Value: test.MustReturnJSONFor(true),
			},
			{
				Name:  "cert-manager.webhook.timeoutSeconds",
				Value: test.MustReturnJSONFor(30),
			},
		},
	})
	return catalog, source
}

func (s *scenario) ExecuteOptionsOverrideScenario(ctx context.Context, namespace string) {
	GinkgoHelper()
	By("creating initial Catalog without overrides")
	var initSource, overriddenSource greenhousev1alpha1.CatalogSource
	initCatalog, initSource := baseCatalog("catalog-no-override", namespace, s.secretName)
	initPluginDefinition := s.createCatalogAndVerifySuccess(ctx, initCatalog, initSource)

	By("creating another Catalog with option overrides")
	overriddenCatalog, overriddenSource := overriddenCatalog("catalog-opt-override", namespace, s.secretName)
	overriddenPluginDefinition := s.createCatalogAndVerifySuccess(ctx, overriddenCatalog, overriddenSource)

	By("comparing the PluginDefinition options before and after override")
	Expect(initPluginDefinition.Spec.Options).ShouldNot(Equal(overriddenPluginDefinition.Spec.Options), "the PluginDefinition options should not be equal after override")
	diff := cmp.Diff(initPluginDefinition.Spec.Options, overriddenPluginDefinition.Spec.Options)
	GinkgoWriter.Printf("PluginDefinition options diff after override:\n%s\n", diff)

	By("cleaning up all PluginDefinitions created in the scenario")
	s.deleteAllScenarioPluginDefinitions(ctx, []string{initCatalog.Name, overriddenCatalog.Name}, namespace)
}

func (s *scenario) createCatalogAndVerifySuccess(ctx context.Context, catalog *greenhousev1alpha1.Catalog, source greenhousev1alpha1.CatalogSource) *greenhousev1alpha1.PluginDefinition {
	GinkgoHelper()
	s.catalog = catalog
	s.catalog.Spec.Sources = append(s.catalog.Spec.Sources, source)
	Expect(s.createCatalogIfNotExists(ctx)).ToNot(HaveOccurred(), "there should be no error creating the initial Catalog")
	s.verifySuccess(ctx, nil)
	groupKey, err := getSourceGroupHash(source, catalog.Name)
	Expect(err).ToNot(HaveOccurred(), "there should be no error getting the source group hash for initial catalog")
	kustomization := s.getKustomizationObject(groupKey)
	err = s.k8sClient.Get(ctx, client.ObjectKeyFromObject(kustomization), kustomization)
	Expect(err).ToNot(HaveOccurred(), "there should be no error getting the Kustomization for initial catalog")
	key := types.NamespacedName{
		Name:      strings.Split(kustomization.Status.Inventory.Entries[0].ID, "_")[1],
		Namespace: catalog.Namespace,
	}
	return checkIfPDExists(ctx, s.k8sClient, key)
}

func (s *scenario) deleteAllScenarioPluginDefinitions(ctx context.Context, labelVals []string, namespace string) {
	GinkgoHelper()
	selector := labels.NewSelector()
	req, err := labels.NewRequirement(
		greenhouseapis.LabelKeyCatalog,
		selection.In,
		labelVals,
	)
	Expect(err).ToNot(HaveOccurred(), "there should be no error creating label requirement for cleanup")
	selector = selector.Add(*req)
	err = s.k8sClient.DeleteAllOf(
		ctx,
		&greenhousev1alpha1.PluginDefinition{},
		client.InNamespace(namespace),
		client.MatchingLabelsSelector{Selector: selector},
	)
	Expect(err).ToNot(HaveOccurred(), "there should be no error deleting all PluginDefinitions in the namespace")
}
