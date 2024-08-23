// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package admission

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

var _ = Describe("Validate Plugin OptionValues", func() {
	DescribeTable("Validate PluginType contains either Value or ValueFrom", func(value *apiextensionsv1.JSON, valueFrom *greenhousev1alpha1.ValueFromSource, expErr bool) {
		plugin := &greenhousev1alpha1.Plugin{
			Spec: greenhousev1alpha1.PluginSpec{
				PluginDefinition: "test",
				ClusterName:      testclustername,
				OptionValues: []greenhousev1alpha1.PluginOptionValue{
					{
						Name:      "test",
						Value:     value,
						ValueFrom: valueFrom,
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

		errList := validatePluginOptionValues(plugin.Spec.OptionValues, pluginDefinition)
		switch expErr {
		case true:
			Expect(errList).ToNot(BeEmpty(), "expected an error, got nil")
		default:
			Expect(errList).To(BeEmpty(), "expected no error, got %v", errList)
		}
	},
		Entry("Value and ValueFrom nil", nil, nil, true),
		Entry("Value and ValueFrom not nil", test.MustReturnJSONFor("test"), &greenhousev1alpha1.ValueFromSource{Secret: &greenhousev1alpha1.SecretKeyReference{Name: "my-secret"}}, true),
		Entry("Value not nil", test.MustReturnJSONFor("test"), nil, false),
		Entry("ValueFrom not nil", nil, &greenhousev1alpha1.ValueFromSource{Secret: &greenhousev1alpha1.SecretKeyReference{Name: "my-secret", Key: "secret-key"}}, false),
	)

	DescribeTable("Validate PluginOptionValue is consistent with PluginOption Type", func(defaultValue any, defaultType greenhousev1alpha1.PluginOptionType, actValue any, expErr bool) {
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

		plugin := &greenhousev1alpha1.Plugin{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "greenhouse",
			},
			Spec: greenhousev1alpha1.PluginSpec{
				PluginDefinition: "test",
				ClusterName:      testclustername,
				OptionValues: []greenhousev1alpha1.PluginOptionValue{
					{
						Name:  "test",
						Value: test.MustReturnJSONFor(actValue),
					},
				},
			},
		}

		errList := validatePluginOptionValues(plugin.Spec.OptionValues, pluginDefinition)
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

	DescribeTable("Validate PluginOptionValue references a Secret", func(actValue *greenhousev1alpha1.ValueFromSource, expErr bool) {
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

		plugin := &greenhousev1alpha1.Plugin{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "greenhouse",
			},
			Spec: greenhousev1alpha1.PluginSpec{
				PluginDefinition: "test",
				OptionValues: []greenhousev1alpha1.PluginOptionValue{
					{
						Name:      "test",
						ValueFrom: actValue,
					},
				},
			},
		}

		errList := validatePluginOptionValues(plugin.Spec.OptionValues, pluginDefinition)
		switch expErr {
		case true:
			Expect(errList).ToNot(BeEmpty(), "expected an error, got nil")
		default:
			Expect(errList).To(BeEmpty(), "expected no error, got %v", errList)
		}
	},
		Entry("PluginOption ValueFrom has a valid SecretReference", &greenhousev1alpha1.ValueFromSource{Secret: &greenhousev1alpha1.SecretKeyReference{Name: "secret", Key: "key"}}, false),
		Entry("PluginOption ValueFrom is missing SecretReference Name", &greenhousev1alpha1.ValueFromSource{Secret: &greenhousev1alpha1.SecretKeyReference{Key: "key"}}, true),
		Entry("PluginOption ValueFrom is missing SecretReference Key", &greenhousev1alpha1.ValueFromSource{Secret: &greenhousev1alpha1.SecretKeyReference{Name: "secret"}}, true),
		Entry("PluginOption ValueFrom does not contain a SecretReference", nil, true),
	)

	Describe("Validate Plugin specifies all required options", func() {
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
		It("should reject a Plugin with missing required options", func() {
			plugin := &greenhousev1alpha1.Plugin{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Plugin",
					APIVersion: greenhousev1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-plugin",
					Namespace: test.TestNamespace,
				},
				Spec: greenhousev1alpha1.PluginSpec{
					PluginDefinition: "test",
					ClusterName:      testclustername,
				},
			}
			errList := validatePluginOptionValues(plugin.Spec.OptionValues, pluginDefinition)
			Expect(errList).NotTo(BeEmpty(), "expected an error, got nil")
		})
		It("should accept a Plugin with supplied required options", func() {
			plugin := &greenhousev1alpha1.Plugin{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Plugin",
					APIVersion: greenhousev1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-plugin",
					Namespace: test.TestNamespace,
				},
				Spec: greenhousev1alpha1.PluginSpec{
					PluginDefinition: "test",
					ClusterName:      testclustername,
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "test",
							Value: test.MustReturnJSONFor("test"),
						},
					},
				},
			}
			errList := validatePluginOptionValues(plugin.Spec.OptionValues, pluginDefinition)
			Expect(errList).To(BeEmpty(), "unexpected error")
		})
	})
})

