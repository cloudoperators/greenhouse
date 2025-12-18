// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/cel"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
)

const (
	// trackingSeparator is used to separate multiple tracking IDs in annotations
	trackingSeparator = ";"
)

// filterValueRefOptions filters option values to only include those with external references (ValueFrom.Ref).
func filterValueRefOptions(optionValues []greenhousev1alpha1.PluginOptionValue) []greenhousev1alpha1.PluginOptionValue {
	return slices.DeleteFunc(optionValues, func(o greenhousev1alpha1.PluginOptionValue) bool {
		return o.ValueFrom == nil || o.ValueFrom.Ref == nil
	})
}

// ResolveValueFromRef resolves a PluginOptionValue which references other Greenhouse resources
// currently references to Plugin, PluginPreset are supported. The validation is done at CRD level.
func ResolveValueFromRef(ctx context.Context, c client.Client, plugin *greenhousev1alpha1.Plugin, option greenhousev1alpha1.PluginOptionValue) (*greenhousev1alpha1.PluginOptionValue, []string, error) {
	resolveKind := plugin.GroupVersionKind().Kind
	// current reconciling plugin as trackerID
	tracker := trackingID(resolveKind, plugin.GetName())
	resolvedValue, objectTrackers, err := resolveValueFromRef(ctx, c, plugin, option, resolveKind, tracker)
	if err != nil {
		return nil, nil, err
	}
	return resolvedValue, objectTrackers, nil
}

// resolveValueFromRef resolves option value from an external reference.
func resolveValueFromRef(ctx context.Context, c client.Client, plugin *greenhousev1alpha1.Plugin, option greenhousev1alpha1.PluginOptionValue, defaultKind, tracker string) (*greenhousev1alpha1.PluginOptionValue, []string, error) {
	var value any
	var trackedObjects []string
	var err error
	resolveKind := defaultKind
	if option.ValueFrom.Ref.Kind != "" {
		resolveKind = option.ValueFrom.Ref.Kind
	}
	gvk := buildGVK(resolveKind)
	// resolve by name
	if option.ValueFrom.Ref.Name != "" {
		value, err = resolveByName(ctx, c, plugin, option, gvk, tracker)
		if err != nil {
			return nil, nil, err
		}
		trackedObjects = append(trackedObjects, trackingID(resolveKind, option.ValueFrom.Ref.Name))
	}
	// resolve by label selector
	if option.ValueFrom.Ref.Selector != nil {
		var selectorTrackedObjects []string
		value, selectorTrackedObjects, err = resolveBySelector(ctx, c, plugin, option, gvk, tracker)
		if err != nil {
			return nil, nil, err
		}
		trackedObjects = append(trackedObjects, selectorTrackedObjects...)
	}
	if value != nil {
		byteVal, err := json.Marshal(value)
		if err != nil {
			log.FromContext(ctx).Error(err, "failed to marshal resolved value",
				"namespace", plugin.GetNamespace(),
				"name", plugin.GetName(),
				"optionName", option.Name)
			return nil, nil, err
		}
		return &greenhousev1alpha1.PluginOptionValue{
			Name:  option.Name,
			Value: &apiextensionsv1.JSON{Raw: byteVal},
		}, trackedObjects, nil
	}
	return nil, trackedObjects, nil
}

// resolveByName resolves an option value by fetching a specific named resource.
func resolveByName(ctx context.Context, c client.Client, plugin *greenhousev1alpha1.Plugin, option greenhousev1alpha1.PluginOptionValue, gvk schema.GroupVersionKind, tracker string) (any, error) {
	key := types.NamespacedName{
		Name:      option.ValueFrom.Ref.Name,
		Namespace: plugin.GetNamespace(),
	}
	uObject := &unstructured.Unstructured{}
	uObject.SetGroupVersionKind(gvk)
	err := c.Get(ctx, key, uObject)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.FromContext(ctx).Info("object does not exist, skipping value resolution...",
				"kind", gvk.Kind,
				"namespace", key.Namespace,
				"name", key.Name)
			return nil, nil
		}
		log.FromContext(ctx).Error(err, "failed to get external value",
			"kind", gvk.Kind,
			"name", key.Name,
			"namespace", key.Namespace)
		return nil, err
	}
	value, err := evaluateExpression(ctx, uObject, option.ValueFrom.Ref.Expression)
	if err != nil {
		return nil, err
	}
	// add tracking information
	if err := annotateObjectWithTracking(ctx, c, gvk, key, tracker); err != nil {
		log.FromContext(ctx).Error(err, "failed to annotate external object with tracking info, will retry on next reconciliation",
			"kind", gvk.Kind,
			"namespace", key.Namespace,
			"name", key.Name)
	}
	return value, nil
}

