// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package flux

import (
	"testing"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	"github.com/fluxcd/pkg/apis/meta"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHelmReleaseBuilder_CreatesValidHelmRelease(t *testing.T) {
	builder := NewHelmReleaseSpecBuilder().
		WithChart(helmv2.HelmChartTemplateSpec{
			Chart:   "nginx",
			Version: "1.2.3",
		}).
		WithMaxHistory(5).
		WithInterval(10 * time.Minute).
		WithTimeout(2 * time.Minute).
		WithReleaseName("custom-release").
		WithStorageNamespace("target-ns").
		WithTargetNamespace("target-ns")

	spec, err := builder.Build()
	assert.NoError(t, err)
	assert.Equal(t, "nginx", spec.Chart.Spec.Chart)
	assert.Equal(t, "1.2.3", spec.Chart.Spec.Version)
	assert.Equal(t, 5, *spec.MaxHistory)
	assert.Equal(t, "custom-release", spec.ReleaseName)
	assert.Equal(t, "target-ns", spec.StorageNamespace)
	assert.Equal(t, "target-ns", spec.TargetNamespace)
}

func TestHelmReleaseBuilder_ChartNameRequired(t *testing.T) {
	builder := NewHelmReleaseSpecBuilder().
		WithChart(helmv2.HelmChartTemplateSpec{
			Chart:   "",
			Version: "1.0.0",
		})

	_, err := builder.Build()
	assert.Error(t, err)
	assert.Equal(t, "chart name is required", err.Error())
}

func TestHelmReleaseBuilder_ChartVersionRequired(t *testing.T) {
	builder := NewHelmReleaseSpecBuilder().
		WithChart(helmv2.HelmChartTemplateSpec{
			Chart:   "nginx",
			Version: "",
		})

	_, err := builder.Build()
	assert.Error(t, err)
	assert.Equal(t, "chart version is required", err.Error())
}

func TestHelmReleaseBuilder_WithMaxHistoryNegativeValueIgnored(t *testing.T) {
	builder := NewHelmReleaseSpecBuilder().
		WithChart(helmv2.HelmChartTemplateSpec{
			Chart:   "nginx",
			Version: "1.0.0",
		}).
		WithMaxHistory(-1)

	spec, err := builder.Build()
	assert.NoError(t, err)
	assert.Nil(t, spec.MaxHistory)
}

func TestHelmReleaseBuilder_WithIntervalZeroOrNegativeIgnored(t *testing.T) {
	builder := NewHelmReleaseSpecBuilder().
		WithChart(helmv2.HelmChartTemplateSpec{
			Chart:   "nginx",
			Version: "1.0.0",
		}).
		WithInterval(0)

	spec, err := builder.Build()
	assert.NoError(t, err)
	assert.Equal(t, metav1.Duration{}, spec.Interval)
}

func TestHelmReleaseBuilder_WithTimeoutZeroOrNegativeIgnored(t *testing.T) {
	builder := NewHelmReleaseSpecBuilder().
		WithChart(helmv2.HelmChartTemplateSpec{
			Chart:   "nginx",
			Version: "1.0.0",
		}).
		WithTimeout(-5 * time.Second)

	spec, err := builder.Build()
	assert.NoError(t, err)
	assert.Nil(t, spec.Timeout)
}

func TestHelmReleaseBuilder_WithValuesNilIgnored(t *testing.T) {
	builder := NewHelmReleaseSpecBuilder().
		WithChart(helmv2.HelmChartTemplateSpec{
			Chart:   "nginx",
			Version: "1.0.0",
		}).
		WithValues(nil)

	spec, err := builder.Build()
	assert.NoError(t, err)
	assert.NotNil(t, spec.Values)
}

func TestHelmReleaseBuilder_WithValuesFromEmptyIgnored(t *testing.T) {
	builder := NewHelmReleaseSpecBuilder().
		WithChart(helmv2.HelmChartTemplateSpec{
			Chart:   "nginx",
			Version: "1.0.0",
		}).
		WithValuesFrom([]helmv2.ValuesReference{})

	spec, err := builder.Build()
	assert.NoError(t, err)
	assert.Nil(t, spec.ValuesFrom)
}

func TestHelmReleaseBuilder_WithReleaseNameEmptyIgnored(t *testing.T) {
	builder := NewHelmReleaseSpecBuilder().
		WithChart(helmv2.HelmChartTemplateSpec{
			Chart:   "nginx",
			Version: "1.0.0",
		}).
		WithReleaseName("")

	spec, err := builder.Build()
	assert.NoError(t, err)
	assert.Equal(t, "", spec.ReleaseName)
}

func TestHelmReleaseBuilder_WithTargetNamespaceEmptyIgnored(t *testing.T) {
	builder := NewHelmReleaseSpecBuilder().
		WithChart(helmv2.HelmChartTemplateSpec{
			Chart:   "nginx",
			Version: "1.0.0",
		}).
		WithTargetNamespace("")

	spec, err := builder.Build()
	assert.NoError(t, err)
	assert.Equal(t, "", spec.TargetNamespace)
}

func TestHelmReleaseBuilder_WithDependsOnEmptyIgnored(t *testing.T) {
	builder := NewHelmReleaseSpecBuilder().
		WithChart(helmv2.HelmChartTemplateSpec{
			Chart:   "nginx",
			Version: "1.0.0",
		}).
		WithDependsOn([]meta.NamespacedObjectReference{})

	spec, err := builder.Build()
	assert.NoError(t, err)
	assert.Nil(t, spec.DependsOn)
}

func TestHelmReleaseBuilder_WithKubeConfigEmptyIgnored(t *testing.T) {
	builder := NewHelmReleaseSpecBuilder().
		WithChart(helmv2.HelmChartTemplateSpec{
			Chart:   "nginx",
			Version: "1.0.0",
		}).
		WithKubeConfig(meta.SecretKeyReference{})

	spec, err := builder.Build()
	assert.NoError(t, err)
	assert.Nil(t, spec.KubeConfig)
}

func TestHelmReleaseBuilder_WithKubeConfigFromSecret(t *testing.T) {
	builder := NewHelmReleaseSpecBuilder().
		WithChart(helmv2.HelmChartTemplateSpec{
			Chart:   "nginx",
			Version: "1.0.0",
		}).
		WithKubeConfig(meta.SecretKeyReference{
			Name: "kubeconfig-secret",
			Key:  "kubeconfig",
		})

	spec, err := builder.Build()
	assert.NoError(t, err)
	assert.NotNil(t, spec.KubeConfig)
	assert.Equal(t, "kubeconfig-secret", spec.KubeConfig.SecretRef.Name)
	assert.Equal(t, "kubeconfig", spec.KubeConfig.SecretRef.Key)
}