var testPlugin = &greenhousev1alpha1.Plugin{
	TypeMeta: metav1.TypeMeta{
		Kind:       "Plugin",
		APIVersion: greenhousev1alpha1.GroupVersion.String(),
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-plugin",
		Namespace: test.TestNamespace,
	},
	Spec: greenhousev1alpha1.PluginSpec{
		PluginDefinition: "test-plugindefinition",
		ReleaseNamespace: "test-namespace",
	},
}

var testPluginDefinition = &greenhousev1alpha1.PluginDefinition{
	TypeMeta: metav1.TypeMeta{
		Kind:       "PluginDefinition",
		APIVersion: greenhousev1alpha1.GroupVersion.String(),
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-plugindefinition",
		Namespace: test.TestNamespace,
	},
	Spec: greenhousev1alpha1.PluginDefinitionSpec{
		Description: "TestPluginDefinition",
		Version:     "1.0.0",
		HelmChart: &greenhousev1alpha1.HelmChartReference{
			Name:       "./../../test/fixtures/myChart",
			Repository: "dummy",
			Version:    "1.0.0",
		},
	},
}

var _ = Describe("Validate pluginConfig clusterName", Ordered, func() {
	BeforeAll(func() {
		err := test.K8sClient.Create(test.Ctx, testPluginDefinition)
		Expect(err).ToNot(HaveOccurred(), "there should be no error creating the pluginDefinition")

	})

	AfterAll(func() {
		Expect(test.K8sClient.Delete(test.Ctx, testPlugin)).To(Succeed(), "there should be no error deleting the plugin")
		Eventually(func(g Gomega) {
			g.Expect(test.K8sClient.Delete(test.Ctx, testPluginDefinition)).To(Succeed(), "there should be no error deleting the pluginDefinition")
		}).Should(Succeed(), "there should be plugindefinition should be deleted eventually")
	})

	It("should accept a plugin without a clusterName", func() {
		err := test.K8sClient.Create(test.Ctx, testPlugin)
		expectClusterMustBeSetError(err)
	})

	It("should reject the pluginConfig when the cluster with clusterName does not exist", func() {
		By("creating the pluginConfig")
		testPlugin.Spec.ClusterName = "non-existent-cluster"
		err := test.K8sClient.Create(test.Ctx, testPlugin)
		expectClusterNotFoundError(err)
	})

	It("should accept the pluginConfig when the cluster with clusterName exists", func() {
		By("creating the pluginConfig")
		// reset resourceVersion to avoid conflict, still using same struct
		testPlugin.ResourceVersion = ""
		testPlugin.Spec.ClusterName = testclustername
		err := test.K8sClient.Create(test.Ctx, testPlugin)
		Expect(err).ToNot(HaveOccurred(), "there should be no error creating the pluginConfig")

		By("checking the label on the plugin")
		actPlugin := &greenhousev1alpha1.Plugin{}
		err = test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: testPlugin.Name, Namespace: testPlugin.Namespace}, actPlugin)
		Expect(err).ToNot(HaveOccurred(), "there should be no error getting the plugin")
		Eventually(func() map[string]string {
			return actPlugin.GetLabels()
		}).Should(HaveKeyWithValue(greenhouseapis.LabelKeyCluster, testclustername), "the plugin should have a matching cluster label")
	})

	It("should reject a plugin update where the clusterName changes", func() {
		testPlugin.Spec.ClusterName = "wrong-cluster-name"
		err := test.K8sClient.Update(test.Ctx, testPlugin)

		Expect(err).To(HaveOccurred(), "there should be an error changing the plugin's clusterName")
		Expect(err.Error()).To(ContainSubstring("field is immutable"))
	})

	It("should not allow deletion of the clusterName reference in existing plugin", func() {
		testPlugin.Spec.ClusterName = ""
		err := test.K8sClient.Update(test.Ctx, testPlugin)
		Expect(err).To(HaveOccurred(), "there should be an error changing the plugin's clusterName")
		Expect(err.Error()).To(ContainSubstring("field is immutable"))
		// reset the clusterName for following tests to pass
		testPlugin.Spec.ClusterName = testclustername
	})

	It("should reject a plugin update where the releaseNamespace changes", func() {
		testPlugin.Spec.ReleaseNamespace = "foo-bar"
		err := test.K8sClient.Update(test.Ctx, testPlugin)
		Expect(err).To(HaveOccurred(), "there should be an error changing the plugin's releaseNamespace")
		Expect(err.Error()).To(ContainSubstring("field is immutable"))
	})
})

