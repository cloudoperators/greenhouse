// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
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

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

var _ = Describe("Validate PluginConfig OptionValues", func() {
	DescribeTable("Validate PluginConfigType contains either Value or ValueFrom", func(value *apiextensionsv1.JSON, valueFrom *greenhousev1alpha1.ValueFromSource, expErr bool) {
		pluginConfig := &greenhousev1alpha1.PluginConfig{
			Spec: greenhousev1alpha1.PluginConfigSpec{
				Plugin: "test",
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

		plugin := &greenhousev1alpha1.Plugin{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "greenhouse",
				Name:      "testPlugin",
			},
			Spec: greenhousev1alpha1.PluginSpec{
				Options: []greenhousev1alpha1.PluginOption{
					{
						Name:    "test",
						Default: defaultVal,
						Type:    optionType,
					},
				},
			},
		}

		err := validatePluginConfigOptionValues(pluginConfig, plugin)
		switch expErr {
		case true:
			Expect(err).To(HaveOccurred(), "expected an error, got nil")
		default:
			Expect(err).ToNot(HaveOccurred(), "expected no error, got %v", err)
		}
	},
		Entry("Value and ValueFrom nil", nil, nil, true),
		Entry("Value and ValueFrom not nil", test.MustReturnJSONFor("test"), &greenhousev1alpha1.ValueFromSource{Secret: &greenhousev1alpha1.SecretKeyReference{Name: "my-secret"}}, true),
		Entry("Value not nil", test.MustReturnJSONFor("test"), nil, false),
		Entry("ValueFrom not nil", nil, &greenhousev1alpha1.ValueFromSource{Secret: &greenhousev1alpha1.SecretKeyReference{Name: "my-secret", Key: "secret-key"}}, false),
	)

	DescribeTable("Validate PluginOptionValue is consistent with PluginOption Type", func(defaultValue any, defaultType greenhousev1alpha1.PluginOptionType, actValue any, expErr bool) {

		plugin := &greenhousev1alpha1.Plugin{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "greenhouse",
				Name:      "testPlugin",
			},
			Spec: greenhousev1alpha1.PluginSpec{
				Options: []greenhousev1alpha1.PluginOption{
					{
						Name:    "test",
						Default: test.MustReturnJSONFor(defaultValue),
						Type:    defaultType,
					},
				},
			},
		}

		pluginConfig := &greenhousev1alpha1.PluginConfig{
			Spec: greenhousev1alpha1.PluginConfigSpec{
				Plugin: "test",
				OptionValues: []greenhousev1alpha1.PluginOptionValue{
					{
						Name:  "test",
						Value: test.MustReturnJSONFor(actValue),
					},
				},
			},
		}

		err := validatePluginConfigOptionValues(pluginConfig, plugin)
		switch expErr {
		case true:
			Expect(err).To(HaveOccurred(), "expected an error, got nil")
		default:
			Expect(err).ToNot(HaveOccurred(), "expected no error, got %v", err)
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
		plugin := &greenhousev1alpha1.Plugin{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "greenhouse",
				Name:      "testPlugin",
			},
			Spec: greenhousev1alpha1.PluginSpec{
				Options: []greenhousev1alpha1.PluginOption{
					{
						Name: "test",
						Type: greenhousev1alpha1.PluginOptionTypeSecret,
					},
				},
			},
		}

		pluginConfig := &greenhousev1alpha1.PluginConfig{
			Spec: greenhousev1alpha1.PluginConfigSpec{
				Plugin: "test",
				OptionValues: []greenhousev1alpha1.PluginOptionValue{
					{
						Name:      "test",
						ValueFrom: actValue,
					},
				},
			},
		}

		err := validatePluginConfigOptionValues(pluginConfig, plugin)
		switch expErr {
		case true:
			Expect(err).To(HaveOccurred(), "expected an error, got nil")
		default:
			Expect(err).ToNot(HaveOccurred(), "expected no error, got %v", err)
		}
	},
		Entry("PluginOption ValueFrom has a valid SecretReference", &greenhousev1alpha1.ValueFromSource{Secret: &greenhousev1alpha1.SecretKeyReference{Name: "secret", Key: "key"}}, false),
		Entry("PluginOption ValueFrom is missing SecretReference Name", &greenhousev1alpha1.ValueFromSource{Secret: &greenhousev1alpha1.SecretKeyReference{Key: "key"}}, true),
		Entry("PluginOption ValueFrom is missing SecretReference Key", &greenhousev1alpha1.ValueFromSource{Secret: &greenhousev1alpha1.SecretKeyReference{Name: "secret"}}, true),
		Entry("PluginOption ValueFrom does not contain a SecretReference", nil, true),
	)

	Describe("Validate PluginConfig specifies all required options", func() {
		plugin := &greenhousev1alpha1.Plugin{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "greenhouse",
				Name:      "testPlugin",
			},
			Spec: greenhousev1alpha1.PluginSpec{
				Options: []greenhousev1alpha1.PluginOption{
					{
						Name:     "test",
						Type:     greenhousev1alpha1.PluginOptionTypeString,
						Required: true,
					},
				},
			},
		}
		It("should reject a PluginConfig with missing required options", func() {
			pluginConfig := &greenhousev1alpha1.PluginConfig{
				TypeMeta: metav1.TypeMeta{
					Kind:       "PluginConfig",
					APIVersion: greenhousev1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-plugin-config",
					Namespace: test.TestNamespace,
				},
				Spec: greenhousev1alpha1.PluginConfigSpec{
					Plugin: "test",
				},
			}
			err := validatePluginConfigOptionValues(pluginConfig, plugin)
			Expect(err).To(HaveOccurred(), "expected an error, got nil")
		})
		It("should accept a PluginConfig with supplied required options", func() {
			pluginConfig := &greenhousev1alpha1.PluginConfig{
				TypeMeta: metav1.TypeMeta{
					Kind:       "PluginConfig",
					APIVersion: greenhousev1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-plugin-config",
					Namespace: test.TestNamespace,
				},
				Spec: greenhousev1alpha1.PluginConfigSpec{
					Plugin: "test",
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "test",
							Value: test.MustReturnJSONFor("test"),
						},
					},
				},
			}
			err := validatePluginConfigOptionValues(pluginConfig, plugin)
			Expect(err).ToNot(HaveOccurred(), "unexpected error")
		})
	})
})

