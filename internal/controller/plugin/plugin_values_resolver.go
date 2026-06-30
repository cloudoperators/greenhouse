// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

const (
	// trackingSeparator is used to separate multiple tracking IDs in annotations
	trackingSeparator = ";"
)

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
