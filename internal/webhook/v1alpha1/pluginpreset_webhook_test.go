// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/test"
)

const (
	pluginPresetClusterDefinition    = "pluginpreset-admission"
	pluginPresetNamespacedDefinition = "pluginpreset-namespaced"
	pluginPresetUpdate               = "pluginpreset-update"
	pluginPresetCreate               = "pluginpreset-create"

	teamWithSupportGroupName = "team-support-true"
)

var _ = Describe("PluginPreset Admission Tests", Ordered, func() {
	var teamWithSupportGroupTrue *greenhousev1alpha1.Team

	BeforeAll(func() {
		clusterPluginDefinition := test.NewClusterPluginDefinition(test.Ctx, pluginPresetClusterDefinition)
		Expect(test.K8sClient.Create(test.Ctx, clusterPluginDefinition)).To(Succeed(), "failed to create test ClusterPluginDefinition")
		namespacedPluginDefinition := test.NewPluginDefinition(test.Ctx, pluginPresetNamespacedDefinition, test.TestNamespace)
		Expect(test.K8sClient.Create(test.Ctx, namespacedPluginDefinition)).To(Succeed(), "failed to create test PluginDefinition")
		By("creating a support-group:true Team")
		teamWithSupportGroupTrue = test.NewTeam(test.Ctx, teamWithSupportGroupName, test.TestNamespace, test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))
		Expect(test.K8sClient.Create(test.Ctx, teamWithSupportGroupTrue)).To(Succeed(), "there should be no error creating the Team")
	})

	AfterAll(func() {
		clusterPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{
			ObjectMeta: metav1.ObjectMeta{Name: pluginPresetClusterDefinition},
		}
		Expect(test.K8sClient.Delete(test.Ctx, clusterPluginDefinition)).To(Succeed(), "failed to delete test ClusterPluginDefinition")
		namespacedPluginDefinition := &greenhousev1alpha1.PluginDefinition{
			ObjectMeta: metav1.ObjectMeta{Name: pluginPresetNamespacedDefinition, Namespace: test.TestNamespace},
		}
		Expect(test.K8sClient.Delete(test.Ctx, namespacedPluginDefinition)).To(Succeed(), "failed to delete test PluginDefinition")

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
		Expect(err.Error()).To(ContainSubstring("PluginDefinition name must be set"))
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
					PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
						Name: pluginPresetClusterDefinition,
						Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
					},
				},
				ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{},
			},
		}

		err := test.K8sClient.Create(test.Ctx, cut)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("ClusterSelector must be set"))
	})

	It("should reject PluginPreset with non-existing ClusterPluginDefinition", func() {
		cut := &greenhousev1alpha1.PluginPreset{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pluginPresetCreate,
				Namespace: test.TestNamespace,
			},
			Spec: greenhousev1alpha1.PluginPresetSpec{
				Plugin: greenhousev1alpha1.PluginSpec{
					PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
						Name: "non-existing",
						Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
					},
				},
				ClusterSelector:        metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
				ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{},
			},
		}

		err := test.K8sClient.Create(test.Ctx, cut)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("PluginDefinition non-existing does not exist"))
	})

	It("should reject PluginPreset with non-existing PluginDefinition", func() {
		cut := &greenhousev1alpha1.PluginPreset{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pluginPresetCreate,
				Namespace: test.TestNamespace,
			},
			Spec: greenhousev1alpha1.PluginPresetSpec{
				Plugin: greenhousev1alpha1.PluginSpec{
					PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
						Name: "non-existing",
						Kind: greenhousev1alpha1.PluginDefinitionKind,
					},
				},
				ClusterSelector:        metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
				ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{},
			},
		}

		err := test.K8sClient.Create(test.Ctx, cut)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("PluginDefinition non-existing does not exist"))
	})

	It("should correctly default the PluginDefinitionRef for existing PluginDefinition", func() {
		cut := test.NewPluginPreset(pluginPresetCreate, test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, teamWithSupportGroupName),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}}),
			test.WithPluginPresetPluginSpec(greenhousev1alpha1.PluginSpec{
				PluginDefinition: pluginPresetNamespacedDefinition,
			}),
		)
		Expect(test.K8sClient.Create(test.Ctx, cut)).
			To(Succeed(), "there must be no error creating the PluginPreset")
		Expect(cut.Spec.Plugin.PluginDefinitionRef.Name).To(Equal(pluginPresetNamespacedDefinition), "PluginDefinitionRef name should be defaulted")
		Expect(cut.Spec.Plugin.PluginDefinitionRef.Kind).To(Equal(greenhousev1alpha1.PluginDefinitionKind), "PluginDefinitionRef kind should be defaulted to PluginDefinition")
		test.EventuallyDeleted(test.Ctx, test.K8sClient, cut)
	})

	It("should accept PluginPreset with namespaced PluginDefinition", func() {
		cut := test.NewPluginPreset(pluginPresetCreate, test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, teamWithSupportGroupName),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}}),
			test.WithPluginPresetPluginSpec(greenhousev1alpha1.PluginSpec{
				PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
					Name: pluginPresetNamespacedDefinition,
					Kind: greenhousev1alpha1.PluginDefinitionKind,
				},
			}),
		)
		Expect(test.K8sClient.Create(test.Ctx, cut)).
			To(Succeed(), "there must be no error creating the PluginPreset")
		test.EventuallyDeleted(test.Ctx, test.K8sClient, cut)
	})

	It("should accept and reject updates to the PluginPreset", func() {
		cut := test.NewPluginPreset(pluginPresetUpdate, test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, teamWithSupportGroupName),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}}),
			test.WithPluginPresetPluginSpec(greenhousev1alpha1.PluginSpec{
				PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
					Name: pluginPresetClusterDefinition,
					Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
				},
			}),
		)
		Expect(test.K8sClient.Create(test.Ctx, cut)).
			To(Succeed(), "there must be no error creating the PluginPreset")

		_, err := clientutil.CreateOrPatch(test.Ctx, test.K8sClient, cut, func() error {
			cut.Spec.ClusterSelector.MatchLabels["foo"] = "baz"
			return nil
		})
		Expect(err).
			NotTo(HaveOccurred(), "there must be no error updating the PluginPreset clusterSelector")

		_, err = clientutil.CreateOrPatch(test.Ctx, test.K8sClient, cut, func() error {
			cut.Spec.Plugin.PluginDefinitionRef.Name = "new-definition"
			return nil
		})
		Expect(err).
			To(HaveOccurred(), "there must be an error updating the PluginPreset pluginDefinition name")
		Expect(err.Error()).
			To(ContainSubstring("field is immutable"), "the error must reflect the field is immutable")

		_, err = clientutil.CreateOrPatch(test.Ctx, test.K8sClient, cut, func() error {
			cut.Spec.Plugin.PluginDefinitionRef.Kind = greenhousev1alpha1.PluginDefinitionKind
			return nil
		})
		Expect(err).
			NotTo(HaveOccurred(), "there must be no error updating the PluginPreset pluginDefinition kind")

		_, err = clientutil.CreateOrPatch(test.Ctx, test.K8sClient, cut, func() error {
			cut.Spec.Plugin.ClusterName = "foo"
			return nil
		})
		Expect(err).
			To(HaveOccurred(), "there must be an error updating the PluginPreset clusterName")
		Expect(err.Error()).
			To(ContainSubstring("field is immutable"), "the error must reflect the field is immutable")

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
					PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
						Name: pluginPresetClusterDefinition,
						Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
					},
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
	DescribeTable("Validate OptionValues in .Spec.Plugin contain either Value or ValueFrom", func(value *apiextensionsv1.JSON, valueFrom *greenhousev1alpha1.PluginValueFromSource, expErr bool) {
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
					PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
						Name: "test",
						Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
					},
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
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

		pluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{
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

		errList := validatePluginOptionValuesForPreset(pluginPreset, pluginDefinition.Name, pluginDefinition.Spec)
		switch expErr {
		case true:
			Expect(errList).ToNot(BeEmpty(), "expected an error, got nil")
		default:
			Expect(errList).To(BeEmpty(), "expected no error, got %v", errList)
		}
	},
		Entry("Value and ValueFrom nil", nil, nil, true),
		Entry("Value and ValueFrom not nil", test.MustReturnJSONFor("test"), &greenhousev1alpha1.PluginValueFromSource{Secret: &greenhousev1alpha1.SecretKeyReference{Name: "my-secret"}}, true),
		Entry("Value not nil", test.MustReturnJSONFor("test"), nil, false),
		Entry("ValueFrom not nil", nil, &greenhousev1alpha1.PluginValueFromSource{Secret: &greenhousev1alpha1.SecretKeyReference{Name: "my-secret", Key: "secret-key"}}, false),
	)

	DescribeTable("Validate OptionValues in .Spec.ClusterOptionOverrides contain either Value or ValueFrom", func(value *apiextensionsv1.JSON, valueFrom *greenhousev1alpha1.PluginValueFromSource, expErr bool) {
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
					PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
						Name: "test",
						Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
					},
					OptionValues: []greenhousev1alpha1.PluginOptionValue{},
				},
				ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
					{
						ClusterName: "test-cluster",
						Overrides: []greenhousev1alpha1.PluginOptionValue{
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

		pluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{
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

		errList := validatePluginOptionValuesForPreset(pluginPreset, pluginDefinition.Name, pluginDefinition.Spec)
		switch expErr {
		case true:
			Expect(errList).ToNot(BeEmpty(), "expected an error, got nil")
		default:
			Expect(errList).To(BeEmpty(), "expected no error, got %v", errList)
		}
	},
		Entry("Value and ValueFrom nil", nil, nil, true),
		Entry("Value and ValueFrom not nil", test.MustReturnJSONFor("test"), &greenhousev1alpha1.PluginValueFromSource{Secret: &greenhousev1alpha1.SecretKeyReference{Name: "my-secret"}}, true),
		Entry("Value not nil", test.MustReturnJSONFor("test"), nil, false),
		Entry("ValueFrom not nil", nil, &greenhousev1alpha1.PluginValueFromSource{Secret: &greenhousev1alpha1.SecretKeyReference{Name: "my-secret", Key: "secret-key"}}, false),
	)

	DescribeTable("Validate OptionValues in .Spec.Plugin are consistent with PluginOption Type", func(defaultValue any, defaultType greenhousev1alpha1.PluginOptionType, actValue any, expErr bool) {
		pluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{
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
					PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
						Name: "test",
						Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
					},
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "test",
							Value: test.MustReturnJSONFor(actValue),
						},
					},
				},
			},
		}

		errList := validatePluginOptionValuesForPreset(pluginPreset, pluginDefinition.Name, pluginDefinition.Spec)
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
		pluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{
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
					PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
						Name: "test",
						Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
					},
					OptionValues: []greenhousev1alpha1.PluginOptionValue{},
				},
				ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
					{
						ClusterName: "test-cluster",
						Overrides: []greenhousev1alpha1.PluginOptionValue{
							{
								Name:  "test",
								Value: test.MustReturnJSONFor(actValue),
							},
						},
					},
				},
			},
		}

		errList := validatePluginOptionValuesForPreset(pluginPreset, pluginDefinition.Name, pluginDefinition.Spec)
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

	DescribeTable("Validate OptionValues in .Spec.Plugin reference a Secret", func(actValue *greenhousev1alpha1.PluginValueFromSource, expErr bool) {
		pluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{
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
					PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
						Name: "test",
						Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
					},
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:      "test",
							ValueFrom: actValue,
						},
					},
				},
			},
		}

		errList := validatePluginOptionValuesForPreset(pluginPreset, pluginDefinition.Name, pluginDefinition.Spec)
		switch expErr {
		case true:
			Expect(errList).ToNot(BeEmpty(), "expected an error, got nil")
		default:
			Expect(errList).To(BeEmpty(), "expected no error, got %v", errList)
		}
	},
		Entry("PluginOption ValueFrom has a valid SecretReference", &greenhousev1alpha1.PluginValueFromSource{Secret: &greenhousev1alpha1.SecretKeyReference{Name: "secret", Key: "key"}}, false),
		Entry("PluginOption ValueFrom is missing SecretReference Name", &greenhousev1alpha1.PluginValueFromSource{Secret: &greenhousev1alpha1.SecretKeyReference{Key: "key"}}, true),
		Entry("PluginOption ValueFrom is missing SecretReference Key", &greenhousev1alpha1.PluginValueFromSource{Secret: &greenhousev1alpha1.SecretKeyReference{Name: "secret"}}, true),
		Entry("PluginOption ValueFrom does not contain a SecretReference", nil, true),
	)

	DescribeTable("Validate OptionValues in .Spec.ClusterOptionOverrides reference a Secret", func(actValue *greenhousev1alpha1.PluginValueFromSource, expErr bool) {
		pluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{
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
					PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
						Name: "test",
						Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
					},
					OptionValues: []greenhousev1alpha1.PluginOptionValue{},
				},
				ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
					{
						ClusterName: "test-cluster",
						Overrides: []greenhousev1alpha1.PluginOptionValue{
							{
								Name:      "test",
								ValueFrom: actValue,
							},
						},
					},
				},
			},
		}

		errList := validatePluginOptionValuesForPreset(pluginPreset, pluginDefinition.Name, pluginDefinition.Spec)
		switch expErr {
		case true:
			Expect(errList).ToNot(BeEmpty(), "expected an error, got nil")
		default:
			Expect(errList).To(BeEmpty(), "expected no error, got %v", errList)
		}
	},
		Entry("PluginOption ValueFrom has a valid SecretReference", &greenhousev1alpha1.PluginValueFromSource{Secret: &greenhousev1alpha1.SecretKeyReference{Name: "secret", Key: "key"}}, false),
		Entry("PluginOption ValueFrom is missing SecretReference Name", &greenhousev1alpha1.PluginValueFromSource{Secret: &greenhousev1alpha1.SecretKeyReference{Key: "key"}}, true),
		Entry("PluginOption ValueFrom is missing SecretReference Key", &greenhousev1alpha1.PluginValueFromSource{Secret: &greenhousev1alpha1.SecretKeyReference{Name: "secret"}}, true),
		Entry("PluginOption ValueFrom does not contain a SecretReference", nil, true),
	)

	Describe("Validate PluginPreset does not have to specify required options", func() {
		pluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{
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
						PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
							Name: "test",
							Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
						},
						OptionValues: []greenhousev1alpha1.PluginOptionValue{},
					},
					ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
						{
							ClusterName: "test-cluster",
							Overrides:   []greenhousev1alpha1.PluginOptionValue{},
						},
					},
				},
			}
			errList := validatePluginOptionValuesForPreset(pluginPreset, pluginDefinition.Name, pluginDefinition.Spec)
			Expect(errList).To(BeEmpty(), "unexpected error")
		})
	})

	DescribeTable("Validate WaitFor PluginRefs", func(waitForItems []greenhousev1alpha1.WaitForItem, expErr bool) {
		errList := validateWaitForPluginRefs(waitForItems, false)
		switch expErr {
		case true:
			Expect(errList).ToNot(BeEmpty(), "expected an error, got nil")
		default:
			Expect(errList).To(BeEmpty(), "expected no error, got %v", errList)
		}
	},
		Entry("WaitFor has unique items", []greenhousev1alpha1.WaitForItem{
			{PluginRef: greenhousev1alpha1.PluginRef{Name: "global-plugin-1", PluginPreset: ""}},
			{PluginRef: greenhousev1alpha1.PluginRef{Name: "global-plugin-2", PluginPreset: ""}},
			{PluginRef: greenhousev1alpha1.PluginRef{Name: "", PluginPreset: "plugin-a"}},
			{PluginRef: greenhousev1alpha1.PluginRef{Name: "", PluginPreset: "plugin-b"}},
		}, false),
		Entry("WaitFor has PluginRef with both fields set", []greenhousev1alpha1.WaitForItem{
			{PluginRef: greenhousev1alpha1.PluginRef{Name: "global-plugin-1", PluginPreset: "plugin-a"}},
		}, true),
		Entry("WaitFor has PluginRef without any field set", []greenhousev1alpha1.WaitForItem{
			{PluginRef: greenhousev1alpha1.PluginRef{Name: "", PluginPreset: ""}},
		}, true),
		Entry("WaitFor has duplicate items on Name field", []greenhousev1alpha1.WaitForItem{
			{PluginRef: greenhousev1alpha1.PluginRef{Name: "global-plugin-1", PluginPreset: ""}},
			{PluginRef: greenhousev1alpha1.PluginRef{Name: "", PluginPreset: "plugin-b"}},
			{PluginRef: greenhousev1alpha1.PluginRef{Name: "global-plugin-1", PluginPreset: ""}},
			{PluginRef: greenhousev1alpha1.PluginRef{Name: "", PluginPreset: "plugin-a"}},
		}, true),
		Entry("WaitFor has duplicate items on PluginPreset field", []greenhousev1alpha1.WaitForItem{
			{PluginRef: greenhousev1alpha1.PluginRef{Name: "global-plugin-1", PluginPreset: ""}},
			{PluginRef: greenhousev1alpha1.PluginRef{Name: "", PluginPreset: "plugin-a"}},
			{PluginRef: greenhousev1alpha1.PluginRef{Name: "global-plugin-2", PluginPreset: ""}},
			{PluginRef: greenhousev1alpha1.PluginRef{Name: "", PluginPreset: "plugin-a"}},
		}, true),
	)
})
