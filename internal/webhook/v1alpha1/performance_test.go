//go:build perf
// +build perf

// Can be run with: ginkgo -tags=perf

package v1alpha1_test

import (
	"context"
	"encoding/json"
	"fmt"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/test"
	"github.com/cloudoperators/greenhouse/internal/webhook"
	"helm.sh/helm/v3/pkg/chartutil"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/strings/slices"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega/gmeasure"
)

var _ = Describe("Webhook Performance Tests", func() {
	var (
		setup   *test.TestSetup
		team    *greenhousev1alpha1.Team
		cluster *greenhousev1alpha1.Cluster
	)

	BeforeEach(func() {
		By("creating a new test setup")
		setup = test.NewTestSetup(test.Ctx, test.K8sClient, "webhook-performance-test")
		By("creating a support-group Team")
		team = setup.CreateTeam(test.Ctx, "test-team", test.WithSupportGroupLabel("true"))
		By("creating a Cluster")
		cluster = setup.CreateCluster(test.Ctx, "test-cluster", test.WithClusterOwnedByLabelValue(team.Name))
	})

	AfterEach(func() {
		By("cleaning up the Cluster")
		test.EventuallyDeleted(test.Ctx, setup.Client, cluster)
		By("deleting the test Team")
		test.EventuallyDeleted(test.Ctx, setup.Client, team)
	})

	It("should compare validateCreateCluster_WithoutOwnerLabelCheck and validateCreateCluster_WithOwnerLabelCheck performance", func() {
		experiment := *gmeasure.NewExperiment("ValidateUpdateCluster performance")
		AddReportEntry(experiment.Name, experiment)

		experiment.SampleDuration("validateCreateCluster_WithoutOwnerLabelCheck", func(_ int) {
			_, _ = validateCreateCluster_WithoutOwnerLabelCheck(test.Ctx, setup.Client, cluster)
		}, gmeasure.SamplingConfig{N: 10000})
		experiment.SampleDuration("validateCreateCluster_WithOwnerLabelCheck", func(_ int) {
			_, _ = validateCreateCluster_WithOwnerLabelCheck(test.Ctx, setup.Client, cluster)
		}, gmeasure.SamplingConfig{N: 10000})

		withoutOwnerLabelCheckStats := experiment.GetStats("validateCreateCluster_WithoutOwnerLabelCheck")
		withOwnerLabelCheckStats := experiment.GetStats("validateCreateCluster_WithOwnerLabelCheck")

		ranking := gmeasure.RankStats(gmeasure.LowerMedianIsBetter, withoutOwnerLabelCheckStats, withOwnerLabelCheckStats)
		AddReportEntry("Ranking", ranking)
	})

	It("should compare validateCreatePlugin_WithoutOwnerLabelCheck and validateCreatePlugin_WithOwnerLabelCheck performance", func() {
		experiment := *gmeasure.NewExperiment("ValidateCreatePlugin performance")
		AddReportEntry(experiment.Name, experiment)

		By("creating a PluginDefinition")
		testPluginDefinition := setup.CreatePluginDefinition(test.Ctx, "test-plugindefinition")
		By("creating a Plugin")
		plugin := setup.CreatePlugin(test.Ctx, "test-plugin",
			test.WithPluginDefinition(testPluginDefinition.Name),
			test.WithCluster(cluster.Name),
			test.WithReleaseNamespace("test-namespace"),
			test.WithReleaseName("test-release"),
			test.WithPluginOwnedByLabelValue(team.Name))

		experiment.SampleDuration("validateCreatePlugin_WithoutOwnerLabelCheck", func(_ int) {
			_, _ = validateCreatePlugin_WithoutOwnerLabelCheck(test.Ctx, setup.Client, plugin)
		}, gmeasure.SamplingConfig{N: 10000})
		experiment.SampleDuration("validateCreatePlugin_WithOwnerLabelCheck", func(_ int) {
			_, _ = validateCreatePlugin_WithOwnerLabelCheck(test.Ctx, setup.Client, plugin)
		}, gmeasure.SamplingConfig{N: 10000})

		withoutOwnerLabelCheckStats := experiment.GetStats("validateCreatePlugin_WithoutOwnerLabelCheck")
		withOwnerLabelCheckStats := experiment.GetStats("validateCreatePlugin_WithOwnerLabelCheck")

		ranking := gmeasure.RankStats(gmeasure.LowerMedianIsBetter, withoutOwnerLabelCheckStats, withOwnerLabelCheckStats)
		AddReportEntry("Ranking", ranking)

		By("cleaning up the Plugin")
		test.EventuallyDeleted(test.Ctx, setup.Client, plugin)
	})
})

