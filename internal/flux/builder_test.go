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
	builder := NewHelmReleaseSpecBuilder().New("my-release", "my-namespace").
		WithChart(helmv2.HelmChartTemplateSpec{
			Chart:   "nginx",
			Version: "1.2.3",
		}).
		WithMaxHistory(5).
		WithInterval(10 * time.Minute).
		WithTimeout(2 * time.Minute).
		WithReleaseName("custom-release").
		WithTargetNamespace("target-ns")

	hr, err := builder.Build()
	assert.NoError(t, err)
	assert.NotNil(t, hr)
	assert.Equal(t, "my-release", hr.Name)
	assert.Equal(t, "my-namespace", hr.Namespace)
	assert.Equal(t, "nginx", hr.Spec.Chart.Spec.Chart)
	assert.Equal(t, "1.2.3", hr.Spec.Chart.Spec.Version)
	assert.Equal(t, 5, *hr.Spec.MaxHistory)
	assert.Equal(t, "custom-release", hr.Spec.ReleaseName)
	assert.Equal(t, "target-ns", hr.Spec.TargetNamespace)
}

func TestHelmReleaseBuilder_ChartNameRequired(t *testing.T) {
	builder := NewHelmReleaseSpecBuilder().New("release", "ns").
		WithChart(helmv2.HelmChartTemplateSpec{
			Chart:   "",
			Version: "1.0.0",
		})

	hr, err := builder.Build()
	assert.Error(t, err)
	assert.Nil(t, hr)
	assert.Equal(t, "chart name is required", err.Error())
}

func TestHelmReleaseBuilder_ChartVersionRequired(t *testing.T) {
	builder := NewHelmReleaseSpecBuilder().New("release", "ns").
		WithChart(helmv2.HelmChartTemplateSpec{
			Chart:   "nginx",
			Version: "",
		})

	hr, err := builder.Build()
	assert.Error(t, err)
	assert.Nil(t, hr)
	assert.Equal(t, "chart version is required", err.Error())
}

func TestHelmReleaseBuilder_WithMaxHistoryNegativeValueIgnored(t *testing.T) {
	builder := NewHelmReleaseSpecBuilder().New("release", "ns").
		WithChart(helmv2.HelmChartTemplateSpec{
			Chart:   "nginx",
			Version: "1.0.0",
		}).
		WithMaxHistory(-1)

	hr, err := builder.Build()
	assert.NoError(t, err)
	assert.Nil(t, hr.Spec.MaxHistory)
}

func TestHelmReleaseBuilder_WithIntervalZeroOrNegativeIgnored(t *testing.T) {
	builder := NewHelmReleaseSpecBuilder().New("release", "ns").
		WithChart(helmv2.HelmChartTemplateSpec{
			Chart:   "nginx",
			Version: "1.0.0",
		}).
		WithInterval(0)

	hr, err := builder.Build()
	assert.NoError(t, err)
	assert.Equal(t, metav1.Duration{}, hr.Spec.Interval)
}

func TestHelmReleaseBuilder_WithTimeoutZeroOrNegativeIgnored(t *testing.T) {
	builder := NewHelmReleaseSpecBuilder().New("release", "ns").
		WithChart(helmv2.HelmChartTemplateSpec{
			Chart:   "nginx",
			Version: "1.0.0",
		}).
		WithTimeout(-5 * time.Second)

	hr, err := builder.Build()
	assert.NoError(t, err)
	assert.Nil(t, hr.Spec.Timeout)
}

func TestHelmReleaseBuilder_WithValuesNilIgnored(t *testing.T) {
	builder := NewHelmReleaseSpecBuilder().New("release", "ns").
		WithChart(helmv2.HelmChartTemplateSpec{
			Chart:   "nginx",
			Version: "1.0.0",
		}).
		WithValues(nil)

	hr, err := builder.Build()
	assert.NoError(t, err)
	assert.NotNil(t, hr.Spec.Values)
}

func TestHelmReleaseBuilder_WithValuesFromEmptyIgnored(t *testing.T) {
	builder := NewHelmReleaseSpecBuilder().New("release", "ns").
		WithChart(helmv2.HelmChartTemplateSpec{
			Chart:   "nginx",
			Version: "1.0.0",
		}).
		WithValuesFrom([]helmv2.ValuesReference{})

	hr, err := builder.Build()
	assert.NoError(t, err)
	assert.Nil(t, hr.Spec.ValuesFrom)
}

func TestHelmReleaseBuilder_WithReleaseNameEmptyIgnored(t *testing.T) {
	builder := NewHelmReleaseSpecBuilder().New("release", "ns").
		WithChart(helmv2.HelmChartTemplateSpec{
			Chart:   "nginx",
			Version: "1.0.0",
		}).
		WithReleaseName("")

	hr, err := builder.Build()
	assert.NoError(t, err)
	assert.Equal(t, "", hr.Spec.ReleaseName)
}

func TestHelmReleaseBuilder_WithTargetNamespaceEmptyIgnored(t *testing.T) {
	builder := NewHelmReleaseSpecBuilder().New("release", "ns").
		WithChart(helmv2.HelmChartTemplateSpec{
			Chart:   "nginx",
			Version: "1.0.0",
		}).
		WithTargetNamespace("")

	hr, err := builder.Build()
	assert.NoError(t, err)
	assert.Equal(t, "", hr.Spec.TargetNamespace)
}

func TestHelmReleaseBuilder_WithDependsOnEmptyIgnored(t *testing.T) {
	builder := NewHelmReleaseSpecBuilder().New("release", "ns").
		WithChart(helmv2.HelmChartTemplateSpec{
			Chart:   "nginx",
			Version: "1.0.0",
		}).
		WithDependsOn([]meta.NamespacedObjectReference{})

	hr, err := builder.Build()
	assert.NoError(t, err)
	assert.Nil(t, hr.Spec.DependsOn)
}

func TestHelmReleaseBuilder_WithKubeConfigEmptyIgnored(t *testing.T) {
	builder := NewHelmReleaseSpecBuilder().New("release", "ns").
		WithChart(helmv2.HelmChartTemplateSpec{
			Chart:   "nginx",
			Version: "1.0.0",
		}).
		WithKubeConfig(meta.SecretKeyReference{})

	hr, err := builder.Build()
	assert.NoError(t, err)
	assert.Equal(t, meta.SecretKeyReference{}, hr.Spec.KubeConfig.SecretRef)
}
