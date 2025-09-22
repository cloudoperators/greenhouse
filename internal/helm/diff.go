// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/pkg/errors"
	"github.com/wI2L/jsondiff"
	"helm.sh/helm/v3/pkg/release"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/dynamic"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"
)

const (
	greenhouseFieldManager = "greenhouse-helm-drift"
	secretMask             = "*****"
	secretBeforeMask       = "***** - before"
	secretAfterMask        = "***** - after"
)

type (
	// DiffObject is a Kubernetes object where the deployed state differs from the one in the Helm chart manifest.
	DiffObject struct {
		// Name is the GVK and namespaced name of the involved Kubernetes object.
		Name,
		// Diff is the JSON-patch style string version of the differences.
		Diff string
	}

	// DiffObjectList is a list of DiffObjects.
	DiffObjectList []DiffObject
)

// String returns the string of the DiffObjectList.
func (d DiffObjectList) String() string {
	if d == nil {
		return ""
	}
	allObjs := make([]string, len(d))
	for idx, o := range d {
		allObjs[idx] = fmt.Sprintf("%s: %s", o.Name, o.Diff)
	}
	return strings.Join(allObjs, ",")
}

// diffAgainstRelease returns the diff between the templated manifest and the manifest of the deployed Helm release.
func diffAgainstRelease(restClientGetter genericclioptions.RESTClientGetter, namespace string, helmTemplateRelease, helmRelease *release.Release) (DiffObjectList, error) {
	remoteObjs, err := ObjectMapFromRelease(restClientGetter, helmRelease, nil)
	if err != nil {
		return nil, err
	}

	for _, hook := range helmRelease.Hooks {
		hookObjs, err := ObjectMapFromManifest(restClientGetter, helmRelease.Namespace, hook.Manifest, nil)
		if err != nil {
			return nil, err
		}
		maps.Copy(remoteObjs, hookObjs)
	}

	localObjs, err := ObjectMapFromManifest(restClientGetter, namespace, helmTemplateRelease.Manifest, nil)
	if err != nil {
		return nil, err
	}

	for _, hook := range helmTemplateRelease.Hooks {
		hookObjs, err := ObjectMapFromManifest(restClientGetter, namespace, hook.Manifest, nil)
		if err != nil {
			return nil, err
		}
		maps.Copy(localObjs, hookObjs)
	}

	// create the set of all keys in local and remote objects
	keys := make(map[ObjectKey]struct{})
	for k := range remoteObjs {
		keys[k] = struct{}{}
	}
	for k := range localObjs {
		keys[k] = struct{}{}
	}

	// Iterate through all manifest objects and find the diff to the deployed version.
	allDiffs := make([]DiffObject, 0)
	for k := range keys {
		diff, err := diffObject(getRuntimeObject(remoteObjs, k), getRuntimeObject(localObjs, k))
		if err != nil {
			return nil, fmt.Errorf("failed to diff %s/%s: %w", k.GVK.Kind, k.Name, err)
		}
		if diff != "" {
			allDiffs = append(allDiffs, DiffObject{
				Name: k.GVK.Kind + "/" + k.Name,
				Diff: diff,
			})
		}
	}
	return allDiffs, err
}

// diffAgainstRemoteCRDs compares the CRDObjects from helm release with CRDs in remote cluster.
func diffAgainstRemoteCRDs(restClientGetter genericclioptions.RESTClientGetter, helmRelease *release.Release) (DiffObjectList, error) {
	restConfig, err := restClientGetter.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	crdGetter := resource.CRDFromDynamic(dynamicClient)
	crdFinder := resource.NewCRDFinder(crdGetter)

	crdList := helmRelease.Chart.CRDObjects()

	allDiffs := make([]DiffObject, 0)
	for _, crdFile := range crdList {
		if crdFile.File == nil || crdFile.File.Data == nil {
			continue
		}
		// Read the manifest to an object.
		crd := &apiextensionsv1.CustomResourceDefinition{}
		if err := yaml.Unmarshal(crdFile.File.Data, crd); err != nil {
			return nil, err
		}
		found, err := crdFinder.HasCRD(schema.GroupKind{Group: crd.Spec.Group, Kind: crd.Spec.Names.Kind})
		if err != nil {
			return nil, err
		}
		if found {
			continue
		}
		allDiffs = append(allDiffs, DiffObject{
			Name: crd.GetName(),
			Diff: "missing CRD",
		})
	}
	return allDiffs, nil
}

