// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package admission

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

const (
	pluginPresetDefinition = "pluginpreset-admission"
	pluginPresetUpdate     = "pluginpreset-update"
	pluginPresetCreate     = "pluginpreset-create"
)

var _ = Describe("PluginPreset Admission Tests", Ordered, func() {
	BeforeAll(func() {
		pluginDefinition := &greenhousev1alpha1.PluginDefinition{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PluginDefinition",
				APIVersion: greenhousev1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: pluginPresetDefinition,
			},
			Spec: greenhousev1alpha1.PluginDefinitionSpec{
				Description: "Testplugin",
				Version:     "1.0.0",
				HelmChart: &greenhousev1alpha1.HelmChartReference{
					Name:       "./../../test/fixtures/myChart",
					Repository: "dummy",
					Version:    "1.0.0",
				},
			},
		}
		Expect(test.K8sClient.Create(test.Ctx, pluginDefinition)).To(Succeed(), "failed to create test PluginDefinition")
	})

	AfterAll(func() {
		pluginDefinition := &greenhousev1alpha1.PluginDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: pluginPresetDefinition,
			},
		}
		Expect(test.K8sClient.Delete(test.Ctx, pluginDefinition)).To(Succeed(), "failed to delete test PluginDefinition")
	})

	It("should reject PluginPreset without PluginDefinition", func() {
		cut := &greenhousev1alpha1.PluginPreset{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pluginPresetCreate,
				Namespace: test.TestNamespace,
			},
			Spec: greenhousev1alpha1.PluginPresetSpec{
				ClusterSelector: metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
			},
		}

		err := test.K8sClient.Create(test.Ctx, cut)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("PluginDefinition must be set"))
	})

	It("should reject PluginPreset with a PluginSpec containing a ClusterName", func() {
		cut := &greenhousev1alpha1.PluginPreset{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pluginPresetCreate,
				Namespace: test.TestNamespace,
			},
			Spec: greenhousev1alpha1.PluginPresetSpec{
				ClusterSelector: metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
				Plugin: greenhousev1alpha1.PluginSpec{
					ClusterName: "cluster",
				},
			},
		}

		err := test.K8sClient.Create(test.Ctx, cut)
		Expect(err).To(HaveOccurred(), "there should be an error creating the PluginPreset with invalid fields")
		Expect(err.Error()).To(ContainSubstring("ClusterName must not be set"), "the error message should reflect that plugin.clusterName should not be set")
	})

	It("should reject PluginPreset without ClusterSelector", func() {
		cut := &greenhousev1alpha1.PluginPreset{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pluginPresetCreate,
				Namespace: test.TestNamespace,
			},
			Spec: greenhousev1alpha1.PluginPresetSpec{
				Plugin: greenhousev1alpha1.PluginSpec{
					PluginDefinition: pluginPresetDefinition,
				},
				ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{},
			},
		}

		err := test.K8sClient.Create(test.Ctx, cut)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("ClusterSelector must be set"))
	})

	It("should reject PluginPreset with non-existing PluginDefinition", func() {
		cut := &greenhousev1alpha1.PluginPreset{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pluginPresetCreate,
				Namespace: test.TestNamespace,
			},
			Spec: greenhousev1alpha1.PluginPresetSpec{
				Plugin: greenhousev1alpha1.PluginSpec{
					PluginDefinition: "non-existing",
				},
				ClusterSelector:        metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
				ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{},
			},
		}

		err := test.K8sClient.Create(test.Ctx, cut)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("PluginDefinition non-existing does not exist"))
	})

	It("should accept and reject updates to the PluginPreset", func() {
		cut := &greenhousev1alpha1.PluginPreset{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pluginPresetUpdate,
				Namespace: test.TestNamespace,
			},
			Spec: greenhousev1alpha1.PluginPresetSpec{
				Plugin: greenhousev1alpha1.PluginSpec{
					PluginDefinition: pluginPresetDefinition,
				},
				ClusterSelector: metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
			},
		}

		Expect(test.K8sClient.Create(test.Ctx, cut)).
			To(Succeed(), "there must be no error creating the PluginPreset")

		_, err := clientutil.CreateOrPatch(test.Ctx, test.K8sClient, cut, func() error {
			cut.Spec.ClusterSelector.MatchLabels["foo"] = "baz"
			return nil
		})
		Expect(err).
			NotTo(HaveOccurred(), "there must be no error updating the PluginPreset clusterSelector")

		_, err = clientutil.CreateOrPatch(test.Ctx, test.K8sClient, cut, func() error {
			cut.Spec.Plugin.PluginDefinition = "new-definition"
			return nil
		})
		Expect(err).
			To(HaveOccurred(), "there must be an error updating the PluginPreset pluginDefinition")
		Expect(err.Error()).
			To(ContainSubstring("field is immutable"), "the error must reflect the field is immutable")

		_, err = clientutil.CreateOrPatch(test.Ctx, test.K8sClient, cut, func() error {
			cut.Spec.Plugin.ClusterName = "foo"
			return nil
		})
		Expect(err).
			To(HaveOccurred(), "there must be an error updating the PluginPreset clusterName")
		Expect(err.Error()).
			To(ContainSubstring("field is immutable"), "the error must reflect the field is immutable")

		_, err = clientutil.CreateOrPatch(test.Ctx, test.K8sClient, cut, func() error {
			delete(cut.Annotations, preventDeletionAnnotation)
			return nil
		})
		Expect(err).
			ToNot(HaveOccurred())
		Expect(test.K8sClient.Delete(test.Ctx, cut)).
			To(Succeed(), "there must be no error deleting the PluginPreset")
	})

	It("should reject delete operation when PluginPreset has prevent deletion annotation", func() {
		pluginPreset := &greenhousev1alpha1.PluginPreset{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pluginPresetUpdate,
				Namespace: test.TestNamespace,
				Annotations: map[string]string{
					preventDeletionAnnotation: "true",
				},
			},
			Spec: greenhousev1alpha1.PluginPresetSpec{
				Plugin: greenhousev1alpha1.PluginSpec{
					PluginDefinition: pluginPresetDefinition,
				},
				ClusterSelector: metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
			},
		}

		err := test.K8sClient.Create(test.Ctx, pluginPreset)
		Expect(err).ToNot(HaveOccurred())

		err = test.K8sClient.Delete(test.Ctx, pluginPreset)
		Expect(err).To(HaveOccurred())

		pluginPreset.Annotations = map[string]string{}
		err = test.K8sClient.Update(test.Ctx, pluginPreset)
		Expect(err).ToNot(HaveOccurred())

		err = test.K8sClient.Delete(test.Ctx, pluginPreset)
		Expect(err).ToNot(HaveOccurred())
	})
})
