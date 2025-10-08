// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var _ = Describe("Validate Plugin OptionValues", func() {
	DescribeTable("Validate PluginType contains either Value or ValueFrom", func(value *apiextensionsv1.JSON, valueFrom *greenhousev1alpha1.ValueFromSource, expErr bool) {
		optionValues := []greenhousev1alpha1.PluginOptionValue{
			{
				Name:      "test",
				Value:     value,
				ValueFrom: valueFrom,
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

		optionsFieldPath := field.NewPath("spec").Child("optionValues")
		errList := validatePluginOptionValues(optionValues, pluginDefinition.Name, pluginDefinition.Spec, true, optionsFieldPath)
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

		optionValues := []greenhousev1alpha1.PluginOptionValue{
			{
				Name:  "test",
				Value: test.MustReturnJSONFor(actValue),
			},
		}

		optionsFieldPath := field.NewPath("spec").Child("optionValues")
		errList := validatePluginOptionValues(optionValues, pluginDefinition.Name, pluginDefinition.Spec, true, optionsFieldPath)
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
		Entry("PluginOption Value Consistent With PluginOption Type Secret", "", greenhousev1alpha1.PluginOptionTypeSecret, "vault+kvv2:///some-path/to/secret", false),
		Entry("PluginOption Value Inconsistent With PluginOption Type Secret", "", greenhousev1alpha1.PluginOptionTypeSecret, "some-string", true),
	)

	DescribeTable("Validate PluginOptionValue references a Secret", func(actValue *greenhousev1alpha1.PluginOptionValue, expErr bool) {
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

		optionValues := []greenhousev1alpha1.PluginOptionValue{
			*actValue,
		}

		optionsFieldPath := field.NewPath("spec").Child("optionValues")
		errList := validatePluginOptionValues(optionValues, pluginDefinition.Name, pluginDefinition.Spec, true, optionsFieldPath)
		switch expErr {
		case true:
			Expect(errList).ToNot(BeEmpty(), "expected an error, got nil")
		default:
			Expect(errList).To(BeEmpty(), "expected no error, got %v", errList)
		}
	},
		Entry("PluginOption ValueFrom has a valid SecretReference", &greenhousev1alpha1.PluginOptionValue{Name: "test", ValueFrom: &greenhousev1alpha1.ValueFromSource{Secret: &greenhousev1alpha1.SecretKeyReference{Name: "secret", Key: "key"}}}, false),
		Entry("PluginOption Value has a valid string with vault schema prefix", &greenhousev1alpha1.PluginOptionValue{Name: "test", Value: test.MustReturnJSONFor("vault+kvv2:///some-path/to/secret")}, false),
		Entry("PluginOption Value has a invalid string", &greenhousev1alpha1.PluginOptionValue{Name: "test", Value: test.MustReturnJSONFor("some-string")}, true),
		Entry("PluginOption ValueFrom is missing SecretReference Name", &greenhousev1alpha1.PluginOptionValue{Name: "test", ValueFrom: &greenhousev1alpha1.ValueFromSource{Secret: &greenhousev1alpha1.SecretKeyReference{Key: "key"}}}, true),
		Entry("PluginOption ValueFrom is missing SecretReference Key", &greenhousev1alpha1.PluginOptionValue{Name: "test", ValueFrom: &greenhousev1alpha1.ValueFromSource{Secret: &greenhousev1alpha1.SecretKeyReference{Name: "secret"}}}, true),
		Entry("PluginOption ValueFrom does not contain a SecretReference", &greenhousev1alpha1.PluginOptionValue{Name: "test"}, true),
	)

	Describe("Validate Plugin specifies all required options", func() {
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
		It("should reject a Plugin with missing required options", func() {
			plugin := test.NewPlugin(test.Ctx, "test-plugin", test.TestNamespace,
				test.WithClusterPluginDefinition("test"),
				test.WithCluster("test-cluster"),
			)
			optionsFieldPath := field.NewPath("spec").Child("optionValues")
			errList := validatePluginOptionValues(plugin.Spec.OptionValues, pluginDefinition.Name, pluginDefinition.Spec, true, optionsFieldPath)
			Expect(errList).NotTo(BeEmpty(), "expected an error, got nil")
		})
		It("should accept a Plugin with supplied required options", func() {
			optionValues := []greenhousev1alpha1.PluginOptionValue{
				{
					Name:  "test",
					Value: test.MustReturnJSONFor("test"),
				},
			}
			optionsFieldPath := field.NewPath("spec").Child("optionValues")
			errList := validatePluginOptionValues(optionValues, pluginDefinition.Name, pluginDefinition.Spec, true, optionsFieldPath)
			Expect(errList).To(BeEmpty(), "unexpected error")
		})
	})
})

var _ = Describe("Validate plugin spec fields", Ordered, func() {
	var (
		setup *test.TestSetup

		team                             *greenhousev1alpha1.Team
		testCluster                      *greenhousev1alpha1.Cluster
		testPlugin                       *greenhousev1alpha1.Plugin
		testClusterPluginDefinition      *greenhousev1alpha1.ClusterPluginDefinition
		testCentralPluginDefinition      *greenhousev1alpha1.ClusterPluginDefinition
		testUIAppClusterPluginDefinition *greenhousev1alpha1.ClusterPluginDefinition
		testPluginDefinition             *greenhousev1alpha1.PluginDefinition
	)

	BeforeAll(func() {
		setup = test.NewTestSetup(test.Ctx, test.K8sClient, "plugin-webhook")
		team = setup.CreateTeam(test.Ctx, "test-team", test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))
		testCluster = setup.CreateCluster(test.Ctx, "test-cluster", test.WithClusterLabel(greenhouseapis.LabelKeyOwnedBy, team.Name))
		testClusterPluginDefinition = setup.CreateClusterPluginDefinition(test.Ctx, "test-plugindefinition")
		testCentralPluginDefinition = setup.CreateClusterPluginDefinition(test.Ctx, "central-plugin")
		testUIAppClusterPluginDefinition = setup.CreateClusterPluginDefinition(test.Ctx, "uiapp-plugindefinition",
			test.WithVersion("1.0.0"),
			test.WithoutHelmChart(),
			test.WithUIApplication(&greenhousev1alpha1.UIApplicationReference{
				Name:    "test-ui-app",
				Version: "0.0.1",
			}),
		)
		testPluginDefinition = setup.CreatePluginDefinition(test.Ctx, "pd-namespaced")
		pluginsAllowedInCentralCluster = append(pluginsAllowedInCentralCluster, testCentralPluginDefinition.Name)
	})

	AfterEach(func() {
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testPlugin)
	})

	AfterAll(func() {
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testCluster)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, team)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testClusterPluginDefinition)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testCentralPluginDefinition)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testUIAppClusterPluginDefinition)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testPluginDefinition)
	})

	It("should not accept a plugin without a clusterName", func() {
		testPlugin = test.NewPlugin(test.Ctx, "test-plugin", setup.Namespace(),
			test.WithClusterPluginDefinition(testClusterPluginDefinition.Name),
			test.WithReleaseNamespace("test-namespace"),
			test.WithReleaseName("test-release"),
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, team.Name))
		expectClusterMustBeSetError(test.K8sClient.Create(test.Ctx, testPlugin))
	})

	It("should not accept a plugin without a plugindefinition", func() {
		testPlugin = test.NewPlugin(test.Ctx, "test-plugin", setup.Namespace(),
			test.WithCluster(testCluster.Name),
			test.WithReleaseNamespace("test-namespace"),
			test.WithReleaseName("test-release"),
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, team.Name))
		err := test.K8sClient.Create(test.Ctx, testPlugin)
		expectPluginDefinitionMustMatchError(err)
	})

	It("should not accept a plugin for the central cluster where releaseNamespace and Plugin Namespace do not match", func() {
		testPlugin = test.NewPlugin(test.Ctx, "test-plugin", setup.Namespace(),
			test.WithClusterPluginDefinition(testCentralPluginDefinition.Name),
			test.WithReleaseNamespace("test-namespace"),
			test.WithReleaseName("test-release"),
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, team.Name))
		expectReleaseNamespaceMustMatchError(test.K8sClient.Create(test.Ctx, testPlugin))
	})

	It("should accept a plugin for a remote cluster where releaseNamespace and Plugin Namespace do not match and the PluginDefinition is allowed on the central cluster", func() {
		tempPluginsAllowedInCentralCluster := pluginsAllowedInCentralCluster
		defer func() {
			pluginsAllowedInCentralCluster = tempPluginsAllowedInCentralCluster
		}()
		pluginsAllowedInCentralCluster = []string{testClusterPluginDefinition.Name}
		testPlugin = test.NewPlugin(test.Ctx, "test-plugin", setup.Namespace(),
			test.WithClusterPluginDefinition(testClusterPluginDefinition.Name),
			test.WithCluster(testCluster.Name),
			test.WithReleaseNamespace("test-namespace"),
			test.WithReleaseName("test-release"),
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, team.Name))
		err := test.K8sClient.Create(test.Ctx, testPlugin)
		Expect(err).ToNot(HaveOccurred(), "there should be no error creating the plugin")
	})

	It("should not accept a plugin if the releaseNamespace changes", func() {
		testPlugin = setup.CreatePlugin(test.Ctx, "test-plugin",
			test.WithClusterPluginDefinition(testClusterPluginDefinition.Name),
			test.WithCluster(testCluster.Name),
			test.WithReleaseNamespace("test-namespace"),
			test.WithReleaseName("test-release"),
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, team.Name))

		By("updating the plugin with a different releaseNamespace")
		testPlugin.Spec.ReleaseNamespace = "new-namespace"
		err := test.K8sClient.Update(test.Ctx, testPlugin)
		Expect(err).To(HaveOccurred(), "there should be an error updating the plugin")
	})

	It("should reject the plugin when the cluster with clusterName does not exist", func() {
		By("creating the plugin")
		testPlugin = test.NewPlugin(test.Ctx, "test-plugin", setup.Namespace(),
			test.WithClusterPluginDefinition(testClusterPluginDefinition.Name),
			test.WithCluster("non-existent-cluster"),
			test.WithReleaseNamespace("test-namespace"),
			test.WithReleaseName("test-release"),
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, team.Name))

		expectClusterNotFoundError(test.K8sClient.Create(test.Ctx, testPlugin))
	})

	It("should keep the template field when merging option values", func() {
		By("creating the plugin")
		testPlugin = setup.CreatePlugin(test.Ctx, "test-plugin",
			test.WithPluginDefinition(testPluginDefinition.Name),
			test.WithCluster(testCluster.Name),
			test.WithReleaseNamespace("test-namespace"),
			test.WithReleaseName("test-release"),
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
			test.WithPluginOptionValueTemplate("templateOption", ptr.To("{{ .global.greenhouse.clusterName }}")))

		By("checking that the label is kept after merging options and optionvalues")
		actPlugin := &greenhousev1alpha1.Plugin{}
		err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: testPlugin.Name, Namespace: testPlugin.Namespace}, actPlugin)
		Expect(err).ToNot(HaveOccurred(), "there should be no error getting the plugin")
		Eventually(func() bool {
			for _, actOption := range actPlugin.Spec.OptionValues {
				if actOption.Name == "templateOption" {
					return actOption.Template != nil
				}
			}
			return false
		}).Should(BeTrue(), "the plugin should have the template field set on the optionValue")
	})

	It("should accept the plugin with reference to ClusterPluginDefinition", func() {
		By("creating the plugin")
		testPlugin = setup.CreatePlugin(test.Ctx, "test-plugin",
			test.WithClusterPluginDefinition(testClusterPluginDefinition.Name),
			test.WithCluster(testCluster.Name),
			test.WithReleaseNamespace("test-namespace"),
			test.WithReleaseName("test-release"),
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, team.Name))

		By("checking the label on the plugin")
		actPlugin := &greenhousev1alpha1.Plugin{}
		err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: testPlugin.Name, Namespace: testPlugin.Namespace}, actPlugin)
		Expect(err).ToNot(HaveOccurred(), "there should be no error getting the plugin")
		Eventually(func() map[string]string {
			return actPlugin.GetLabels()
		}).Should(HaveKeyWithValue(greenhouseapis.LabelKeyCluster, testCluster.Name), "the plugin should have a matching cluster label")
	})

	It("should accept the plugin with reference to namespaced PluginDefinition", func() {
		By("creating the plugin")
		testPlugin = setup.CreatePlugin(test.Ctx, "test-plugin",
			test.WithPluginDefinition(testPluginDefinition.Name),
			test.WithCluster(testCluster.Name),
			test.WithReleaseNamespace("test-namespace"),
			test.WithReleaseName("test-release"),
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, team.Name))

		By("checking the label on the plugin")
		actPlugin := &greenhousev1alpha1.Plugin{}
		err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: testPlugin.Name, Namespace: testPlugin.Namespace}, actPlugin)
		Expect(err).ToNot(HaveOccurred(), "there should be no error getting the plugin")
		Eventually(func() map[string]string {
			return actPlugin.GetLabels()
		}).Should(HaveKeyWithValue(greenhouseapis.LabelKeyCluster, testCluster.Name), "the plugin should have a matching cluster label")
	})

	It("should reject to update a plugin when the clusterName changes", func() {
		testPlugin = setup.CreatePlugin(test.Ctx, "test-plugin",
			test.WithClusterPluginDefinition(testClusterPluginDefinition.Name),
			test.WithCluster(testCluster.Name),
			test.WithReleaseNamespace("test-namespace"),
			test.WithReleaseName("test-release"),
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, team.Name))
		testPlugin.Spec.ClusterName = "wrong-cluster-name"
		err := test.K8sClient.Update(test.Ctx, testPlugin)

		Expect(err).To(HaveOccurred(), "there should be an error changing the plugin's clusterName")
		Expect(err.Error()).To(ContainSubstring(validation.FieldImmutableErrorMsg))
	})

	It("should reject to update a plugin when the clustername is removed", func() {
		testPlugin = setup.CreatePlugin(test.Ctx, "test-plugin",
			test.WithClusterPluginDefinition(testClusterPluginDefinition.Name),
			test.WithCluster(testCluster.Name),
			test.WithReleaseNamespace("test-namespace"),
			test.WithReleaseName("test-release"),
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, team.Name))
		testPlugin.Spec.ClusterName = ""
		err := test.K8sClient.Update(test.Ctx, testPlugin)
		Expect(err).To(HaveOccurred(), "there should be an error changing the plugin's clusterName")
		Expect(err.Error()).To(ContainSubstring(validation.FieldImmutableErrorMsg))
	})

	It("should reject to update a plugin when the releaseNamespace changes", func() {
		testPlugin = setup.CreatePlugin(test.Ctx, "test-plugin",
			test.WithClusterPluginDefinition(testClusterPluginDefinition.Name),
			test.WithCluster(testCluster.Name),
			test.WithReleaseNamespace("test-namespace"),
			test.WithReleaseName("test-release"),
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, team.Name))
		testPlugin.Spec.ReleaseNamespace = "foo-bar"
		err := test.K8sClient.Update(test.Ctx, testPlugin)
		Expect(err).To(HaveOccurred(), "there should be an error changing the plugin's releaseNamespace")
		Expect(err.Error()).To(ContainSubstring(validation.FieldImmutableErrorMsg))
	})

	It("should accept to update a UI-only plugin when ReleaseNamespace is changed", func() {
		testPlugin = setup.CreatePlugin(test.Ctx, "test-plugin",
			test.WithClusterPluginDefinition(testUIAppClusterPluginDefinition.Name),
			test.WithCluster(testCluster.Name),
			test.WithReleaseNamespace("test-namespace"),
			test.WithReleaseName("test-release"),
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, team.Name))
		testPlugin.Spec.ReleaseNamespace = ""
		err := test.K8sClient.Update(test.Ctx, testPlugin)
		Expect(err).ToNot(HaveOccurred(), "there should be no error changing the UI-only plugin's releaseNamespace")
	})

	It("should reject to update a plugin when the pluginDefinition reference name changes", func() {
		secondClusterPluginDefinition := setup.CreateClusterPluginDefinition(test.Ctx, "foo-bar")
		testPlugin = setup.CreatePlugin(test.Ctx, "test-plugin",
			test.WithClusterPluginDefinition(testClusterPluginDefinition.Name),
			test.WithCluster(testCluster.Name),
			test.WithReleaseNamespace("test-namespace"),
			test.WithReleaseName("test-release"),
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, team.Name))

		testPlugin.Spec.PluginDefinitionRef.Name = secondClusterPluginDefinition.Name
		err := test.K8sClient.Update(test.Ctx, testPlugin)
		Expect(err).To(HaveOccurred(), "there should be an error changing the plugin's pluginDefinition name")
		Expect(err.Error()).To(ContainSubstring(validation.FieldImmutableErrorMsg))

		test.EventuallyDeleted(test.Ctx, test.K8sClient, secondClusterPluginDefinition)
	})

	It("should accept to update a plugin when the pluginDefinition reference kind changes to PluginDefinition", func() {
		By("creating ClusterPluginDefinition with the same name as namespaced PluginDefinition")
		secondClusterPluginDefinition := test.NewClusterPluginDefinition(test.Ctx, testPluginDefinition.Name)
		Expect(setup.Create(test.Ctx, secondClusterPluginDefinition)).To(Succeed(), "failed to create ClusterPluginDefinition")

		testPlugin = setup.CreatePlugin(test.Ctx, "test-plugin",
			test.WithClusterPluginDefinition(secondClusterPluginDefinition.Name),
			test.WithCluster(testCluster.Name),
			test.WithReleaseNamespace("test-namespace"),
			test.WithReleaseName("test-release"),
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, team.Name))

		testPlugin.Spec.PluginDefinitionRef.Kind = greenhousev1alpha1.PluginDefinitionKind
		err := test.K8sClient.Update(test.Ctx, testPlugin)
		Expect(err).ToNot(HaveOccurred(), "there should be no error changing the plugin's pluginDefinition kind")
		labels := testPlugin.GetLabels()
		Expect(labels).To(HaveKeyWithValue(greenhouseapis.LabelKeyPluginDefinition, testPluginDefinition.Name),
			"the plugin should have the plugindefinition label set to the pluginDefinition name")
		Expect(labels).ToNot(HaveKey(greenhouseapis.LabelKeyClusterPluginDefinition),
			"the plugin should not have the clusterplugindefinition label set")

		test.EventuallyDeleted(test.Ctx, test.K8sClient, secondClusterPluginDefinition)
	})

	It("should accept to update a plugin when the pluginDefinition reference kind changes to ClusterPluginDefinition", func() {
		By("creating PluginDefinition with the same name as ClusterPluginDefinition")
		secondPluginDefinition := test.NewPluginDefinition(test.Ctx, testClusterPluginDefinition.Name, setup.Namespace())
		Expect(setup.Create(test.Ctx, secondPluginDefinition)).To(Succeed(), "failed to create PluginDefinition")

		testPlugin = setup.CreatePlugin(test.Ctx, "test-plugin",
			test.WithPluginDefinition(secondPluginDefinition.Name),
			test.WithCluster(testCluster.Name),
			test.WithReleaseNamespace("test-namespace"),
			test.WithReleaseName("test-release"),
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, team.Name))

		testPlugin.Spec.PluginDefinitionRef.Kind = greenhousev1alpha1.ClusterPluginDefinitionKind
		err := test.K8sClient.Update(test.Ctx, testPlugin)
		Expect(err).ToNot(HaveOccurred(), "there should be no error changing the plugin's pluginDefinition kind")
		labels := testPlugin.GetLabels()
		Expect(labels).To(HaveKeyWithValue(greenhouseapis.LabelKeyClusterPluginDefinition, testClusterPluginDefinition.Name),
			"the plugin should have the plugindefinition label set to the pluginDefinition name")
		Expect(labels).ToNot(HaveKey(greenhouseapis.LabelKeyPluginDefinition),
			"the plugin should not have the clusterplugindefinition label set")

		test.EventuallyDeleted(test.Ctx, test.K8sClient, secondPluginDefinition)
	})

	It("should correctly default the PluginDefinitionKind", func() {
		testPlugin := test.NewPlugin(test.Ctx, "test-plugin", setup.Namespace(),
			test.WithCluster(testCluster.Name),
			test.WithReleaseNamespace("test-namespace"),
			test.WithReleaseName("test-release"),
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
		)
		//nolint:staticcheck
		testPlugin.Spec.PluginDefinition = testPluginDefinition.Name

		Expect(test.K8sClient.Create(test.Ctx, testPlugin)).To(Succeed(), "there must be no error creating the Plugin")
		Expect(testPlugin.Spec.PluginDefinitionRef.Kind).To(Equal(greenhousev1alpha1.PluginDefinitionKind), "PluginDefinitionKind should be defaulted to PluginDefinition")
	})
})

