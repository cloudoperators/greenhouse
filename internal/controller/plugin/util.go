// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"fmt"
	"slices"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

// exposedConditions are the conditions that are exposed in the StatusConditions of the Plugin.
var exposedConditions = []greenhousemetav1alpha1.ConditionType{
	greenhousemetav1alpha1.ReadyCondition,
	greenhousev1alpha1.ClusterAccessReadyCondition,
	greenhousev1alpha1.HelmDriftDetectedCondition,
	greenhousev1alpha1.HelmReconcileFailedCondition,
	greenhousev1alpha1.StatusUpToDateCondition,
	greenhousemetav1alpha1.OwnerLabelSetCondition,
	greenhousev1alpha1.WaitingForDependenciesCondition,
	greenhousev1alpha1.RetriesExhaustedCondition,
}

type reconcileResult struct {
	requeueAfter time.Duration
}

// InitPluginStatus initializes all empty Plugin Conditions to "unknown"
func InitPluginStatus(plugin *greenhousev1alpha1.Plugin) greenhousev1alpha1.PluginStatus {
	for _, t := range exposedConditions {
		if plugin.Status.GetConditionByType(t) == nil {
			plugin.SetCondition(greenhousemetav1alpha1.UnknownCondition(t, "", ""))
		}
	}
	if plugin.Status.HelmReleaseStatus == nil {
		plugin.Status.HelmReleaseStatus = &greenhousev1alpha1.HelmReleaseStatus{Status: "unknown"}
	}
	return plugin.Status
}

// initClientGetter returns a RestClientGetter for the given Plugin.
// If the Plugin has a clusterName set, the RestClientGetter is initialized from the cluster secret.
// Otherwise, the RestClientGetter is initialized with in-cluster config
func initClientGetter(
	ctx context.Context,
	k8sClient client.Client,
	kubeClientOpts []clientutil.KubeClientOption,
	plugin greenhousev1alpha1.Plugin,
) (genericclioptions.RESTClientGetter, error) {

	// early return if spec.clusterName is not set
	if plugin.Spec.ClusterName == "" {
		restClientGetter, err := clientutil.NewRestClientGetterForInCluster(plugin.Spec.ReleaseNamespace, kubeClientOpts...)
		if err != nil {
			errorMessage := "cannot access greenhouse cluster: " + err.Error()
			plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(
				greenhousev1alpha1.ClusterAccessReadyCondition, "", errorMessage))
			return nil, errors.New(errorMessage)
		}
		plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousev1alpha1.ClusterAccessReadyCondition, "", ""))
		return restClientGetter, nil
	}

	cluster := new(greenhousev1alpha1.Cluster)
	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: plugin.Namespace, Name: plugin.Spec.ClusterName}, cluster)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to get cluster %s: %s", plugin.Spec.ClusterName, err.Error())
		plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(
			greenhousev1alpha1.ClusterAccessReadyCondition, "", errorMessage))
		return nil, errors.New(errorMessage)
	}

	readyConditionInCluster := cluster.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
	if readyConditionInCluster == nil || readyConditionInCluster.Status != metav1.ConditionTrue {
		errorMessage := fmt.Sprintf("cluster %s is not ready", plugin.Spec.ClusterName)
		plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(
			greenhousev1alpha1.ClusterAccessReadyCondition, "", errorMessage))
		return nil, errors.New(errorMessage)
	}

	// get restclientGetter from cluster if clusterName is set
	secret := corev1.Secret{}
	err = k8sClient.Get(ctx, types.NamespacedName{Namespace: plugin.Namespace, Name: plugin.Spec.ClusterName}, &secret)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to get secret for cluster %s: %s", plugin.Spec.ClusterName, err.Error())
		plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(
			greenhousev1alpha1.ClusterAccessReadyCondition, "", errorMessage))
		return nil, errors.New(errorMessage)
	}
	restClientGetter, err := clientutil.NewRestClientGetterFromSecret(&secret, plugin.Spec.ReleaseNamespace, kubeClientOpts...)
	if err != nil {
		errorMessage := fmt.Sprintf("cannot access cluster %s: %s", plugin.Spec.ClusterName, err.Error())
		plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(
			greenhousev1alpha1.ClusterAccessReadyCondition, "", errorMessage))
		return nil, errors.New(errorMessage)
	}
	plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(
		greenhousev1alpha1.ClusterAccessReadyCondition, "", ""))
	return restClientGetter, nil
}

