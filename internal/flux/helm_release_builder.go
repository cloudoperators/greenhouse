// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package flux

import (
	"errors"
	"time"

	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
)

type HelmReleaseBuilder struct {
	spec helmv2.HelmReleaseSpec
}

func NewHelmReleaseSpecBuilder() *HelmReleaseBuilder {
	return &HelmReleaseBuilder{
		spec: helmv2.HelmReleaseSpec{
			Install: &helmv2.Install{
				Remediation: &helmv2.InstallRemediation{},
			},
			Upgrade: &helmv2.Upgrade{
				Remediation: &helmv2.UpgradeRemediation{},
			},
			KubeConfig:     nil,
			DriftDetection: &helmv2.DriftDetection{},
			Test:           &helmv2.Test{},
			Values:         &v1.JSON{},
		},
	}
}

func (b *HelmReleaseBuilder) WithHelmChartRef(chartRef *helmv2.CrossNamespaceSourceReference) *HelmReleaseBuilder {
	if chartRef == nil {
		return b
	}
	b.spec.ChartRef = chartRef
	return b
}

// WithChart sets the chart specification for the Helm release.
func (b *HelmReleaseBuilder) WithChart(specRef helmv2.HelmChartTemplateSpec) *HelmReleaseBuilder {
	b.spec.Chart = &helmv2.HelmChartTemplate{
		Spec: specRef,
	}
	return b
}

// WithMaxHistory sets the maximum history for the Helm release.
func (b *HelmReleaseBuilder) WithMaxHistory(num int) *HelmReleaseBuilder {
	if num < 0 {
		return b
	}
	b.spec.MaxHistory = ptr.To[int](num)
	return b
}

// WithInterval sets the interval for the Helm release.
func (b *HelmReleaseBuilder) WithInterval(duration time.Duration) *HelmReleaseBuilder {
	if duration <= 0 {
		return b
	}
	b.spec.Interval = metav1.Duration{Duration: duration}
	return b
}

// WithTimeout sets the timeout for the Helm release.
func (b *HelmReleaseBuilder) WithTimeout(timeout time.Duration) *HelmReleaseBuilder {
	if timeout <= 0 {
		return b
	}
	b.spec.Timeout = &metav1.Duration{Duration: timeout}
	return b
}

// WithValues sets the values for the Helm release.
func (b *HelmReleaseBuilder) WithValues(byteValues []byte) *HelmReleaseBuilder {
	if byteValues == nil {
		return b
	}
	b.spec.Values = &v1.JSON{Raw: byteValues}
	return b
}

// WithValuesFrom sets the values references for the Helm release. Only secret references are supported on the plugin side
func (b *HelmReleaseBuilder) WithValuesFrom(ref []helmv2.ValuesReference) *HelmReleaseBuilder {
	if len(ref) == 0 {
		return b
	}
	b.spec.ValuesFrom = ref
	return b
}

// WithReleaseName sets the release name for the Helm release.
func (b *HelmReleaseBuilder) WithReleaseName(name string) *HelmReleaseBuilder {
	if name == "" {
		return b
	}
	b.spec.ReleaseName = name
	return b
}

// WithStorageNamespace sets the target namespace for the Helm release.
func (b *HelmReleaseBuilder) WithStorageNamespace(namespace string) *HelmReleaseBuilder {
	if namespace == "" {
		return b
	}
	b.spec.StorageNamespace = namespace
	return b
}

// WithTargetNamespace sets the target namespace for the Helm release.
func (b *HelmReleaseBuilder) WithTargetNamespace(namespace string) *HelmReleaseBuilder {
	if namespace == "" {
		return b
	}
	b.spec.TargetNamespace = namespace
	return b
}

// WithInstall sets the installation configuration for the Helm release.
func (b *HelmReleaseBuilder) WithInstall(install *helmv2.Install) *HelmReleaseBuilder {
	if install == nil {
		install = &helmv2.Install{
			Timeout: &metav1.Duration{Duration: DefaultTimeout},
			Remediation: &helmv2.InstallRemediation{
				Retries: DefaultRetry,
			},
		}
	}
	b.spec.Install = install
	return b
}