var _ = Describe("Validate PluginDefinition labels on Defaulting", Ordered, func() {
	var (
		setup *test.TestSetup

		team                        *greenhousev1alpha1.Team
		testCluster                 *greenhousev1alpha1.Cluster
		testPlugin                  *greenhousev1alpha1.Plugin
		testClusterPluginDefinition *greenhousev1alpha1.ClusterPluginDefinition
		testCentralPluginDefinition *greenhousev1alpha1.ClusterPluginDefinition
		testPluginDefinition        *greenhousev1alpha1.PluginDefinition
	)

	BeforeAll(func() {
		setup = test.NewTestSetup(test.Ctx, test.K8sClient, "plugin-webhook")
		team = setup.CreateTeam(test.Ctx, "test-team", test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))
		testCluster = setup.CreateCluster(test.Ctx, "test-cluster", test.WithClusterLabel(greenhouseapis.LabelKeyOwnedBy, team.Name))
		testClusterPluginDefinition = setup.CreateClusterPluginDefinition(test.Ctx, "test-clusterplugindefinition")
		testCentralPluginDefinition = setup.CreateClusterPluginDefinition(test.Ctx, "central-plugin")
		testPluginDefinition = setup.CreatePluginDefinition(test.Ctx, "test-plugindefinition")
		pluginsAllowedInCentralCluster = append(pluginsAllowedInCentralCluster, testCentralPluginDefinition.Name)
	})

	AfterEach(func() {
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testPlugin)
	})

	AfterAll(func() {
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testCluster)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, team)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testClusterPluginDefinition)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testCentralPluginDefinition)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testPluginDefinition)
	})

	It("should accept a plugin for a remote cluster where releaseNamespace and Plugin Namespace do not match and the ClusterPluginDefinition is allowed on the central cluster", func() {
		tempPluginsAllowedInCentralCluster := pluginsAllowedInCentralCluster
		defer func() {
			pluginsAllowedInCentralCluster = tempPluginsAllowedInCentralCluster
		}()
		pluginsAllowedInCentralCluster = []string{testClusterPluginDefinition.Name}
		testPlugin = test.NewPlugin(test.Ctx, "test-plugin", setup.Namespace(),
			test.WithClusterPluginDefinition(testClusterPluginDefinition.Name),
			test.WithCluster(testCluster.Name),
			test.WithReleaseNamespace("test-namespace"),
			test.WithReleaseName("test-release"),
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
			test.WithPluginLabel(greenhouseapis.LabelKeyPluginDefinition, testClusterPluginDefinition.Name),
		)
		err := test.K8sClient.Create(test.Ctx, testPlugin)
		Expect(err).ToNot(HaveOccurred(), "there should be no error creating the plugin")

		By("checking the label on the plugin")
		Eventually(func(g Gomega) {
			err = test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: testPlugin.Name, Namespace: testPlugin.Namespace}, testPlugin)
			Expect(err).ToNot(HaveOccurred(), "there should be no error getting the plugin")
			labels := testPlugin.GetLabels()
			g.Expect(labels).To(HaveKeyWithValue(greenhouseapis.LabelKeyClusterPluginDefinition, testClusterPluginDefinition.Name),
				"the plugin should have the clusterplugindefinition label set to the pluginDefinition name")
			g.Expect(labels).ToNot(HaveKey(greenhouseapis.LabelKeyPluginDefinition),
				"the plugin should not have the plugindefinition label set")
		}).Should(Succeed(), "the plugin should have only clusterplugindefinition label set")
	})

	It("should only set the plugindefinition label when referencing namespaced PluginDefinition", func() {
		testPlugin = test.NewPlugin(test.Ctx, "test-plugin", setup.Namespace(),
			test.WithPluginDefinition(testPluginDefinition.Name),
			test.WithCluster(testCluster.Name),
			test.WithReleaseNamespace("test-namespace"),
			test.WithReleaseName("test-release"),
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
			test.WithPluginLabel(greenhouseapis.LabelKeyClusterPluginDefinition, testPluginDefinition.Name),
		)
		err := test.K8sClient.Create(test.Ctx, testPlugin)
		Expect(err).ToNot(HaveOccurred(), "there should be no error creating the plugin")

		By("checking the label on the plugin")
		Eventually(func(g Gomega) {
			err = test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: testPlugin.Name, Namespace: testPlugin.Namespace}, testPlugin)
			Expect(err).ToNot(HaveOccurred(), "there should be no error getting the plugin")
			labels := testPlugin.GetLabels()
			g.Expect(labels).To(HaveKeyWithValue(greenhouseapis.LabelKeyPluginDefinition, testPluginDefinition.Name),
				"the plugin should have the plugindefinition label set to the pluginDefinition name")
			g.Expect(labels).ToNot(HaveKey(greenhouseapis.LabelKeyClusterPluginDefinition),
				"the plugin should not have the clusterplugindefinition label set")
		}).Should(Succeed(), "the plugin should have only plugindefinition label set")
	})
})

