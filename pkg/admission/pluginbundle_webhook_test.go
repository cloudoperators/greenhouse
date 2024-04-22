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
	pluginbundleDefinition = "pluginbundle-admission"
	pluginBundleUpdate     = "pluginbundle-update"
	pluginBundleCreate     = "pluginbundle-create"
)

var _ = Describe("PluginBundleAdmission", Ordered, func() {
	BeforeAll(func() {
		pluginDefinition := &greenhousev1alpha1.PluginDefinition{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PluginDefinition",
				APIVersion: greenhousev1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: pluginbundleDefinition,
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
				Name: pluginbundleDefinition,
			},
		}
		Expect(test.K8sClient.Delete(test.Ctx, pluginDefinition)).To(Succeed(), "failed to delete test PluginDefinition")
	})

	It("should reject PluginBundle without PluginDefinition", func() {
		cut := &greenhousev1alpha1.PluginBundle{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pluginBundleCreate,
				Namespace: test.TestNamespace,
			},
			Spec: greenhousev1alpha1.PluginBundleSpec{
				ClusterSelector: metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
			},
		}

		err := test.K8sClient.Create(test.Ctx, cut)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("PluginDefinition must be set"))
	})

	It("should reject PluginBundle without ClusterSelector", func() {
		cut := &greenhousev1alpha1.PluginBundle{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pluginBundleCreate,
				Namespace: test.TestNamespace,
			},
			Spec: greenhousev1alpha1.PluginBundleSpec{
				PluginDefinition: pluginbundleDefinition,
			},
		}

		err := test.K8sClient.Create(test.Ctx, cut)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("ClusterSelector must be set"))
	})

	It("should reject PluginBundle with non-existing PluginDefinition", func() {
		cut := &greenhousev1alpha1.PluginBundle{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pluginBundleCreate,
				Namespace: test.TestNamespace,
			},
			Spec: greenhousev1alpha1.PluginBundleSpec{
				PluginDefinition: "non-existing",
				ClusterSelector:  metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
			},
		}

		err := test.K8sClient.Create(test.Ctx, cut)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("PluginDefinition non-existing does not exist"))
	})

	It("should reject updates to Immutable Fields of a PluginBundle", func() {
		cut := &greenhousev1alpha1.PluginBundle{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pluginBundleUpdate,
				Namespace: test.TestNamespace,
			},
			Spec: greenhousev1alpha1.PluginBundleSpec{
				PluginDefinition: pluginbundleDefinition,
				ClusterSelector:  metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
			},
		}

		Expect(test.K8sClient.Create(test.Ctx, cut)).To(Succeed())

		_, err := clientutil.CreateOrPatch(test.Ctx, test.K8sClient, cut, func() error {
			cut.Spec.ClusterSelector.MatchLabels["foo"] = "baz"
			return nil
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("field is immutable"))

		_, err = clientutil.CreateOrPatch(test.Ctx, test.K8sClient, cut, func() error {
			cut.Spec.PluginDefinition = "new-definition"
			return nil
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("field is immutable"))

		Expect(test.K8sClient.Delete(test.Ctx, cut)).To(Succeed())
	})
})