// diffAgainstLiveObjects compares the objects in the templated manifest with the objects deployed in the cluster.
func diffAgainstLiveObjects(restClientGetter genericclioptions.RESTClientGetter, namespace, manifest string) (DiffObjectList, error) {
	r, err := loadManifest(restClientGetter, namespace, manifest)
	if err != nil {
		return nil, err
	}
	// Iterate through all manifest objects and find the diff to the deployed version.
	allDiffs := make([]DiffObject, 0)
	err = r.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}
		// keep a copy of the original object from the chart manifest
		local := info.Object.DeepCopyObject()
		// get the deployed object from the cluster
		if err := info.Get(); err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}
			// We use this to indicate the object does not exist in the cluster.
			info.Object = nil
		}
		diff, err := diffApplyObject(info, local)
		if err != nil {
			return fmt.Errorf("failed to server-side diff %s/%s: %w", info.Mapping.GroupVersionKind.Kind, info.Name, err)
		}
		if diff != "" {
			allDiffs = append(allDiffs, DiffObject{
				Name: info.ObjectName(),
				Diff: diff,
			})
		}
		return nil
	})
	return allDiffs, err
}

// diffApplyObject returns the diff between the "live" object deployed in the cluster and the "local" object from the Helm chart manifest.
// the diff is calculated by doing a "server-side apply dry-run" of the chart object and comparing the result with the live object retrieved
// from the server.
func diffApplyObject(live *resource.Info, local runtime.Object) (diff string, err error) {
	// Prune the info object before getting merged object.
	for _, f := range []pruneFunc{
		pruneManagedFields, pruneLastAppliedAnnotation,
	} {
		live.Object = f(live.Object)
	}

	// server-side apply the chart object for comparison
	merged, err := getMergedObject(live, local)
	if err != nil {
		return "", err
	}
	// Prune the merged object as well.
	for _, f := range []pruneFunc{
		pruneManagedFields, pruneLastAppliedAnnotation,
	} {
		merged = f(merged)
	}
	return diffObject(live.Object, merged)
}

