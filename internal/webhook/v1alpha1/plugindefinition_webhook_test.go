// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var _ = DescribeTable("Validate PluginOption Type and Value are consistent", func(expectedType greenhousev1alpha1.PluginOptionType, defaultValue any, expErr bool) {

	clusterPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{
		Spec: greenhousev1alpha1.PluginDefinitionSpec{
			Options: []greenhousev1alpha1.PluginOption{
				{
					Name:    "test",
					Default: test.MustReturnJSONFor(defaultValue),
					Type:    expectedType,
				},
			},
		},
	}
	actErr := validatePluginDefinitionOptionValueAndType(clusterPluginDefinition.Spec, clusterPluginDefinition.GroupVersionKind(), clusterPluginDefinition.GetName())
	switch expErr {
	case false:
		Expect(actErr).ToNot(HaveOccurred(), "unexpected error occurred")
	default:
		var err *apierrors.StatusError
		Expect(errors.As(actErr, &err)).To(BeTrue(), "expected an *apierrors.StatusError, got %T", actErr)

		Expect(err.ErrStatus.Reason).To(Equal(metav1.StatusReasonInvalid), "expected an error with reason %s, got %s", metav1.StatusReasonInvalid, err.ErrStatus)
	}
},
	Entry("PluginOptionTypeBool Consistent", greenhousev1alpha1.PluginOptionTypeBool, true, false),
	Entry("PluginOptionTypeBool Inconsistent", greenhousev1alpha1.PluginOptionTypeBool, "notabool", true),
	Entry("PluginOptionTypeString Consistent", greenhousev1alpha1.PluginOptionTypeString, "string", nil),
	Entry("PluginOptionTypeString Consistent Integer as String", greenhousev1alpha1.PluginOptionTypeString, "1", false),
	Entry("PluginOptionTypeString Inconsistent", greenhousev1alpha1.PluginOptionTypeString, 1, true),
	Entry("PluginOptionTypeInt Consistent", greenhousev1alpha1.PluginOptionTypeInt, 1, false),
	Entry("PluginOptionTypeInt Inconsistent", greenhousev1alpha1.PluginOptionTypeInt, "one", true),
	Entry("PluginOptionTypeList Consistent", greenhousev1alpha1.PluginOptionTypeList, []string{"one", "two"}, false),
	Entry("PluginOptionTypeList Inconsistent", greenhousev1alpha1.PluginOptionTypeList, "one", true),
	Entry("PluginOptionTypeMap Consistent", greenhousev1alpha1.PluginOptionTypeMap, map[string]any{"key": "value"}, false),
	Entry("PluginOptionTypeMap Inconsistent", greenhousev1alpha1.PluginOptionTypeMap, "one", true),
	Entry("PluginOptionTypeMap Consistent Nested Map", greenhousev1alpha1.PluginOptionTypeMap, map[string]any{"key": map[string]any{"nestedKey": "value"}}, false),
	Entry("PluginOptionTypeSecret Consistent Nil", greenhousev1alpha1.PluginOptionTypeSecret, nil, false),
	Entry("PluginOptionTypeSecret Consistent Empty", greenhousev1alpha1.PluginOptionTypeSecret, "", false),
	Entry("PluginOptionTypeSecret Defaulted", greenhousev1alpha1.PluginOptionTypeSecret, "secret", true),
	Entry("PluginOptionTypeSecret Inconsistent", greenhousev1alpha1.PluginOptionTypeSecret, []string{"one", "two"}, true),
)

