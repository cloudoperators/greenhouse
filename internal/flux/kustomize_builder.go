// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package flux

import (
	"encoding/json"
	"errors"
	"fmt"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	fluxkust "github.com/fluxcd/pkg/apis/kustomize"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

const (
	kustomizeOperationTest       = "test"
	kustomizeReplacePluginOption = "/spec/options/%d"
	kustomizeTestPluginOption    = "/spec/options/%d/name"
	kustomizeOperationReplace    = "replace"
	kustomizeMetadataNamePath    = "/metadata/name"
	kustomizeHelmRepoPatch       = "/spec/helmChart/repository"
)

// Operation model for patch operations
type Operation struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value,omitempty"`
}

func constructPatchOperations(op, path string, value any) Operation {
	return Operation{
		Op:    op,
		Path:  path,
		Value: value,
	}
}

func BuildJSONPatchReplace(opt greenhousev1alpha1.PluginOption, index int, group, name string) (fluxkust.Patch, error) {
	ops := []Operation{
		constructPatchOperations(kustomizeOperationTest, fmt.Sprintf(kustomizeTestPluginOption, index), opt.Name),
		constructPatchOperations(kustomizeOperationReplace, fmt.Sprintf(kustomizeReplacePluginOption, index), opt),
	}
	raw, err := json.Marshal(ops)
	if err != nil {
		return fluxkust.Patch{}, err
	}
	return fluxkust.Patch{
		Patch: string(raw),
		Target: &fluxkust.Selector{
			Group: group,
			Name:  name,
		},
	}, nil
}

func PrepareKustomizePatches(overrides []greenhousev1alpha1.CatalogOverrides, group string) ([]fluxkust.Patch, error) {
	patches := make([]fluxkust.Patch, 0)
	for _, override := range overrides {
		if override.Alias == "" && override.Repository == "" {
			continue
		}
		operations := make([]Operation, 0, len(overrides))
		if override.Alias != "" {
			operations = append(operations, constructPatchOperations(kustomizeOperationReplace, kustomizeMetadataNamePath, override.Alias))
		}
		if override.Repository != "" {
			operations = append(operations, constructPatchOperations(kustomizeOperationReplace, kustomizeHelmRepoPatch, override.Repository))
		}
		patched, err := json.Marshal(operations)
		if err != nil {
			return nil, err
		}
		patch := fluxkust.Patch{
			Patch: string(patched),
			Target: &fluxkust.Selector{
				Group: group,
				Name:  override.Name,
			},
		}
		patches = append(patches, patch)
	}
	return patches, nil
}

type KustomizeBuilder struct {
	log  logr.Logger
	spec kustomizev1.KustomizationSpec
}

func NewKustomizationSpecBuilder(logger logr.Logger) *KustomizeBuilder {
	return &KustomizeBuilder{
		log: logger.WithName("kustomization-builder"),
		spec: kustomizev1.KustomizationSpec{
			SourceRef: kustomizev1.CrossNamespaceSourceReference{},
		},
	}
}

func (k *KustomizeBuilder) WithSourceRef(apiVersion, kind, name, namespace string) *KustomizeBuilder {
	ref := kustomizev1.CrossNamespaceSourceReference{
		APIVersion: apiVersion,
		Kind:       kind,
		Name:       name,
		Namespace:  namespace,
	}
	k.spec.SourceRef = ref
	return k
}

func (k *KustomizeBuilder) WithPatches(patches []fluxkust.Patch) *KustomizeBuilder {
	if len(patches) > 0 {
		k.spec.Patches = patches
	}
	return k
}

func (k *KustomizeBuilder) WithServiceAccountName(name string) *KustomizeBuilder {
	k.spec.ServiceAccountName = name
	return k
}

func (k *KustomizeBuilder) WithPath(path string) *KustomizeBuilder {
	if path != "" {
		k.spec.Path = path
	}
	return k
}

func (k *KustomizeBuilder) WithTargetNamespace(namespace string) *KustomizeBuilder {
	k.spec.TargetNamespace = namespace
	return k
}

func (k *KustomizeBuilder) WithCommonLabels(labels map[string]string) *KustomizeBuilder {
	if labels != nil {
		k.spec.CommonMetadata = &kustomizev1.CommonMetadata{
			Labels: labels,
		}
	}
	return k
}

func (k *KustomizeBuilder) WithSuspend(suspend bool) *KustomizeBuilder {
	k.spec.Suspend = suspend
	return k
}

func (k *KustomizeBuilder) WithWait(wait bool) *KustomizeBuilder {
	k.spec.Wait = wait
	return k
}

func (k *KustomizeBuilder) Build() (kustomizev1.KustomizationSpec, error) {
	if k.spec.SourceRef.Kind == "" {
		return kustomizev1.KustomizationSpec{}, errors.New("source reference kind is required")
	}
	if k.spec.SourceRef.Name == "" {
		return kustomizev1.KustomizationSpec{}, errors.New("source reference name is required")
	}
	k.spec.Interval = metav1.Duration{Duration: DefaultInterval}
	return k.spec, nil
}