// resolveBySelector resolves option values by fetching resources matching a label selector.
func resolveBySelector(ctx context.Context, c client.Client, plugin *greenhousev1alpha1.Plugin, option greenhousev1alpha1.PluginOptionValue, gvk schema.GroupVersionKind, tracker string) (value any, trackedObjects []string, err error) {
	selector, err := metav1.LabelSelectorAsSelector(option.ValueFrom.Ref.Selector)
	if err != nil {
		log.FromContext(ctx).Error(err, "failed to parse label selector",
			"namespace", plugin.GetNamespace(),
			"name", plugin.GetName(),
			"optionName", option.Name)
		return nil, nil, err
	}

	value, trackedObjects, err = resolveMany(ctx, c, gvk, selector, plugin.GetNamespace(), option.ValueFrom.Ref.Expression, tracker)
	if err != nil {
		return nil, nil, err
	}

	return value, trackedObjects, nil
}

// evaluateExpression evaluates a CEL expression against a Kubernetes resource.
func evaluateExpression(ctx context.Context, uObject *unstructured.Unstructured, expression string) (any, error) {
	value, err := cel.Evaluate(expression, uObject)
	if err != nil {
		log.FromContext(ctx).Error(err, "failed to evaluate CEL expression",
			"kind", uObject.GetKind(),
			"namespace", uObject.GetNamespace(),
			"name", uObject.GetName(),
			"expression", expression)
		return nil, err
	}
	return value, nil
}

// resolveMany fetches multiple Kubernetes resources matching a label selector and evaluates
// a CEL expression against each.
func resolveMany(ctx context.Context, c client.Client, gvk schema.GroupVersionKind, selector labels.Selector, namespace, expression, tracker string) (values any, trackedObjects []string, err error) {
	// Fetch all resources matching the selector
	uObjectList := &unstructured.UnstructuredList{}
	uObjectList.SetGroupVersionKind(gvk)
	if err := c.List(
		ctx,
		uObjectList,
		client.InNamespace(namespace),
		client.MatchingLabelsSelector{Selector: selector},
	); err != nil {
		log.FromContext(ctx).Error(err, "failed to list external values",
			"kind", gvk.Kind,
			"labelSelector", selector)
		return nil, nil, err
	}
	values, trackedObjects, err = processResourceList(ctx, c, uObjectList.Items, gvk, expression, tracker)
	if err != nil {
		return nil, nil, err
	}
	return values, trackedObjects, nil
}

// processResourceList evaluates a CEL expression against each resource in a list
// and tracks all processed resources.
func processResourceList(ctx context.Context, c client.Client, items []unstructured.Unstructured, gvk schema.GroupVersionKind, expression, tracker string) (values []any, trackedObjects []string, err error) {
	for _, item := range items {
		// avoid self-referencing resources to prevent circular dependencies
		self := trackingID(gvk.Kind, item.GetName())
		if self == tracker {
			log.FromContext(ctx).Info("skipping self-referencing resource to avoid circular dependency",
				"kind", gvk.Kind,
				"namespace", item.GetNamespace(),
				"name", item.GetName())
			continue
		}
		value, err := cel.Evaluate(expression, &item)
		if err != nil {
			log.FromContext(ctx).Error(err, "failed to evaluate CEL expression",
				"kind", gvk.Kind,
				"namespace", item.GetNamespace(),
				"name", item.GetName(),
				"expression", expression)
			return nil, nil, fmt.Errorf("failed to evaluate CEL expression from object %s/%s: %w", item.GetNamespace(), item.GetName(), err)
		}
		values = appendToSlice(values, value)
		// track this resource
		trackedObjects = append(trackedObjects, trackingID(gvk.Kind, item.GetName()))
		key := types.NamespacedName{Name: item.GetName(), Namespace: item.GetNamespace()}
		if err := annotateObjectWithTracking(ctx, c, gvk, key, tracker); err != nil {
			log.FromContext(ctx).Error(err, "failed to annotate external object with tracking info, will retry on next reconciliation",
				"kind", gvk.Kind,
				"namespace", item.GetNamespace(),
				"name", item.GetName())
		}
	}
	return values, trackedObjects, nil
}

