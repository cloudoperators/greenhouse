// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

var _ = Describe("PluginValuesResolver Helper Functions", func() {
	Describe("parseTrackingID", func() {
		It("should parse valid tracking ID", func() {
			kind, name, err := parseTrackingID("Plugin/my-plugin")

			Expect(err).ToNot(HaveOccurred())
			Expect(kind).To(Equal("Plugin"))
			Expect(name).To(Equal("my-plugin"))
		})

		It("should parse tracking ID with special characters", func() {
			kind, name, err := parseTrackingID("PluginPreset/my-preset-123")

			Expect(err).ToNot(HaveOccurred())
			Expect(kind).To(Equal("PluginPreset"))
			Expect(name).To(Equal("my-preset-123"))
		})

		It("should return error for invalid format without separator", func() {
			_, _, err := parseTrackingID("PluginMyPlugin")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid tracking ID format"))
		})

		It("should return error for invalid format with multiple separators", func() {
			_, _, err := parseTrackingID("Plugin/My/Plugin")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid tracking ID format"))
		})

		It("should return error for empty string", func() {
			_, _, err := parseTrackingID("")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid tracking ID format"))
		})

		It("should handle tracking ID with only separator", func() {
			kind, name, err := parseTrackingID("/")

			Expect(err).ToNot(HaveOccurred())
			Expect(kind).To(Equal(""))
			Expect(name).To(Equal(""))
		})
	})

	Describe("getTrackerIDsFromAnnotations", func() {
		It("should extract single tracker ID from plugin annotations", func() {
			plugin := &greenhousev1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						greenhouseapis.AnnotationKeyPluginTackingID: "Plugin/my-plugin",
					},
				},
			}

			trackers := getTrackerIDsFromAnnotations(plugin)

			Expect(trackers).To(HaveLen(1))
			Expect(trackers[0]).To(Equal("Plugin/my-plugin"))
		})

		It("should extract multiple tracker IDs separated by semicolon", func() {
			plugin := &greenhousev1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						greenhouseapis.AnnotationKeyPluginTackingID: "Plugin/plugin-a;Plugin/plugin-b;PluginPreset/preset-c",
					},
				},
			}

			trackers := getTrackerIDsFromAnnotations(plugin)

			Expect(trackers).To(HaveLen(3))
			Expect(trackers).To(ContainElements("Plugin/plugin-a", "Plugin/plugin-b", "PluginPreset/preset-c"))
		})

		It("should return nil when plugin has no annotations", func() {
			plugin := &greenhousev1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{},
			}

			trackers := getTrackerIDsFromAnnotations(plugin)

			Expect(trackers).To(BeNil())
		})

		It("should return nil when tracking annotation is not present", func() {
			plugin := &greenhousev1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"other-annotation": "value",
					},
				},
			}

			trackers := getTrackerIDsFromAnnotations(plugin)

			Expect(trackers).To(BeNil())
		})

		It("should return nil when tracking annotation is empty", func() {
			plugin := &greenhousev1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						greenhouseapis.AnnotationKeyPluginTackingID: "",
					},
				},
			}

			trackers := getTrackerIDsFromAnnotations(plugin)

			Expect(trackers).To(BeNil())
		})
	})

	Describe("findUntrackedObjects", func() {
		It("should find objects that are no longer tracked", func() {
			previousTracked := []string{"Plugin/plugin-a", "Plugin/plugin-b", "Plugin/plugin-c"}
			currentTracked := []string{"Plugin/plugin-a", "Plugin/plugin-c"}

			untracked := findUntrackedObjects(previousTracked, currentTracked)

			Expect(untracked).To(HaveLen(1))
			Expect(untracked[0]).To(Equal("Plugin/plugin-b"))
		})

		It("should return empty slice when all previous objects are still tracked", func() {
			previousTracked := []string{"Plugin/plugin-a", "Plugin/plugin-b"}
			currentTracked := []string{"Plugin/plugin-a", "Plugin/plugin-b", "Plugin/plugin-c"}

			untracked := findUntrackedObjects(previousTracked, currentTracked)

			Expect(untracked).To(BeEmpty())
		})

		It("should return all previous objects when current is empty", func() {
			previousTracked := []string{"Plugin/plugin-a", "Plugin/plugin-b"}
			var currentTracked []string

			untracked := findUntrackedObjects(previousTracked, currentTracked)

			Expect(untracked).To(HaveLen(2))
			Expect(untracked).To(ContainElements("Plugin/plugin-a", "Plugin/plugin-b"))
		})

		It("should return empty slice when previous is empty", func() {
			var previousTracked []string
			currentTracked := []string{"Plugin/plugin-a"}

			untracked := findUntrackedObjects(previousTracked, currentTracked)

			Expect(untracked).To(BeEmpty())
		})

		It("should handle multiple untracked objects", func() {
			previousTracked := []string{"Plugin/a", "Plugin/b", "Plugin/c", "Plugin/d"}
			currentTracked := []string{"Plugin/a", "Plugin/d"}

			untracked := findUntrackedObjects(previousTracked, currentTracked)

			Expect(untracked).To(HaveLen(2))
			Expect(untracked).To(ContainElements("Plugin/b", "Plugin/c"))
		})
	})

	Describe("Integration: trackingID and parseTrackingID", func() {
		It("should create and parse tracking ID correctly", func() {
			originalKind := "Plugin"
			originalName := "my-plugin"

			// Create tracking ID
			id := trackingID(originalKind, originalName)

			// Parse it back
			parsedKind, parsedName, err := parseTrackingID(id)

			Expect(err).ToNot(HaveOccurred())
			Expect(parsedKind).To(Equal(originalKind))
			Expect(parsedName).To(Equal(originalName))
		})

		It("should handle round-trip with special characters", func() {
			originalKind := "PluginPreset"
			originalName := "my-preset-v1.2.3"

			id := trackingID(originalKind, originalName)
			parsedKind, parsedName, err := parseTrackingID(id)

			Expect(err).ToNot(HaveOccurred())
			Expect(parsedKind).To(Equal(originalKind))
			Expect(parsedName).To(Equal(originalName))
		})
	})

	Describe("Integration: buildGVK with schema operations", func() {
		It("should create GVK that can be used for schema operations", func() {
			gvk := buildGVK("Plugin")

			// Verify it creates a valid GVK
			Expect(gvk).To(Equal(schema.GroupVersionKind{
				Group:   greenhousev1alpha1.GroupVersion.Group,
				Version: greenhousev1alpha1.GroupVersion.Version,
				Kind:    "Plugin",
			}))
		})
	})
})