func validateCreateCluster_WithoutOwnerLabelCheck(ctx context.Context, _ client.Client, obj runtime.Object) (admission.Warnings, error) {
	logger := ctrl.LoggerFrom(ctx)
	cluster, ok := obj.(*greenhousev1alpha1.Cluster)
	if !ok {
		return nil, nil
	}
	if err := webhook.InvalidateDoubleDashesInName(cluster, logger); err != nil {
		return nil, err
	}
	// capping the name at 40 chars, so we ensure to get unique urls for exposed services per cluster. service-name/namespace hash needs to fit (max 63 chars)
	if err := webhook.CapName(cluster, logger, 40); err != nil {
		return nil, err
	}
	annotations := cluster.GetAnnotations()
	_, deletionMarked := annotations[greenhouseapis.MarkClusterDeletionAnnotation]
	_, scheduleExists := annotations[greenhouseapis.ScheduleClusterDeletionAnnotation]
	if deletionMarked || scheduleExists {
		err := apierrors.NewInvalid(cluster.GroupVersionKind().GroupKind(), cluster.GetName(), nil)
		logger.Error(err, "found deletion annotation on cluster creation, admission will be denied")
		return admission.Warnings{"you cannot create a cluster with deletion annotation"}, err
	}

	return nil, nil
}

func validateCreateCluster_WithOwnerLabelCheck(ctx context.Context, c client.Client, obj runtime.Object) (admission.Warnings, error) {
	logger := ctrl.LoggerFrom(ctx)
	cluster, ok := obj.(*greenhousev1alpha1.Cluster)
	if !ok {
		return nil, nil
	}
	if err := webhook.InvalidateDoubleDashesInName(cluster, logger); err != nil {
		return nil, err
	}
	// capping the name at 40 chars, so we ensure to get unique urls for exposed services per cluster. service-name/namespace hash needs to fit (max 63 chars)
	if err := webhook.CapName(cluster, logger, 40); err != nil {
		return nil, err
	}
	annotations := cluster.GetAnnotations()
	_, deletionMarked := annotations[greenhouseapis.MarkClusterDeletionAnnotation]
	_, scheduleExists := annotations[greenhouseapis.ScheduleClusterDeletionAnnotation]
	if deletionMarked || scheduleExists {
		err := apierrors.NewInvalid(cluster.GroupVersionKind().GroupKind(), cluster.GetName(), nil)
		logger.Error(err, "found deletion annotation on cluster creation, admission will be denied")
		return admission.Warnings{"you cannot create a cluster with deletion annotation"}, err
	}

	labelValidationWarning := webhook.ValidateLabelOwnedBy(ctx, c, cluster)
	if labelValidationWarning != "" {
		return admission.Warnings{"Cluster should have a support-group Team set as its owner", labelValidationWarning}, nil
	}

	return nil, nil
}

func validateCreatePlugin_WithoutOwnerLabelCheck(ctx context.Context, c client.Client, obj runtime.Object) (admission.Warnings, error) {
	plugin, ok := obj.(*greenhousev1alpha1.Plugin)
	if !ok {
		return nil, nil
	}

	pluginDefinition := new(greenhousev1alpha1.PluginDefinition)
	err := c.Get(ctx, client.ObjectKey{Namespace: "", Name: plugin.Spec.PluginDefinition}, pluginDefinition)
	if err != nil {
		// TODO: provide actual APIError
		return nil, err
	}

	if err := validateReleaseName(plugin.Spec.ReleaseName); err != nil {
		return nil, field.Invalid(field.NewPath("spec").Child("releaseName"), plugin.Spec.ReleaseName, err.Error())
	}

	optionsFieldPath := field.NewPath("spec").Child("optionValues")
	errList := validatePluginOptionValues(plugin.Spec.OptionValues, pluginDefinition, true, optionsFieldPath)
	if len(errList) > 0 {
		return nil, apierrors.NewInvalid(plugin.GroupVersionKind().GroupKind(), plugin.Name, errList)
	}
	if err := validatePluginForCluster(ctx, c, plugin, pluginDefinition); err != nil {
		return nil, err
	}

	return nil, nil
}

func validateCreatePlugin_WithOwnerLabelCheck(ctx context.Context, c client.Client, obj runtime.Object) (admission.Warnings, error) {
	plugin, ok := obj.(*greenhousev1alpha1.Plugin)
	if !ok {
		return nil, nil
	}

	pluginDefinition := new(greenhousev1alpha1.PluginDefinition)
	err := c.Get(ctx, client.ObjectKey{Namespace: "", Name: plugin.Spec.PluginDefinition}, pluginDefinition)
	if err != nil {
		// TODO: provide actual APIError
		return nil, err
	}

	if err := validateReleaseName(plugin.Spec.ReleaseName); err != nil {
		return nil, field.Invalid(field.NewPath("spec").Child("releaseName"), plugin.Spec.ReleaseName, err.Error())
	}

	optionsFieldPath := field.NewPath("spec").Child("optionValues")
	errList := validatePluginOptionValues(plugin.Spec.OptionValues, pluginDefinition, true, optionsFieldPath)
	if len(errList) > 0 {
		return nil, apierrors.NewInvalid(plugin.GroupVersionKind().GroupKind(), plugin.Name, errList)
	}
	if err := validatePluginForCluster(ctx, c, plugin, pluginDefinition); err != nil {
		return nil, err
	}

	labelValidationWarning := webhook.ValidateLabelOwnedBy(ctx, c, plugin)
	if labelValidationWarning != "" {
		return admission.Warnings{"Plugin should have a support-group Team set as its owner", labelValidationWarning}, nil
	}
	return nil, nil
}