var _ = Describe("Validate (Cluster-)PluginDefinition Creation", func() {
	It("should deny creation of ClusterPluginDefinition with defaulted Secret OptionValue", func() {
		clusterPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{
			Spec: greenhousev1alpha1.PluginDefinitionSpec{
				Version: "1.0.0",
				UIApplication: &greenhousev1alpha1.UIApplicationReference{
					Name:    "test-ui",
					Version: "1.0.0",
				},
				Options: []greenhousev1alpha1.PluginOption{
					{
						Name:    "test-secret",
						Default: test.MustReturnJSONFor("some-secret"),
						Type:    greenhousev1alpha1.PluginOptionTypeSecret,
					},
				},
			},
		}

		c := fake.NewClientBuilder().WithScheme(test.GreenhouseV1Alpha1Scheme()).Build()

		_, err := ValidateCreateClusterPluginDefinition(context.TODO(), c, clusterPluginDefinition)
		Expect(err).To(HaveOccurred(), "there should be an error creating the ClusterPluginDefinition")
		Expect(err.Error()).To(ContainSubstring("defaults are not allowed in PluginOptions of the 'Secret' type"))
	})
	It("should deny creation of ClusterPluginDefinition without spec.Version", func() {
		clusterPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{
			Spec: greenhousev1alpha1.PluginDefinitionSpec{
				UIApplication: &greenhousev1alpha1.UIApplicationReference{
					Name: "test-no-version",
				},
				Options: []greenhousev1alpha1.PluginOption{},
			},
		}

		c := fake.NewClientBuilder().WithScheme(test.GreenhouseV1Alpha1Scheme()).Build()

		_, err := ValidateCreateClusterPluginDefinition(context.TODO(), c, clusterPluginDefinition)
		Expect(err).To(HaveOccurred(), "there should be an error creating the ClusterPluginDefinition")
		Expect(err.Error()).To(ContainSubstring("PluginDefinition without spec.version is invalid."))
	})

	It("should deny creation of PluginDefinition with defaulted Secret OptionValue", func() {
		pluginDefinition := &greenhousev1alpha1.PluginDefinition{
			Spec: greenhousev1alpha1.PluginDefinitionSpec{
				Version: "1.0.0",
				UIApplication: &greenhousev1alpha1.UIApplicationReference{
					Name:    "test-ui",
					Version: "1.0.0",
				},
				Options: []greenhousev1alpha1.PluginOption{
					{
						Name:    "test-secret",
						Default: test.MustReturnJSONFor("some-secret"),
						Type:    greenhousev1alpha1.PluginOptionTypeSecret,
					},
				},
			},
		}

		c := fake.NewClientBuilder().WithScheme(test.GreenhouseV1Alpha1Scheme()).Build()

		_, err := ValidateCreatePluginDefinition(context.TODO(), c, pluginDefinition)
		Expect(err).To(HaveOccurred(), "there should be an error creating the PluginDefinition")
		Expect(err.Error()).To(ContainSubstring("defaults are not allowed in PluginOptions of the 'Secret' type"))
	})
	It("should deny creation of PluginDefinition without spec.Version", func() {
		pluginDefinition := &greenhousev1alpha1.PluginDefinition{
			Spec: greenhousev1alpha1.PluginDefinitionSpec{
				UIApplication: &greenhousev1alpha1.UIApplicationReference{
					Name: "test-no-version",
				},
				Options: []greenhousev1alpha1.PluginOption{},
			},
		}

		c := fake.NewClientBuilder().WithScheme(test.GreenhouseV1Alpha1Scheme()).Build()

		_, err := ValidateCreatePluginDefinition(context.TODO(), c, pluginDefinition)
		Expect(err).To(HaveOccurred(), "there should be an error creating the PluginDefinition")
		Expect(err.Error()).To(ContainSubstring("PluginDefinition without spec.version is invalid."))
	})
})

var _ = Describe("Validate (Cluster-)PluginDefinition Update", func() {
	It("should deny updating ClusterPluginDefinition with defaulted Secret OptionValue", func() {
		clusterPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{
			Spec: greenhousev1alpha1.PluginDefinitionSpec{
				Version: "1.0.0",
				UIApplication: &greenhousev1alpha1.UIApplicationReference{
					Name:    "test-ui",
					Version: "1.0.0",
				},
				Options: []greenhousev1alpha1.PluginOption{
					{
						Name:    "test-secret",
						Default: test.MustReturnJSONFor("some-secret"),
						Type:    greenhousev1alpha1.PluginOptionTypeSecret,
					},
				},
			},
		}

		c := fake.NewClientBuilder().WithScheme(test.GreenhouseV1Alpha1Scheme()).Build()

		_, err := ValidateUpdateClusterPluginDefinition(context.TODO(), c, nil, clusterPluginDefinition)
		Expect(err).To(HaveOccurred(), "there should be an error updating the ClusterPluginDefinition")
		Expect(err.Error()).To(ContainSubstring("defaults are not allowed in PluginOptions of the 'Secret' type"))
	})
	It("should deny updating ClusterPluginDefinition without spec.Version", func() {
		clusterPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{
			Spec: greenhousev1alpha1.PluginDefinitionSpec{
				UIApplication: &greenhousev1alpha1.UIApplicationReference{
					Name: "test-no-version",
				},
				Options: []greenhousev1alpha1.PluginOption{},
			},
		}

		c := fake.NewClientBuilder().WithScheme(test.GreenhouseV1Alpha1Scheme()).Build()

		_, err := ValidateUpdateClusterPluginDefinition(context.TODO(), c, nil, clusterPluginDefinition)
		Expect(err).To(HaveOccurred(), "there should be an error updating the ClusterPluginDefinition")
		Expect(err.Error()).To(ContainSubstring("PluginDefinition without spec.version is invalid."))
	})

	It("should deny updating PluginDefinition with defaulted Secret OptionValue", func() {
		pluginDefinition := &greenhousev1alpha1.PluginDefinition{
			Spec: greenhousev1alpha1.PluginDefinitionSpec{
				Version: "1.0.0",
				UIApplication: &greenhousev1alpha1.UIApplicationReference{
					Name:    "test-ui",
					Version: "1.0.0",
				},
				Options: []greenhousev1alpha1.PluginOption{
					{
						Name:    "test-secret",
						Default: test.MustReturnJSONFor("some-secret"),
						Type:    greenhousev1alpha1.PluginOptionTypeSecret,
					},
				},
			},
		}

		c := fake.NewClientBuilder().WithScheme(test.GreenhouseV1Alpha1Scheme()).Build()

		_, err := ValidateUpdatePluginDefinition(context.TODO(), c, nil, pluginDefinition)
		Expect(err).To(HaveOccurred(), "there should be an error updating the PluginDefinition")
		Expect(err.Error()).To(ContainSubstring("defaults are not allowed in PluginOptions of the 'Secret' type"))
	})
	It("should deny updating PluginDefinition without spec.Version", func() {
		pluginDefinition := &greenhousev1alpha1.PluginDefinition{
			Spec: greenhousev1alpha1.PluginDefinitionSpec{
				UIApplication: &greenhousev1alpha1.UIApplicationReference{
					Name: "test-no-version",
				},
				Options: []greenhousev1alpha1.PluginOption{},
			},
		}

		c := fake.NewClientBuilder().WithScheme(test.GreenhouseV1Alpha1Scheme()).Build()

		_, err := ValidateUpdatePluginDefinition(context.TODO(), c, nil, pluginDefinition)

		Expect(err).To(HaveOccurred(), "there should be an error updating the PluginDefinition")
		Expect(err.Error()).To(ContainSubstring("PluginDefinition without spec.version is invalid."))
	})
})

