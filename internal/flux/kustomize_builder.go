// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package flux

import (
	"bytes"
	"errors"
	"strings"
	"text/template"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	fluxkust "github.com/fluxcd/pkg/apis/kustomize"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

// Operation model for patch operations
type Operation struct {
	Op    string
	Path  string
	Value string
}

// Data holds all the patch ops
type Data struct {
	Operations []Operation
}

const opsTmpl = `{{- range .Operations}}
- op: {{ .Op }}
  path: {{ .Path }}
  value: |
{{ .Value | indent 4 }}
{{- end}}`

// indent is a small helper to indent a multi-line string
func indent(spaces int, s string) string {
	prefix := bytes.Repeat([]byte(" "), spaces)
	out := &bytes.Buffer{}
	for i, line := range bytes.Split([]byte(s), []byte("\n")) {
		if i > 0 {
			out.WriteByte('\n')
		}
		out.Write(prefix)
		out.Write(line)
	}
	return out.String()
}

func metadataPatchTemplate(alias string) (string, error) {
	tmpl, err := template.New("patchOps").Funcs(template.FuncMap{
		"indent": indent,
	}).Parse(opsTmpl)
	if err != nil {
		return "", err
	}
	d := Data{
		Operations: []Operation{
			{
				Op:    "replace",
				Path:  "/metadata/name",
				Value: alias,
			},
		},
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, d); err != nil {
		return "", err
	}

	// Remove leading newline characters from the output
	return strings.TrimLeft(buf.String(), "\n"), nil
}

func PrepareKustomizePatches(overrides []greenhouseapisv1alpha1.CatalogOverrides, group string) ([]fluxkust.Patch, error) {
	patches := make([]fluxkust.Patch, 0)
	for _, override := range overrides {
		metadataPatch, err := metadataPatchTemplate(override.Alias)
		if err != nil {
			return nil, err
		}
		patch := fluxkust.Patch{
			Patch: metadataPatch,
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

func (k *KustomizeBuilder) WithPath(path string) *KustomizeBuilder {
	if path != "" {
		k.spec.Path = path
	}
	return k
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
	k.spec.Patches = patches
	return k
}

func (k *KustomizeBuilder) WithServiceAccountName(name string) *KustomizeBuilder {
	k.spec.ServiceAccountName = name
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