// validateReleaseName checks if the release name is valid according to Helm's rules.
func validateReleaseName(name string) error {
	if name == "" {
		return nil
	}
	return chartutil.ValidateReleaseName(name)
}

func validatePluginForCluster(ctx context.Context, c client.Client, plugin *greenhousev1alpha1.Plugin, pluginDefinition *greenhousev1alpha1.PluginDefinition) error {
	// Exclude front-end only Plugins as well as the greenhouse namespace from the below check.
	if pluginDefinition.Spec.HelmChart == nil || plugin.GetNamespace() == "greenhouse" {
		return nil
	}
	// Ensure whitelisted plugins are deployed in the organization namespace
	if slices.Contains(pluginsAllowedInCentralCluster, plugin.Spec.PluginDefinition) {
		if plugin.Spec.ReleaseNamespace != plugin.GetNamespace() {
			return field.Forbidden(field.NewPath("spec").Child("releaseNamespace"), "plugins running in the central cluster can only be deployed in the same namespace as the plugin")
		}
		return nil
	}

	// If the PluginDefinition is not allowed in the central cluster, the Plugin must have a spec.clusterName set.
	clusterName := plugin.Spec.ClusterName
	if clusterName == "" {
		return field.Required(field.NewPath("spec").Child("clusterName"), "the clusterName must be set")
	}
	// Verify that the cluster exists.
	var cluster = new(greenhousev1alpha1.Cluster)
	if err := c.Get(ctx, types.NamespacedName{Namespace: plugin.Namespace, Name: clusterName}, cluster); err != nil {
		switch {
		case apierrors.IsNotFound(err):
			return field.NotFound(field.NewPath("spec").Child("clusterName"), clusterName)
		default:
			return field.InternalError(field.NewPath("spec").Child("clusterName"), err)
		}
	}
	return nil
}

func validatePluginOptionValues(
	optionValues []greenhousev1alpha1.PluginOptionValue,
	pluginDefinition *greenhousev1alpha1.PluginDefinition,
	checkRequiredOptions bool,
	optionsFieldPath *field.Path,
) field.ErrorList {

	var allErrs field.ErrorList
	var isOptionValueSet bool
	for _, pluginOption := range pluginDefinition.Spec.Options {
		isOptionValueSet = false
		for idx, val := range optionValues {
			if pluginOption.Name != val.Name {
				continue
			}
			// If the option is required, it must be set.
			isOptionValueSet = true
			fieldPathWithIndex := optionsFieldPath.Index(idx)

			// Value and ValueFrom are mutually exclusive, but one must be provided.
			if (val.Value == nil && val.ValueFrom == nil) || (val.Value != nil && val.ValueFrom != nil) {
				allErrs = append(allErrs, field.Required(
					fieldPathWithIndex,
					"must provide either value or valueFrom for value "+val.Name,
				))
				continue
			}

			// Validate that OptionValue has a secret reference.
			if pluginOption.Type == greenhousev1alpha1.PluginOptionTypeSecret {
				switch {
				case val.Value != nil:
					allErrs = append(allErrs, field.TypeInvalid(fieldPathWithIndex.Child("value"), "*****",
						fmt.Sprintf("optionValue %s of type secret must use valueFrom to reference a secret", val.Name)))
					continue
				case val.ValueFrom != nil:
					if val.ValueFrom.Secret.Name == "" {
						allErrs = append(allErrs, field.Required(fieldPathWithIndex.Child("valueFrom").Child("name"),
							fmt.Sprintf("optionValue %s of type secret must reference a secret by name", val.Name)))
						continue
					}
					if val.ValueFrom.Secret.Key == "" {
						allErrs = append(allErrs, field.Required(fieldPathWithIndex.Child("valueFrom").Child("key"),
							fmt.Sprintf("optionValue %s of type secret must reference a key in a secret", val.Name)))
						continue
					}
				}
				continue
			}

			// validate that the Plugin.OptionValue matches the type of the PluginDefinition.Option
			if val.Value != nil {
				if err := pluginOption.IsValidValue(val.Value); err != nil {
					var v any
					if err := json.Unmarshal(val.Value.Raw, &v); err != nil {
						v = err
					}
					allErrs = append(allErrs, field.Invalid(
						fieldPathWithIndex.Child("value"), v, err.Error(),
					))
				}
			}
		}
		if checkRequiredOptions && pluginOption.Required && !isOptionValueSet {
			allErrs = append(allErrs, field.Required(optionsFieldPath,
				fmt.Sprintf("Option '%s' is required by PluginDefinition '%s'", pluginOption.Name, pluginDefinition.Name)))
		}
	}
	if len(allErrs) == 0 {
		return nil
	}
	return allErrs
}

var pluginsAllowedInCentralCluster = []string{
	"alerts", "doop", "heureka", "service-proxy", "teams2slack", "kubeconfig-generator", "repo-guard",
}