var _ = Describe("Validate (Cluster-)PluginDefinition Deletion", func() {
	It("should allow deletion of ClusterPluginDefinition without Plugin", func() {
		clusterPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
		}
		pluginList := &greenhousev1alpha1.PluginList{}

		c := fake.NewClientBuilder().WithScheme(test.GreenhouseV1Alpha1Scheme()).WithLists(pluginList).Build()

		_, err := ValidateDeleteClusterPluginDefinition(context.TODO(), c, clusterPluginDefinition)
		Expect(err).ToNot(HaveOccurred(), "there should be no error deleting the ClusterPluginDefinition")
	})
	It("should prevent deletion of ClusterPluginDefinition with Plugin", func() {
		clusterPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
		}
		pluginList := &greenhousev1alpha1.PluginList{
			Items: []greenhousev1alpha1.Plugin{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-plugin",
						Namespace: "default",
						Labels: map[string]string{
							greenhouseapis.LabelKeyClusterPluginDefinition: "test",
						},
					},
				},
			},
		}

		c := fake.NewClientBuilder().WithScheme(test.GreenhouseV1Alpha1Scheme()).WithLists(pluginList).Build()

		_, err := ValidateDeleteClusterPluginDefinition(context.TODO(), c, clusterPluginDefinition)
		Expect(err).To(HaveOccurred(), "there should be an error deleting the ClusterPluginDefinition when Plugins still exist")
	})

	It("should allow deletion of PluginDefinition without Plugin", func() {
		pluginDefinition := &greenhousev1alpha1.PluginDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
			},
		}
		pluginList := &greenhousev1alpha1.PluginList{}

		c := fake.NewClientBuilder().WithScheme(test.GreenhouseV1Alpha1Scheme()).WithLists(pluginList).Build()

		_, err := ValidateDeletePluginDefinition(context.TODO(), c, pluginDefinition)
		Expect(err).ToNot(HaveOccurred(), "there should be no error deleting the PluginDefinition")
	})
	It("should allow deletion of PluginDefinition with Plugin in a different namespace", func() {
		pluginDefinition := &greenhousev1alpha1.PluginDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "testns",
			},
		}
		pluginList := &greenhousev1alpha1.PluginList{
			Items: []greenhousev1alpha1.Plugin{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-plugin",
						Namespace: "default",
						Labels: map[string]string{
							greenhouseapis.LabelKeyPluginDefinition: "test",
						},
					},
				},
			},
		}

		c := fake.NewClientBuilder().WithScheme(test.GreenhouseV1Alpha1Scheme()).WithLists(pluginList).Build()

		_, err := ValidateDeletePluginDefinition(context.TODO(), c, pluginDefinition)
		Expect(err).ToNot(HaveOccurred(), "there should be no error deleting the PluginDefinition")
	})
	It("should prevent deletion of PluginDefinition with Plugin in the same namespace", func() {
		pluginDefinition := &greenhousev1alpha1.PluginDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
			},
		}
		pluginList := &greenhousev1alpha1.PluginList{
			Items: []greenhousev1alpha1.Plugin{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-plugin",
						Namespace: "default",
						Labels: map[string]string{
							greenhouseapis.LabelKeyPluginDefinition: "test",
						},
					},
				},
			},
		}

		c := fake.NewClientBuilder().WithScheme(test.GreenhouseV1Alpha1Scheme()).WithLists(pluginList).Build()

		_, err := ValidateDeletePluginDefinition(context.TODO(), c, pluginDefinition)
		Expect(err).To(HaveOccurred(), "there should be an error deleting the PluginDefinition when Plugins still exist")
	})
})