func getPortForExposedService(o runtime.Object) (*corev1.ServicePort, error) {
	svc, err := convertRuntimeObject[corev1.Service](o)
	if err != nil {
		return nil, err
	}

	if len(svc.Spec.Ports) == 0 {
		return nil, errors.New("service has no ports")
	}

	// Check for matching of named port set by label
	var namedPort = svc.Annotations[greenhouseapis.AnnotationKeyExposedNamedPort]

	if namedPort != "" {
		for _, port := range svc.Spec.Ports {
			if port.Name == namedPort {
				return port.DeepCopy(), nil
			}
		}
	}

	// Default to first port
	return svc.Spec.Ports[0].DeepCopy(), nil
}

func getURLForExposedIngress(o runtime.Object) (url string, err error) {
	ingress, err := convertRuntimeObject[networkingv1.Ingress](o)
	if err != nil {
		return "", err
	}

	if len(ingress.Spec.Rules) == 0 {
		return "", errors.New("ingress has no rules")
	}

	var host string
	if specificHost := ingress.Annotations[greenhouseapis.AnnotationKeyExposedIngressHost]; specificHost != "" {
		for _, rule := range ingress.Spec.Rules {
			if rule.Host == specificHost {
				host = rule.Host
				break
			}
		}
		if host == "" {
			return "", fmt.Errorf("specified host %q not found in ingress rules", specificHost)
		}
	} else {
		if ingress.Spec.Rules[0].Host == "" {
			return "", errors.New("first ingress rule has no host")
		}
		host = ingress.Spec.Rules[0].Host
	}

	protocol := "http"
	for _, tls := range ingress.Spec.TLS {
		if len(tls.Hosts) == 0 || slices.Contains(tls.Hosts, host) {
			protocol = "https"
			break
		}
	}

	return fmt.Sprintf("%s://%s", protocol, host), nil
}

func convertRuntimeObject[T any](o any) (*T, error) {
	switch obj := o.(type) {
	case *T:
		// If it's already the target type, no conversion needed
		return obj, nil
	case *unstructured.Unstructured:
		// If it's an unstructured object, convert it to the target type.
		var target T
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &target)
		return &target, errors.Wrap(err, fmt.Sprintf("failed to convert to %T from unstructured object", target))
	default:
		return nil, fmt.Errorf("unsupported runtime.Object type: %T", obj)
	}
}

func shouldReconcileOrRequeue(ctx context.Context, c client.Client, plugin *greenhousev1alpha1.Plugin) (*reconcileResult, error) {
	logger := ctrl.LoggerFrom(ctx)
	if plugin.Spec.ClusterName == "" {
		logger.Info("plugin does not have a clusterName set, will skip requeue")
		return nil, nil
	}
	cluster := &greenhousev1alpha1.Cluster{}
	err := c.Get(ctx, types.NamespacedName{Namespace: plugin.Namespace, Name: plugin.Spec.ClusterName}, cluster)
	if err != nil {
		return nil, err
	}
	scheduleExists, schedule, err := clientutil.ExtractDeletionSchedule(cluster.GetAnnotations())
	if err != nil {
		return nil, err
	}
	if scheduleExists {
		msg := fmt.Sprintf("cluster %s is scheduled for deletion at %s", plugin.Spec.ClusterName, schedule)
		plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousemetav1alpha1.DeleteCondition, lifecycle.ScheduledDeletionReason, msg))
		requeueAfter := time.Until(schedule)
		return &reconcileResult{
			requeueAfter: requeueAfter,
		}, nil
	}

	return nil, nil
}

// resolvePluginDependencies transforms the WaitFor PluginRefs so that only Plugin names are set in the output and returns flux HelmRelease dependencies.
func resolvePluginDependencies(dependencies []greenhousev1alpha1.WaitForItem, clusterName string) []helmv2.DependencyReference {
	out := make([]helmv2.DependencyReference, len(dependencies))

	for i, pluginRef := range dependencies {
		// The name of the HelmRelease is the same as the name of the Plugin.
		dependencyName := pluginRef.Name
		if pluginRef.PluginPreset != "" {
			dependencyName = buildPluginName(pluginRef.PluginPreset, clusterName)
		}
		out[i] = helmv2.DependencyReference{
			Name: dependencyName,
		}
	}

	return out
}