func expectClusterNotFoundError(err error) {
	Expect(err).To(HaveOccurred(), "there should be an error updating the plugin")
	var statusErr *apierrors.StatusError
	ok := errors.As(err, &statusErr)
	Expect(ok).To(BeTrue(), "error should be a status error")
	Expect(statusErr.ErrStatus.Reason).To(Equal(metav1.StatusReasonForbidden), "the error should be a status forbidden error")
	Expect(statusErr.ErrStatus.Message).To(ContainSubstring("spec.clusterName: Not found"), "the error message should reflect clustername not found")
}

func expectClusterMustBeSetError(err error) {
	Expect(err).To(HaveOccurred(), "there should be an error updating the plugin")
	var statusErr *apierrors.StatusError
	ok := errors.As(err, &statusErr)
	Expect(ok).To(BeTrue(), "error should be a status error")
	Expect(statusErr.ErrStatus.Reason).To(Equal(metav1.StatusReasonForbidden), "the error should be a status forbidden error")
	Expect(statusErr.ErrStatus.Message).To(
		ContainSubstring("spec.clusterName: Required value: the clusterName must be set"),
		"the error message should reflect that the clusterName must be set",
	)
}

var _ = Describe("Validate Plugin with OwnerReference from PluginPresets", func() {
	var testPlugin = &greenhousev1alpha1.Plugin{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Plugin",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-plugin",
			Namespace: test.TestNamespace,
		},
		Spec: greenhousev1alpha1.PluginSpec{
			PluginDefinition: "test-plugindefinition",
			ClusterName:      testclustername,
		},
	}

	var ownerReference = metav1.OwnerReference{
		APIVersion: "greenhouse.cloud.sap/v1alpha1",
		Kind:       "PluginPreset",
		Name:       "test-preset",
		Controller: ptr.To(false),
	}

	It("should return a warning if the Plugin has an OwnerReference from a PluginPreset", func() {
		cut := testPlugin.DeepCopy()
		cut.SetOwnerReferences([]metav1.OwnerReference{ownerReference})
		warnings := validateOwnerReference(cut)
		Expect(warnings).NotTo(BeNil(), "expected a warning, got nil")
	})
	It("should return no warning if the Plugin has no OwnerReference from a PluginPreset", func() {
		warnings := validateOwnerReference(testPlugin)
		Expect(warnings).To(BeNil(), "expected no warning, got %v", warnings)
	})
})
