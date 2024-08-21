// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"fmt"
	"strings"

	"helm.sh/helm/v3/pkg/release"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
)

// ObjectKey is a unique key for a ManifestObject.
type ObjectKey struct {
	GVK       schema.GroupVersionKind
	Namespace string
	Name      string
}

// ManifestObject represents an object in a Helm manifest.
type ManifestObject struct {
	Namespace,
	Name string
	Object runtime.Object
}

// ManifestObjectFilter is used to filter for objects in a Helm manifest.
type ManifestObjectFilter struct {
	APIVersion,
	Kind string
	Labels map[string]string
}

type ObjectList struct {
	ObjectKey
	*ManifestObject
}

type ManifestMultipleObjectFilter struct {
	Filters []ManifestObjectFilter
}

// Matches returns true if the given object matches the filters.
func (f *ManifestMultipleObjectFilter) Matches(obj *resource.Info) bool {
	for _, filter := range f.Filters {
		if filter.Matches(obj) {
			return true
		}
	}
	return false
}

// Matches returns true if the given object matches the filter.
func (o *ManifestObjectFilter) Matches(obj *resource.Info) bool {
	gvk := obj.Object.GetObjectKind().GroupVersionKind()
	if o.Kind != "" && o.Kind != gvk.Kind {
		return false
	}
	if o.APIVersion != "" && o.APIVersion != gvk.Version {
		return false
	}
	if o.Labels != nil {
		metaAccessor, err := meta.Accessor(obj.Object)
		if err != nil {
			return false
		}
		if metaAccessor.GetLabels() == nil {
			return false
		}
		for k, v := range o.Labels {
			val, ok := metaAccessor.GetLabels()[k]
			if !ok || v != val {
				return false
			}
		}
	}
	return true
}

// ObjectMapFromReleaseWithMultipleFilters returns a map of objects from the helm release manifest matching the filter or an error.
func ObjectMapFromReleaseWithMultipleFilters(restClientGetter genericclioptions.RESTClientGetter, r *release.Release, f *ManifestMultipleObjectFilter) ([]ObjectList, error) {
	return ObjectMapFromManifestWithMultipleFilters(restClientGetter, r.Namespace, r.Manifest, f)
}

// ObjectMapFromManifestWithMultipleFilters returns a map of objects from the manifests matching the filter or an error.
func ObjectMapFromManifestWithMultipleFilters(restClientGetter genericclioptions.RESTClientGetter, namespace, manifest string, f *ManifestMultipleObjectFilter) ([]ObjectList, error) {
	allObjects := ObjectList{}
	filteredObjects := []ObjectList{}
	r, err := loadManifest(restClientGetter, namespace, manifest)
	if err != nil {
		return nil, fmt.Errorf("error loading manifest: %w", err)
	}
	err = r.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}
		if f != nil {
			for _, filter := range f.Filters {
				if filter.Matches(info) {
					allObjects.ObjectKey = ObjectKey{
						GVK:       info.Mapping.GroupVersionKind,
						Namespace: info.Namespace,
						Name:      info.Name,
					}
					allObjects.ManifestObject = &ManifestObject{
						Namespace: info.Namespace,
						Name:      info.Name,
						Object:    info.Object.DeepCopyObject(),
					}
					filteredObjects = append(filteredObjects, allObjects)
				}
			}
		}
		return nil
	})
	return filteredObjects, err
}

// ObjectMapFromRelease returns a map of objects from the helm release manifest matching the filter or an error.
func ObjectMapFromRelease(restClientGetter genericclioptions.RESTClientGetter, r *release.Release, f *ManifestObjectFilter) (map[ObjectKey]*ManifestObject, error) {
	return ObjectMapFromManifest(restClientGetter, r.Namespace, r.Manifest, f)
}

// ObjectMapFromManifest returns a map of objects from the manifests matching the filter or an error.
func ObjectMapFromManifest(restClientGetter genericclioptions.RESTClientGetter, namespace, manifest string, f *ManifestObjectFilter) (map[ObjectKey]*ManifestObject, error) {
	r, err := loadManifest(restClientGetter, namespace, manifest)
	if err != nil {
		return nil, fmt.Errorf("error loading manifest: %w", err)
	}
	allObjects := make(map[ObjectKey]*ManifestObject, 0)
	err = r.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}
		if f != nil && !f.Matches(info) {
			return nil
		}
		key := ObjectKey{
			GVK:       info.Mapping.GroupVersionKind,
			Namespace: info.Namespace,
			Name:      info.Name,
		}
		allObjects[key] = &ManifestObject{
			Namespace: info.Namespace,
			Name:      info.Name,
			Object:    info.Object.DeepCopyObject(),
		}
		return nil
	})
	return allObjects, err
}

// loadManifest loads a manifest string into a resource.Result. It ignores unknown schema errors if the CRD is not yet present.
func loadManifest(restClientGetter genericclioptions.RESTClientGetter, namespace, manifest string) (*resource.Result, error) {
	reader := strings.NewReader(manifest)
	r := resource.
		NewBuilder(restClientGetter).
		Unstructured().
		NamespaceParam(namespace).DefaultNamespace().
		Stream(reader, "manifest").
		ContinueOnError().
		Flatten().
		Do().
		IgnoreErrors(meta.IsNoMatchError)
	if err := r.Err(); err != nil {
		return nil, err
	}
	return r, nil
}
