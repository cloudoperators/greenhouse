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
	APIVersion, Kind, Name string
	Annotations            map[string]string
}

type ObjectList struct {
	ObjectKey
	*ManifestObject
}

type ManifestFilter interface {
	Matches(obj *resource.Info) bool
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
	if o.Name != "" && o.Name != obj.Name {
		return false
	}
	if o.Annotations != nil {
		metaAccessor, err := meta.Accessor(obj.Object)
		if err != nil {
			return false
		}

		if metaAccessor.GetAnnotations() == nil {
			return false
		}
		for k, v := range o.Annotations {
			val, ok := metaAccessor.GetAnnotations()[k]
			if !ok || v != val {
				return false
			}
		}
	}
	return true
}

// ObjectMapFromRelease returns a map of objects from the helm release manifest matching the filter or an error.
func ObjectMapFromRelease(restClientGetter genericclioptions.RESTClientGetter, r *release.Release, f ManifestFilter) (map[ObjectKey]*ManifestObject, error) {
	return ObjectMapFromManifest(restClientGetter, r.Namespace, r.Manifest, f)
}

// ObjectMapFromManifest returns a map of objects from the manifests matching the filter or an error.
func ObjectMapFromManifest(restClientGetter genericclioptions.RESTClientGetter, namespace, manifest string, f ManifestFilter) (map[ObjectKey]*ManifestObject, error) {
	r, err := loadManifest(restClientGetter, namespace, manifest)
	if err != nil {
		return nil, fmt.Errorf("error loading manifest: %w", err)
	}
	return objectMapping(r, f)
}

func ObjectMapFromLocalManifest(f ManifestFilter, manifest string) (map[ObjectKey]*ManifestObject, error) {
	r, err := loadLocalManifest(manifest)
	if err != nil {
		return nil, fmt.Errorf("error loading local manifest: %w", err)
	}
	return objectMapping(r, f)
}

func objectMapping(r *resource.Result, f ManifestFilter) (map[ObjectKey]*ManifestObject, error) {
	allObjects := make(map[ObjectKey]*ManifestObject)
	err := r.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}
		if f != nil && !f.Matches(info) {
			return nil
		}
		var gvk schema.GroupVersionKind
		if info.Mapping != nil {
			gvk = info.Mapping.GroupVersionKind
		} else {
			gvk = info.Object.GetObjectKind().GroupVersionKind()
		}
		key := ObjectKey{
			GVK:       gvk,
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

func loadLocalManifest(manifest string) (*resource.Result, error) {
	reader := strings.NewReader(manifest)
	r := resource.
		NewLocalBuilder().
		Unstructured().
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
