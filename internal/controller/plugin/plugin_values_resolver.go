// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"fmt"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

const (
	// trackingSeparator is used to separate multiple tracking IDs in annotations
	trackingSeparator = ";"
)

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