var testPluginConfig = &greenhousev1alpha1.PluginConfig{
	TypeMeta: metav1.TypeMeta{
		Kind:       "PluginConfig",
		APIVersion: greenhousev1alpha1.GroupVersion.String(),
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-plugin-config",
		Namespace: test.TestNamespace,
	},
	Spec: greenhousev1alpha1.PluginConfigSpec{
		Plugin: "test-plugin",
	},
}

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
		Description: "Testplugin",
		Version:     "1.0.0",
		HelmChart: &greenhousev1alpha1.HelmChartReference{
			Name:       "./../../test/fixtures/myChart",
			Repository: "dummy",
			Version:    "1.0.0",
		},
	},
}

var testCluster = &greenhousev1alpha1.Cluster{
	TypeMeta: metav1.TypeMeta{
		Kind:       "Cluster",
		APIVersion: greenhousev1alpha1.GroupVersion.String(),
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-cluster",
		Namespace: test.TestNamespace,
	},
	Spec: greenhousev1alpha1.ClusterSpec{
		AccessMode: greenhousev1alpha1.ClusterAccessModeDirect,
	},
}

var _ = Describe("Validate pluginConfig clusterName", Ordered, func() {

	BeforeAll(func() {
		err := test.K8sClient.Create(test.Ctx, testPlugin)
		Expect(err).ToNot(HaveOccurred(), "there should be no error creating the plugin")

	})

	AfterAll(func() {
		err := test.K8sClient.Delete(test.Ctx, testPlugin)
		Expect(err).ToNot(HaveOccurred(), "there should be no error deleting the plugin")
	})

	It("should accept a pluginConfig without a clusterName", func() {

		err := test.K8sClient.Create(test.Ctx, testPluginConfig)
		Expect(err).ToNot(HaveOccurred(), "there should be no error creating the pluginConfig")

		err = test.K8sClient.Delete(test.Ctx, testPluginConfig)
		Expect(err).ToNot(HaveOccurred(), "there should be no error deleting the pluginConfig")
	})

	It("should reject the pluginConfig when the cluster with clusterName does not exist", func() {
		By("creating the pluginConfig")
		testPluginConfig.Spec.ClusterName = "test-cluster"
		err := test.K8sClient.Create(test.Ctx, testPluginConfig)
		expectClusterNotFoundError(err)
	})

	It("should accept the pluginConfig when the cluster with clusterName exists", func() {
		By("creating the cluster")
		err := test.K8sClient.Create(test.Ctx, testCluster)
		Expect(err).ToNot(HaveOccurred(), "there should be no error creating the cluster")

		By("creating the pluginConfig")
		//reset resourceVersion to avoid conflict, still using same struct
		testPluginConfig.ResourceVersion = ""
		err = test.K8sClient.Create(test.Ctx, testPluginConfig)
		Expect(err).ToNot(HaveOccurred(), "there should be no error creating the pluginConfig")

		By("checking the label on the pluginConfig")
		actPluginConfig := &greenhousev1alpha1.PluginConfig{}
		err = test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: testPluginConfig.Name, Namespace: testPluginConfig.Namespace}, actPluginConfig)
		Expect(err).ToNot(HaveOccurred(), "there should be no error getting the pluginConfig")
		Eventually(func() map[string]string {
			return actPluginConfig.GetLabels()
		}).Should(HaveKeyWithValue(greenhouseapis.LabelKeyCluster, "test-cluster"), "the pluginConfig should have a matching cluster label")
	})

	It("should reject a pluginConfig update with a wrong cluster name reference", func() {
		testPluginConfig.Spec.ClusterName = "wrong-cluster-name"
		err := test.K8sClient.Update(test.Ctx, testPluginConfig)

		expectClusterNotFoundError(err)
	})

	It("should allow deletion of the clusterName reference in existing pluginConfig", func() {
		testPluginConfig.Spec.ClusterName = ""
		err := test.K8sClient.Update(test.Ctx, testPluginConfig)
		Expect(err).ToNot(HaveOccurred(), "there should be no error updating the pluginConfig")

		By("checking the label on the pluginConfig")
		actPluginConfig := &greenhousev1alpha1.PluginConfig{}
		err = test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: testPluginConfig.Name, Namespace: testPluginConfig.Namespace}, actPluginConfig)
		Expect(err).ToNot(HaveOccurred(), "there should be no error getting the pluginConfig")
		Eventually(func() map[string]string {
			return actPluginConfig.GetLabels()
		}).Should(HaveKeyWithValue(greenhouseapis.LabelKeyCluster, ""), "the pluginConfig should have an empty cluster label")
	})

})

func expectClusterNotFoundError(err error) {
	Expect(err).To(HaveOccurred(), "there should be an error updating the pluginConfig")
	var statusErr *apierrors.StatusError
	ok := errors.As(err, &statusErr)
	Expect(ok).To(BeTrue(), "error should be a status error")
	Expect(statusErr.ErrStatus.Reason).To(Equal(metav1.StatusReasonForbidden), "the error should be a status forbidden error")
	Expect(statusErr.ErrStatus.Message).To(ContainSubstring("spec.clusterName: Not found"), "the error message should reflect clustername not found")
}