// annotateObjectWithTracking adds tracking labels and annotations to a Kubernetes resource.
// This enables dependency tracking between plugins and the resources they reference.
func annotateObjectWithTracking(ctx context.Context, c client.Client, gvk schema.GroupVersionKind, key types.NamespacedName, tracker string) error {
	uObject := &unstructured.Unstructured{}
	uObject.SetGroupVersionKind(gvk)

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := c.Get(ctx, key, uObject); err != nil {
			return err
		}
		addPluginIntegrationLabel(uObject)
		addTrackingAnnotation(uObject, tracker)
		return c.Update(ctx, uObject)
	})
}

// addPluginIntegrationLabel adds a label to indicate the resource is integrated with a plugin.
func addPluginIntegrationLabel(uObject *unstructured.Unstructured) {
	oLabels := uObject.GetLabels()
	if oLabels == nil {
		oLabels = make(map[string]string)
	}

	if _, ok := oLabels[greenhouseapis.LabelKeyPluginIntegration]; !ok {
		oLabels[greenhouseapis.LabelKeyPluginIntegration] = greenhouseapis.LabelValuePluginIntegration
		uObject.SetLabels(oLabels)
	}
}

// addTrackingAnnotation adds or updates the tracking annotation with the given tracker ID.
// tracker IDs are stored as semicolon-separated values.
func addTrackingAnnotation(uObject *unstructured.Unstructured, tracker string) {
	annotations := uObject.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	val, ok := annotations[greenhouseapis.AnnotationKeyPluginTackingID]
	if !ok || val == "" {
		// add new tracking annotation
		annotations[greenhouseapis.AnnotationKeyPluginTackingID] = tracker
		uObject.SetAnnotations(annotations)
	} else {
		// existing tracking annotation - append if not already present
		trackers := strings.Split(val, trackingSeparator)
		if !slices.Contains(trackers, tracker) {
			trackers = append(trackers, tracker)
			annotations[greenhouseapis.AnnotationKeyPluginTackingID] = strings.Join(trackers, trackingSeparator)
			uObject.SetAnnotations(annotations)
		}
	}
}

// removeUntrackedObjectAnnotations removes tracking annotations from resources that are no longer being tracked.
// This ensures that when a Plugin A changes its value references (e.g., from Plugin B to Plugin C),
// the tracking annotation is removed from the old resource (Plugin B).
// It compares the current tracked objects with the previous ones and removes the tracker ID
// from resources that are no longer in the tracked list.
func removeUntrackedObjectAnnotations(ctx context.Context, c client.Client, plugin *greenhousev1alpha1.Plugin, currentTrackedObjects []string) error {
	// previously tracked objects from plugin status
	previousTrackedObjects := plugin.Status.TrackedObjects
	if len(previousTrackedObjects) == 0 {
		// No previous tracking, nothing to clean up
		return nil
	}

	// find objects previously tracked objects
	untrackedObjects := findUntrackedObjects(previousTrackedObjects, currentTrackedObjects)
	if len(untrackedObjects) == 0 {
		// No untracked objects to clean up
		return nil
	}

	// tracker ID for reconciling plugin
	tracker := trackingID(plugin.GroupVersionKind().Kind, plugin.GetName())

	// remove tracking-id from each untracked object
	allErrors := make([]error, 0)
	for _, untrackedObjectID := range untrackedObjects {
		if err := removeTrackingAnnotation(ctx, c, plugin.GetNamespace(), untrackedObjectID, tracker); err != nil {
			log.FromContext(ctx).Error(err, "failed to remove tracking annotation from untracked object",
				"plugin", plugin.GetName(),
				"untrackedObject", untrackedObjectID)
			// continue on error the failed attempt can be retried on next reconciliation
			allErrors = append(allErrors, err)
		}
	}
	return utilerrors.NewAggregate(allErrors)
}

// findUntrackedObjects returns objects that were previously tracked but are not in the current tracked list.
func findUntrackedObjects(previousTracked, currentTracked []string) []string {
	// create a map of current tracked objects for quick lookup
	currentMap := make(map[string]bool, len(currentTracked))
	for _, obj := range currentTracked {
		currentMap[obj] = true
	}

	// find objects that are in previous but not in current
	untrackedObjects := make([]string, 0)
	for _, prevObj := range previousTracked {
		if !currentMap[prevObj] {
			untrackedObjects = append(untrackedObjects, prevObj)
		}
	}

	return untrackedObjects
}

