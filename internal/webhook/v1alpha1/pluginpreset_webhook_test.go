// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/test"
)

const (
	pluginPresetDefinition = "pluginpreset-admission"
	pluginPresetUpdate     = "pluginpreset-update"
	pluginPresetCreate     = "pluginpreset-create"

	teamWithSupportGroupName = "team-support-true"
)

var _ = Describe("PluginPreset Admission Tests", Ordered, func() {
	var teamWithSupportGroupTrue *greenhousev1alpha1.Team

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

		By("creating a support-group:true Team")
		teamWithSupportGroupTrue = test.NewTeam(test.Ctx, teamWithSupportGroupName, test.TestNamespace, test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))
		Expect(test.K8sClient.Create(test.Ctx, teamWithSupportGroupTrue)).To(Succeed(), "there should be no error creating the Team")
	})

	AfterAll(func() {
		pluginDefinition := &greenhousev1alpha1.PluginDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: pluginPresetDefinition,
			},
		}
		Expect(test.K8sClient.Delete(test.Ctx, pluginDefinition)).To(Succeed(), "failed to delete test PluginDefinition")

		By("deleting the test Team")
		test.EventuallyDeleted(test.Ctx, test.K8sClient, teamWithSupportGroupTrue)
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
		Expect(err.Error()).To(Or(ContainSubstring("ClusterSelector must be set"), ContainSubstring("must specify either spec.clusterSelector.clusterName or spec.clusterSelector.labelSelector")))
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
				Labels:    map[string]string{greenhouseapis.LabelKeyOwnedBy: teamWithSupportGroupName},
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

		Eventually(func(g Gomega) {
			g.Expect(test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(cut), cut)).
				To(Succeed(), "there must be no error getting the plugin preset")
			base := cut.DeepCopy()
			annotations := cut.GetAnnotations()
			delete(annotations, greenhousev1alpha1.PreventDeletionAnnotation)
			cut.SetAnnotations(annotations)
			g.Expect(test.K8sClient.Patch(test.Ctx, cut, client.MergeFrom(base))).To(Succeed(), "there must be no error updating the pluginpreset")
		}).Should(Succeed(), "there should be no error removing the deletion projection")
		test.EventuallyDeleted(test.Ctx, test.K8sClient, cut)
	})

	It("should reject delete operation when PluginPreset has prevent deletion annotation", func() {
		pluginPreset := &greenhousev1alpha1.PluginPreset{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pluginPresetUpdate,
				Namespace: test.TestNamespace,
				Labels:    map[string]string{greenhouseapis.LabelKeyOwnedBy: teamWithSupportGroupName},
				Annotations: map[string]string{
					greenhousev1alpha1.PreventDeletionAnnotation: "true",
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

var _ = Describe("Validate Plugin OptionValues for PluginPreset", func() {
	DescribeTable("Validate OptionValues in .Spec.Plugin contain either Value or ValueFrom", func(value *apiextensionsv1.JSON, valueFrom *greenhousemetav1alpha1.ValueFromSource, expErr bool) {
		pluginPreset := &greenhousev1alpha1.PluginPreset{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PluginPreset",
				APIVersion: greenhousev1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-plugin-preset",
				Namespace: test.TestNamespace,
			},
			Spec: greenhousev1alpha1.PluginPresetSpec{
				Plugin: greenhousev1alpha1.PluginSpec{
					PluginDefinition: "test",
					OptionValues: []greenhousemetav1alpha1.PluginOptionValue{
						{
							Name:      "test",
							Value:     value,
							ValueFrom: valueFrom,
						},
					},
				},
			},
		}

		var defaultVal *apiextensionsv1.JSON
		var optionType greenhousev1alpha1.PluginOptionType
		switch {
		case value != nil:
			defaultVal = value
			optionType = greenhousev1alpha1.PluginOptionTypeString
		case valueFrom != nil:
			defaultVal = test.MustReturnJSONFor(valueFrom.Secret.Name)
			optionType = greenhousev1alpha1.PluginOptionTypeSecret
		}

		pluginDefinition := &greenhousev1alpha1.PluginDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "greenhouse",
				Name:      "testPlugin",
			},
			Spec: greenhousev1alpha1.PluginDefinitionSpec{
				Options: []greenhousev1alpha1.PluginOption{
					{
						Name:    "test",
						Default: defaultVal,
						Type:    optionType,
					},
				},
			},
		}

		errList := validatePluginOptionValuesForPreset(pluginPreset, pluginDefinition)
		switch expErr {
		case true:
			Expect(errList).ToNot(BeEmpty(), "expected an error, got nil")
		default:
			Expect(errList).To(BeEmpty(), "expected no error, got %v", errList)
		}
	},
		Entry("Value and ValueFrom nil", nil, nil, true),
		Entry("Value and ValueFrom not nil", test.MustReturnJSONFor("test"), &greenhousemetav1alpha1.ValueFromSource{Secret: &greenhousemetav1alpha1.SecretKeyReference{Name: "my-secret"}}, true),
		Entry("Value not nil", test.MustReturnJSONFor("test"), nil, false),
		Entry("ValueFrom not nil", nil, &greenhousemetav1alpha1.ValueFromSource{Secret: &greenhousemetav1alpha1.SecretKeyReference{Name: "my-secret", Key: "secret-key"}}, false),
	)

	DescribeTable("Validate OptionValues in .Spec.ClusterOptionOverrides contain either Value or ValueFrom", func(value *apiextensionsv1.JSON, valueFrom *greenhousemetav1alpha1.ValueFromSource, expErr bool) {
		pluginPreset := &greenhousev1alpha1.PluginPreset{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PluginPreset",
				APIVersion: greenhousev1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-plugin-preset",
				Namespace: test.TestNamespace,
			},
			Spec: greenhousev1alpha1.PluginPresetSpec{
				Plugin: greenhousev1alpha1.PluginSpec{
					PluginDefinition: "test",
					OptionValues:     []greenhousemetav1alpha1.PluginOptionValue{},
				},
				ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
					{
						ClusterName: "test-cluster",
						Overrides: []greenhousemetav1alpha1.PluginOptionValue{
							{
								Name:      "test",
								Value:     value,
								ValueFrom: valueFrom,
							},
						},
					},
				},
			},
		}

		var defaultVal *apiextensionsv1.JSON
		var optionType greenhousev1alpha1.PluginOptionType
		switch {
		case value != nil:
			defaultVal = value
			optionType = greenhousev1alpha1.PluginOptionTypeString
		case valueFrom != nil:
			defaultVal = test.MustReturnJSONFor(valueFrom.Secret.Name)
			optionType = greenhousev1alpha1.PluginOptionTypeSecret
		}

		pluginDefinition := &greenhousev1alpha1.PluginDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "greenhouse",
				Name:      "testPlugin",
			},
			Spec: greenhousev1alpha1.PluginDefinitionSpec{
				Options: []greenhousev1alpha1.PluginOption{
					{
						Name:    "test",
						Default: defaultVal,
						Type:    optionType,
					},
				},
			},
		}

		errList := validatePluginOptionValuesForPreset(pluginPreset, pluginDefinition)
		switch expErr {
		case true:
			Expect(errList).ToNot(BeEmpty(), "expected an error, got nil")
		default:
			Expect(errList).To(BeEmpty(), "expected no error, got %v", errList)
		}
	},
		Entry("Value and ValueFrom nil", nil, nil, true),
		Entry("Value and ValueFrom not nil", test.MustReturnJSONFor("test"), &greenhousemetav1alpha1.ValueFromSource{Secret: &greenhousemetav1alpha1.SecretKeyReference{Name: "my-secret"}}, true),
		Entry("Value not nil", test.MustReturnJSONFor("test"), nil, false),
		Entry("ValueFrom not nil", nil, &greenhousemetav1alpha1.ValueFromSource{Secret: &greenhousemetav1alpha1.SecretKeyReference{Name: "my-secret", Key: "secret-key"}}, false),
	)

	DescribeTable("Validate OptionValues in .Spec.Plugin are consistent with PluginOption Type", func(defaultValue any, defaultType greenhousev1alpha1.PluginOptionType, actValue any, expErr bool) {
		pluginDefinition := &greenhousev1alpha1.PluginDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "greenhouse",
				Name:      "testPlugin",
			},
			Spec: greenhousev1alpha1.PluginDefinitionSpec{
				Options: []greenhousev1alpha1.PluginOption{
					{
						Name:    "test",
						Default: test.MustReturnJSONFor(defaultValue),
						Type:    defaultType,
					},
				},
			},
		}

		pluginPreset := &greenhousev1alpha1.PluginPreset{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PluginPreset",
				APIVersion: greenhousev1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-plugin-preset",
				Namespace: test.TestNamespace,
			},
			Spec: greenhousev1alpha1.PluginPresetSpec{
				Plugin: greenhousev1alpha1.PluginSpec{
					PluginDefinition: "test",
					OptionValues: []greenhousemetav1alpha1.PluginOptionValue{
						{
							Name:  "test",
							Value: test.MustReturnJSONFor(actValue),
						},
					},
				},
			},
		}

		errList := validatePluginOptionValuesForPreset(pluginPreset, pluginDefinition)
		switch expErr {
		case true:
			Expect(errList).ToNot(BeEmpty(), "expected an error, got nil")
		default:
			Expect(errList).To(BeEmpty(), "expected no error, got %v", errList)
		}
	},
		Entry("PluginOption Value Consistent With PluginOption Type Bool", false, greenhousev1alpha1.PluginOptionTypeBool, true, false),
		Entry("PluginOption Value Inconsistent With PluginOption Type Bool", true, greenhousev1alpha1.PluginOptionTypeBool, "notabool", true),
		Entry("PluginOption Value Consistent With PluginOption Type String", "string", greenhousev1alpha1.PluginOptionTypeString, "mystring", false),
		Entry("PluginOption Value Consistent With PluginOption Type String Escaped Integer", "1", greenhousev1alpha1.PluginOptionTypeString, "1", false),
		Entry("PluginOption Value Inconsistent With PluginOption Type String", "string", greenhousev1alpha1.PluginOptionTypeString, 1, true),
		Entry("PluginOption Value Consistent With PluginOption Type Int", 1, greenhousev1alpha1.PluginOptionTypeInt, 1, false),
		Entry("PluginOption Value Inconsistent With PluginOption Type Int", 1, greenhousev1alpha1.PluginOptionTypeInt, "one", true),
		Entry("PluginOption Value Consistent With PluginOption Type List", []string{"one", "two"}, greenhousev1alpha1.PluginOptionTypeList, []string{"one", "two", "three"}, false),
		Entry("PluginOption Value Inconsistent With PluginOption Type List", []string{"one", "two"}, greenhousev1alpha1.PluginOptionTypeList, "one,two", true),
		Entry("PluginOption Value Consistent With PluginOption Type Map", map[string]any{"key": "value"}, greenhousev1alpha1.PluginOptionTypeMap, map[string]any{"key": "custom"}, false),
		Entry("PluginOption Value Inconsistent With PluginOption Type Map", map[string]any{"key": "value"}, greenhousev1alpha1.PluginOptionTypeMap, "one", true),
		Entry("PluginOption Value Consistent With PluginOption Type Map Nested Map", map[string]any{"key": map[string]any{"nestedKey": "value"}}, greenhousev1alpha1.PluginOptionTypeMap, map[string]any{"key": map[string]any{"nestedKey": "custom"}}, false),
		Entry("PluginOption Value not supported With PluginOption Type Secret", "", greenhousev1alpha1.PluginOptionTypeSecret, "string", true),
	)

	DescribeTable("Validate OptionValues in .Spec.ClusterOptionOverrides are consistent with PluginOption Type", func(defaultValue any, defaultType greenhousev1alpha1.PluginOptionType, actValue any, expErr bool) {
		pluginDefinition := &greenhousev1alpha1.PluginDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "greenhouse",
				Name:      "testPlugin",
			},
			Spec: greenhousev1alpha1.PluginDefinitionSpec{
				Options: []greenhousev1alpha1.PluginOption{
					{
						Name:    "test",
						Default: test.MustReturnJSONFor(defaultValue),
						Type:    defaultType,
					},
				},
			},
		}

		pluginPreset := &greenhousev1alpha1.PluginPreset{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PluginPreset",
				APIVersion: greenhousev1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-plugin-preset",
				Namespace: test.TestNamespace,
			},
			Spec: greenhousev1alpha1.PluginPresetSpec{
				Plugin: greenhousev1alpha1.PluginSpec{
					PluginDefinition: "test",
					OptionValues:     []greenhousemetav1alpha1.PluginOptionValue{},
				},
				ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
					{
						ClusterName: "test-cluster",
						Overrides: []greenhousemetav1alpha1.PluginOptionValue{
							{
								Name:  "test",
								Value: test.MustReturnJSONFor(actValue),
							},
						},
					},
				},
			},
		}

		errList := validatePluginOptionValuesForPreset(pluginPreset, pluginDefinition)
		switch expErr {
		case true:
			Expect(errList).ToNot(BeEmpty(), "expected an error, got nil")
		default:
			Expect(errList).To(BeEmpty(), "expected no error, got %v", errList)
		}
	},
		Entry("PluginOption Value Consistent With PluginOption Type Bool", false, greenhousev1alpha1.PluginOptionTypeBool, true, false),
		Entry("PluginOption Value Inconsistent With PluginOption Type Bool", true, greenhousev1alpha1.PluginOptionTypeBool, "notabool", true),
		Entry("PluginOption Value Consistent With PluginOption Type String", "string", greenhousev1alpha1.PluginOptionTypeString, "mystring", false),
		Entry("PluginOption Value Consistent With PluginOption Type String Escaped Integer", "1", greenhousev1alpha1.PluginOptionTypeString, "1", false),
		Entry("PluginOption Value Inconsistent With PluginOption Type String", "string", greenhousev1alpha1.PluginOptionTypeString, 1, true),
		Entry("PluginOption Value Consistent With PluginOption Type Int", 1, greenhousev1alpha1.PluginOptionTypeInt, 1, false),
		Entry("PluginOption Value Inconsistent With PluginOption Type Int", 1, greenhousev1alpha1.PluginOptionTypeInt, "one", true),
		Entry("PluginOption Value Consistent With PluginOption Type List", []string{"one", "two"}, greenhousev1alpha1.PluginOptionTypeList, []string{"one", "two", "three"}, false),
		Entry("PluginOption Value Inconsistent With PluginOption Type List", []string{"one", "two"}, greenhousev1alpha1.PluginOptionTypeList, "one,two", true),
		Entry("PluginOption Value Consistent With PluginOption Type Map", map[string]any{"key": "value"}, greenhousev1alpha1.PluginOptionTypeMap, map[string]any{"key": "custom"}, false),
		Entry("PluginOption Value Inconsistent With PluginOption Type Map", map[string]any{"key": "value"}, greenhousev1alpha1.PluginOptionTypeMap, "one", true),
		Entry("PluginOption Value Consistent With PluginOption Type Map Nested Map", map[string]any{"key": map[string]any{"nestedKey": "value"}}, greenhousev1alpha1.PluginOptionTypeMap, map[string]any{"key": map[string]any{"nestedKey": "custom"}}, false),
		Entry("PluginOption Value not supported With PluginOption Type Secret", "", greenhousev1alpha1.PluginOptionTypeSecret, "string", true),
	)

	DescribeTable("Validate OptionValues in .Spec.Plugin reference a Secret", func(actValue *greenhousemetav1alpha1.ValueFromSource, expErr bool) {
		pluginDefinition := &greenhousev1alpha1.PluginDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "greenhouse",
				Name:      "testPlugin",
			},
			Spec: greenhousev1alpha1.PluginDefinitionSpec{
				Options: []greenhousev1alpha1.PluginOption{
					{
						Name: "test",
						Type: greenhousev1alpha1.PluginOptionTypeSecret,
					},
				},
			},
		}

		pluginPreset := &greenhousev1alpha1.PluginPreset{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PluginPreset",
				APIVersion: greenhousev1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-plugin-preset",
				Namespace: test.TestNamespace,
			},
			Spec: greenhousev1alpha1.PluginPresetSpec{
				Plugin: greenhousev1alpha1.PluginSpec{
					PluginDefinition: "test",
					OptionValues: []greenhousemetav1alpha1.PluginOptionValue{
						{
							Name:      "test",
							ValueFrom: actValue,
						},
					},
				},
			},
		}

		errList := validatePluginOptionValuesForPreset(pluginPreset, pluginDefinition)
		switch expErr {
		case true:
			Expect(errList).ToNot(BeEmpty(), "expected an error, got nil")
		default:
			Expect(errList).To(BeEmpty(), "expected no error, got %v", errList)
		}
	},
		Entry("PluginOption ValueFrom has a valid SecretReference", &greenhousemetav1alpha1.ValueFromSource{Secret: &greenhousemetav1alpha1.SecretKeyReference{Name: "secret", Key: "key"}}, false),
		Entry("PluginOption ValueFrom is missing SecretReference Name", &greenhousemetav1alpha1.ValueFromSource{Secret: &greenhousemetav1alpha1.SecretKeyReference{Key: "key"}}, true),
		Entry("PluginOption ValueFrom is missing SecretReference Key", &greenhousemetav1alpha1.ValueFromSource{Secret: &greenhousemetav1alpha1.SecretKeyReference{Name: "secret"}}, true),
		Entry("PluginOption ValueFrom does not contain a SecretReference", nil, true),
	)

	DescribeTable("Validate OptionValues in .Spec.ClusterOptionOverrides reference a Secret", func(actValue *greenhousemetav1alpha1.ValueFromSource, expErr bool) {
		pluginDefinition := &greenhousev1alpha1.PluginDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "greenhouse",
				Name:      "testPlugin",
			},
			Spec: greenhousev1alpha1.PluginDefinitionSpec{
				Options: []greenhousev1alpha1.PluginOption{
					{
						Name: "test",
						Type: greenhousev1alpha1.PluginOptionTypeSecret,
					},
				},
			},
		}

		pluginPreset := &greenhousev1alpha1.PluginPreset{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PluginPreset",
				APIVersion: greenhousev1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-plugin-preset",
				Namespace: test.TestNamespace,
			},
			Spec: greenhousev1alpha1.PluginPresetSpec{
				Plugin: greenhousev1alpha1.PluginSpec{
					PluginDefinition: "test",
					OptionValues:     []greenhousemetav1alpha1.PluginOptionValue{},
				},
				ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
					{
						ClusterName: "test-cluster",
						Overrides: []greenhousemetav1alpha1.PluginOptionValue{
							{
								Name:      "test",
								ValueFrom: actValue,
							},
						},
					},
				},
			},
		}

		errList := validatePluginOptionValuesForPreset(pluginPreset, pluginDefinition)
		switch expErr {
		case true:
			Expect(errList).ToNot(BeEmpty(), "expected an error, got nil")
		default:
			Expect(errList).To(BeEmpty(), "expected no error, got %v", errList)
		}
	},
		Entry("PluginOption ValueFrom has a valid SecretReference", &greenhousemetav1alpha1.ValueFromSource{Secret: &greenhousemetav1alpha1.SecretKeyReference{Name: "secret", Key: "key"}}, false),
		Entry("PluginOption ValueFrom is missing SecretReference Name", &greenhousemetav1alpha1.ValueFromSource{Secret: &greenhousemetav1alpha1.SecretKeyReference{Key: "key"}}, true),
		Entry("PluginOption ValueFrom is missing SecretReference Key", &greenhousemetav1alpha1.ValueFromSource{Secret: &greenhousemetav1alpha1.SecretKeyReference{Name: "secret"}}, true),
		Entry("PluginOption ValueFrom does not contain a SecretReference", nil, true),
	)

	Describe("Validate PluginPreset does not have to specify required options", func() {
		pluginDefinition := &greenhousev1alpha1.PluginDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "greenhouse",
				Name:      "testPlugin",
			},
			Spec: greenhousev1alpha1.PluginDefinitionSpec{
				Options: []greenhousev1alpha1.PluginOption{
					{
						Name:     "test",
						Type:     greenhousev1alpha1.PluginOptionTypeString,
						Required: true,
					},
				},
			},
		}
		It("should accept a PluginPreset with missing required options", func() {
			pluginPreset := &greenhousev1alpha1.PluginPreset{
				TypeMeta: metav1.TypeMeta{
					Kind:       "PluginPreset",
					APIVersion: greenhousev1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-plugin-preset",
					Namespace: test.TestNamespace,
				},
				Spec: greenhousev1alpha1.PluginPresetSpec{
					Plugin: greenhousev1alpha1.PluginSpec{
						PluginDefinition: "test",
						OptionValues:     []greenhousemetav1alpha1.PluginOptionValue{},
					},
					ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
						{
							ClusterName: "test-cluster",
							Overrides:   []greenhousemetav1alpha1.PluginOptionValue{},
						},
					},
				},
			}
			errList := validatePluginOptionValuesForPreset(pluginPreset, pluginDefinition)
			Expect(errList).To(BeEmpty(), "unexpected error")
		})
	})
})