// WithUpgrade sets the upgrade configuration for the Helm release.
func (b *HelmReleaseBuilder) WithUpgrade(upgrade *helmv2.Upgrade) *HelmReleaseBuilder {
	if upgrade == nil {
		upgrade = &helmv2.Upgrade{
			Timeout: &metav1.Duration{Duration: DefaultTimeout},
			Remediation: &helmv2.UpgradeRemediation{
				Retries: DefaultRetry,
			},
		}
	}
	b.spec.Upgrade = upgrade
	return b
}

// WithRollback sets the rollback configuration for the Helm release.
func (b *HelmReleaseBuilder) WithRollback(rollback *helmv2.Rollback) *HelmReleaseBuilder {
	if rollback == nil {
		rollback = &helmv2.Rollback{
			Timeout: &metav1.Duration{Duration: DefaultTimeout},
		}
	}
	b.spec.Rollback = rollback
	return b
}

// WithDriftDetection sets the drift detection configuration for the Helm release.
func (b *HelmReleaseBuilder) WithDriftDetection(driftDetection *helmv2.DriftDetection) *HelmReleaseBuilder {
	if driftDetection == nil {
		driftDetection = &helmv2.DriftDetection{
			Mode: helmv2.DriftDetectionEnabled,
		}
	}
	b.spec.DriftDetection = driftDetection
	return b
}

// WithSuspend sets the suspend flag for the Helm release.
func (b *HelmReleaseBuilder) WithSuspend(suspend bool) *HelmReleaseBuilder {
	b.spec.Suspend = suspend
	return b
}

// WithTest sets the test configuration for the Helm release.
func (b *HelmReleaseBuilder) WithTest(test *helmv2.Test) *HelmReleaseBuilder {
	if test == nil {
		test = &helmv2.Test{
			Enable:  true,
			Timeout: &metav1.Duration{Duration: DefaultTimeout},
		}
	}
	if !test.Enable {
		return b
	}
	b.spec.Test = test
	return b
}

// WithUninstall sets the uninstallation configuration for the Helm release.
func (b *HelmReleaseBuilder) WithUninstall(uninstall *helmv2.Uninstall) *HelmReleaseBuilder {
	if uninstall == nil {
		uninstall = &helmv2.Uninstall{
			Timeout: &metav1.Duration{Duration: DefaultTimeout},
		}
	}
	b.spec.Uninstall = uninstall
	return b
}

// WithDependsOn sets the dependencies for the Helm release.
func (b *HelmReleaseBuilder) WithDependsOn(dependencies []helmv2.DependencyReference) *HelmReleaseBuilder {
	if len(dependencies) == 0 {
		return b
	}
	b.spec.DependsOn = dependencies
	return b
}

// WithKubeConfig sets the kubeconfig reference for the Helm release. If the fluxmeta.SecretKeyReference does not contain a name, the Plugin targets the central cluster and no specific kubeconfig is needed.
func (b *HelmReleaseBuilder) WithKubeConfig(kc *fluxmeta.SecretKeyReference) *HelmReleaseBuilder {
	if kc.Name == "" { // Name is empty if Plugin is deployed in central cluster
		return b
	}
	b.spec.KubeConfig = &fluxmeta.KubeConfigReference{
		SecretRef: kc,
	}
	return b
}

// WithPostRenderers sets the post renderers for the Helm release.
func (b *HelmReleaseBuilder) WithPostRenderers(postRenderers []helmv2.PostRenderer) *HelmReleaseBuilder {
	if len(postRenderers) == 0 {
		return b
	}
	b.spec.PostRenderers = postRenderers
	return b
}

// Build validates the HelmRelease and returns it.
func (b *HelmReleaseBuilder) Build() (helmv2.HelmReleaseSpec, error) {
	if b.spec.ChartRef == nil {
		return helmv2.HelmReleaseSpec{}, errors.New("chartRef must be set")
	}
	return b.spec, nil
}