// removeTrackingAnnotation removes a specific tracker ID from a resource's tracking annotation.
func removeTrackingAnnotation(ctx context.Context, c client.Client, namespace, objectID, tracker string) error {
	kind, name, err := parseTrackingID(objectID)
	if err != nil {
		return err
	}
	gvk := buildGVK(kind)
	key := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		uObject := &unstructured.Unstructured{}
		uObject.SetGroupVersionKind(gvk)

		if err := c.Get(ctx, key, uObject); err != nil {
			if apierrors.IsNotFound(err) {
				// Resource no longer exists, nothing to clean up
				log.FromContext(ctx).Info("untracked object not found, skipping cleanup",
					"kind", kind,
					"namespace", namespace,
					"name", name)
				return nil
			}
			return err
		}

		// current annotations
		annotations := uObject.GetAnnotations()
		if annotations == nil {
			// No annotations, nothing to remove
			return nil
		}

		// current tracking annotation value
		trackingValue, ok := annotations[greenhouseapis.AnnotationKeyPluginTackingID]
		if !ok || trackingValue == "" {
			// No tracking annotation, nothing to remove
			return nil
		}

		// spread trackers and remove specified one
		trackers := strings.Split(trackingValue, trackingSeparator)
		updatedTrackers := make([]string, 0, len(trackers))
		for _, t := range trackers {
			if t != tracker {
				updatedTrackers = append(updatedTrackers, t)
			}
		}

		switch {
		case len(updatedTrackers) == 0:
			// no trackers, remove the annotation entirely
			delete(annotations, greenhouseapis.AnnotationKeyPluginTackingID)
			log.FromContext(ctx).Info("removed tracking annotation from resource",
				"kind", kind,
				"namespace", namespace,
				"name", name,
				"tracker", tracker)

		case len(updatedTrackers) < len(trackers):
			// trackers remaining, update the annotation
			annotations[greenhouseapis.AnnotationKeyPluginTackingID] = strings.Join(updatedTrackers, trackingSeparator)
			log.FromContext(ctx).Info("removed tracker from resource",
				"kind", kind,
				"namespace", namespace,
				"name", name,
				"tracker", tracker)

		default:
			// no tracker found
			return nil
		}

		uObject.SetAnnotations(annotations)
		return c.Update(ctx, uObject)
	})
}

// buildGVK constructs a GroupVersionKind for Greenhouse resources.
func buildGVK(kind string) schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Kind:    kind,
		Version: greenhousev1alpha1.GroupVersion.Version,
		Group:   greenhousev1alpha1.GroupVersion.Group,
	}
}

// trackingID creates a unique identifier for tracking resource dependencies.
// The format is "Kind/Name" (e.g., "Plugin/my-plugin").
func trackingID(kind, name string) string {
	return kind + "/" + name
}

// parseTrackingID parses a tracking ID string into kind and name components.
// Returns an error if the format is invalid.
func parseTrackingID(trackingID string) (kind, name string, err error) {
	parts := strings.Split(trackingID, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid tracking ID format: %s", trackingID)
	}
	return parts[0], parts[1], nil
}

// getTrackerIDsFromAnnotations extracts tracker IDs from a plugin's tracking annotation.
// Returns a slice of tracker IDs, or nil if no tracking annotation exists.
func getTrackerIDsFromAnnotations(plugin *greenhousev1alpha1.Plugin) []string {
	annotations := plugin.GetAnnotations()
	if annotations == nil {
		return nil
	}

	trackerIDsStr, ok := annotations[greenhouseapis.AnnotationKeyPluginTackingID]
	if !ok || trackerIDsStr == "" {
		return nil
	}

	return strings.Split(trackerIDsStr, trackingSeparator)
}

// updateResourceWithAnnotation updates a resource with the given annotations using retry logic.
func updateResourceWithAnnotation(ctx context.Context, c client.Client, gvk schema.GroupVersionKind, key types.NamespacedName) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		uObject := &unstructured.Unstructured{}
		uObject.SetGroupVersionKind(gvk)

		if err := c.Get(ctx, key, uObject); err != nil {
			return err
		}

		annotations := uObject.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}

		// Apply the annotation update
		annotations[lifecycle.ReconcileAnnotation] = time.Now().Format(time.DateTime)

		uObject.SetAnnotations(annotations)
		return c.Update(ctx, uObject)
	})
}

// appendToSlice appends a value to a destination slice.
// CEL expressions can return slices and to avoid nested slices, this function
// flattens any slice values before appending.
func appendToSlice(dst []any, v any) []any {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Slice {
		n := rv.Len()
		// flatten slice values
		for i := range n {
			dst = append(dst, rv.Index(i).Interface())
		}
	} else {
		// Append single value
		dst = append(dst, v)
	}
	return dst
}