// diffObject returns the diff between the "live" object deployed in the cluster and the "local" object from the Helm chart manifest.
func diffObject(live, local runtime.Object) (diff string, err error) {
	if isSecret(live) || isSecret(local) {
		maskedLive, maskedLocal, err := maskSecret(live, local)
		if err != nil {
			return "", fmt.Errorf("error masking secret: %w", err)
		}
		live = maskedLive
		local = maskedLocal
	}

	patch, err := jsondiff.Compare(live, local, jsondiff.Equivalent())
	if err != nil {
		return "", err
	}
	if len(patch) == 0 {
		return "", nil
	}

	b, err := json.MarshalIndent(patch, "", "    ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// getMergedObject returns the object as it would look after a server-side apply dry-run.
// this will ensure all server-side defaults are applied to the object.
func getMergedObject(live *resource.Info, local runtime.Object) (runtime.Object, error) {
	helper := resource.
		NewHelper(live.Client, live.Mapping).
		DryRun(true).
		WithFieldManager(greenhouseFieldManager)
	// The object doesn't exist.
	if live.Object == nil {
		newObject, err := helper.CreateWithOptions(live.Namespace, true, local,
			&metav1.CreateOptions{
				DryRun: []string{metav1.DryRunAll},
			},
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create object")
		}
		return newObject, nil
	}
	// Patch with the chart object.
	data, err := runtime.Encode(unstructured.UnstructuredJSONScheme, local)
	if err != nil {
		return nil, err
	}

	newObject, err := helper.Patch(live.Namespace, live.Name, types.ApplyPatchType, data,
		&metav1.PatchOptions{
			// To avoid conflicts Kubernetes will only allow one field manager to update an object at a time. Force ensures the dry-run patch is applied. see https://kubernetes.io/docs/reference/using-api/server-side-apply/#conflicts
			Force:        ptr.To(true),
			FieldManager: greenhouseFieldManager,
			DryRun:       []string{metav1.DryRunAll},
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to patch object")
	}
	return newObject, nil
}

// maskSecret masks the secret data in the live and merged object.
func maskSecret(live, merged runtime.Object) (maskedBefore, maskedAfter runtime.Object, err error) {
	unstrucBefore, err := toUnstructuredContent(live)
	if err != nil {
		return nil, nil, err
	}
	unstrucAfter, err := toUnstructuredContent(merged)
	if err != nil {
		return nil, nil, err
	}

	beforeData, err := secretData(unstrucBefore)
	if err != nil {
		return nil, nil, err
	}

	afterData, err := secretData(unstrucAfter)
	if err != nil {
		return nil, nil, err
	}

	for k := range beforeData {
		if _, ok := afterData[k]; ok {
			if beforeData[k] != afterData[k] {
				// value is different, use mask with suffix to indicate a difference
				afterData[k] = secretAfterMask
				beforeData[k] = secretBeforeMask
				continue
			}
			afterData[k] = secretMask
			beforeData[k] = secretMask
		}
	}

	for k := range afterData {
		if _, ok := beforeData[k]; !ok {
			afterData[k] = secretMask
		}
	}

	if unstrucBefore != nil && beforeData != nil {
		if err := unstructured.SetNestedMap(unstrucBefore, beforeData, "data"); err != nil {
			return nil, nil, fmt.Errorf("failed to set masked data in before secret: %w", err)
		}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstrucBefore, live); err != nil {
			return nil, nil, fmt.Errorf("failed to update before object: %w", err)
		}
	}
	if unstrucAfter != nil && afterData != nil {
		if err := unstructured.SetNestedMap(unstrucAfter, afterData, "data"); err != nil {
			return nil, nil, fmt.Errorf("failed to set masked data in after secret: %w", err)
		}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstrucAfter, merged); err != nil {
			return nil, nil, fmt.Errorf("failed to update after object: %w", err)
		}
	}

	return live, merged, nil
}

// toUnstructuredContent returns the unstructured content of runtime.Object.
func toUnstructuredContent(o runtime.Object) (map[string]any, error) {
	if o == nil {
		return nil, nil
	}
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(o.DeepCopyObject())
	if err != nil {
		return nil, fmt.Errorf("failed to convert object to unstructured: %w", err)
	}

	if u == nil { // no content, nothing to do
		return nil, nil
	}
	return u, nil
}

// secretData returns the secret data if found in the unstructured data.
func secretData(u map[string]any) (map[string]any, error) {
	// secret has not "data" specified, nothing to do
	if u["data"] == nil {
		return nil, nil
	}
	data, found, err := unstructured.NestedMap(u, "data")
	if err != nil {
		return nil, fmt.Errorf("failed to get data from secret: %w", err)
	}
	if !found { // no data nothing to do
		return nil, nil
	}
	return data, nil
}

// isSecret returns true if the object has a GVK v1/Secret
func isSecret(o runtime.Object) bool {
	if o == nil {
		return false
	}
	if gvk := o.GetObjectKind().GroupVersionKind(); gvk.Version == "v1" && gvk.Kind == "Secret" {
		return true
	}
	return false
}

// getRuntimeObject returns the runtime.Object for a given key exists, else the Object is nil
func getRuntimeObject(m map[ObjectKey]*ManifestObject, k ObjectKey) runtime.Object {
	if v, ok := m[k]; ok {
		return v.Object
	}
	return nil
}