func expectClusterNotFoundError(err error) {
	GinkgoHelper()
	Expect(err).To(HaveOccurred(), "there should be an error updating the plugin")
	var statusErr *apierrors.StatusError
	ok := errors.As(err, &statusErr)
	Expect(ok).To(BeTrue(), "error should be a status error")
	Expect(statusErr.ErrStatus.Reason).To(Equal(metav1.StatusReasonForbidden), "the error should be a status forbidden error")
	Expect(statusErr.ErrStatus.Message).To(ContainSubstring("spec.clusterName: Not found"), "the error message should reflect clustername not found")
}

func expectClusterMustBeSetError(err error) {
	GinkgoHelper()
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

func expectPluginDefinitionMustMatchError(err error) {
	GinkgoHelper()
	Expect(err).To(HaveOccurred(), "there should be an error creating/updating the plugin")
	var statusErr *apierrors.StatusError
	ok := errors.As(err, &statusErr)
	Expect(ok).To(BeTrue(), "error should be a status error")
	Expect(statusErr.ErrStatus.Reason).To(Equal(metav1.StatusReasonForbidden), "the error should be a status forbidden error")
	Expect(err.Error()).To(And(ContainSubstring("spec.pluginDefinitionRef.name"), ContainSubstring("PluginDefinition name must be set")), "the error message should reflect that the pluginDefinition name must be set")
}

func expectReleaseNamespaceMustMatchError(err error) {
	GinkgoHelper()
	Expect(err).To(HaveOccurred(), "there should be an error creating/updating the plugin")
	var statusErr *apierrors.StatusError
	ok := errors.As(err, &statusErr)
	Expect(ok).To(BeTrue(), "error should be a status error")
	Expect(statusErr.ErrStatus.Reason).To(Equal(metav1.StatusReasonForbidden), "the error should be a status forbidden error")
	Expect(statusErr.ErrStatus.Message).To(
		ContainSubstring("central cluster can only be deployed in the same namespace as the plugin"),
		"the error message should reflect that the releaseNamespace must be the same as the plugin namespace for a plugin in the central cluster",
	)
}

var _ = Describe("Validate Plugin with OwnerReference from PluginPresets", func() {
	testPlugin := test.NewPlugin(test.Ctx, "test-plugin", test.TestNamespace,
		test.WithClusterPluginDefinition("test-plugindefinition"),
		test.WithCluster("test-cluster"),
	)

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

var _ = Describe("Validation and defaulting of releaseName", func() {
	var (
		testPlugin       *greenhousev1alpha1.Plugin
		pluginDefinition *greenhousev1alpha1.ClusterPluginDefinition
	)

	BeforeEach(func() {
		pluginDefinition = test.NewClusterPluginDefinition(test.Ctx, "test-definition", test.WithHelmChart(&greenhousev1alpha1.HelmChartReference{Name: "test-helm-chart"}))

		testPlugin = test.NewPlugin(test.Ctx, "test-plugin", "testing", test.WithClusterPluginDefinition("test-definition"))
		// ensure the Plugin is in the deployed state
		testPlugin.Status.HelmReleaseStatus = &greenhousev1alpha1.HelmReleaseStatus{
			Status: "deployed",
		}
	})

	It("should default releaseName to the Plugin Name if previously deployed", func() {
		fakeClient := fake.NewClientBuilder().WithScheme(test.GreenhouseV1Alpha1Scheme()).WithObjects(pluginDefinition).Build()
		// Call the DefaultPlugin function
		err := DefaultPlugin(context.TODO(), fakeClient, testPlugin)
		Expect(err).ToNot(HaveOccurred(), "there should be no error defaulting the plugin")

		// Assert that the releaseName is set to the existing Helm release name
		Expect(testPlugin.Spec.ReleaseName).To(Equal(testPlugin.Name), "releaseName should be defaulted to the Plugin name")
	})
	It("should default releaseName to the Helm chart name from the PluginDefinition if not set and no Helm release exists", func() {
		fakeClient := fake.NewClientBuilder().WithScheme(test.GreenhouseV1Alpha1Scheme()).WithObjects(pluginDefinition).Build()
		testPlugin.Status.HelmReleaseStatus = nil // No existing Helm release
		testPlugin.Spec.ReleaseName = ""          // ReleaseName not set

		// Call the DefaultPlugin function
		err := DefaultPlugin(context.TODO(), fakeClient, testPlugin)
		Expect(err).ToNot(HaveOccurred(), "there should be no error defaulting the plugin")

		// Assert that the releaseName is set to the Helm chart name
		Expect(testPlugin.Spec.ReleaseName).To(Equal("test-helm-chart"), "releaseName should be defaulted to the Helm chart name from the PluginDefinition")
	})

	It("should not allow arbitrary releaseName if Plugin already deployed", func() {
		fakeClient := fake.NewClientBuilder().WithScheme(test.GreenhouseV1Alpha1Scheme()).WithObjects(testPlugin, pluginDefinition).Build()
		cut := testPlugin.DeepCopy()
		cut.Spec.ReleaseName = "arbitrary-release-name"
		_, err := ValidateUpdatePlugin(test.Ctx, fakeClient, testPlugin, cut)
		Expect(err).To(HaveOccurred(), "there should be an error updating the plugin")
	})
})
