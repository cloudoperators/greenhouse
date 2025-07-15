// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package flux

import (
	"errors"
	"time"

	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	helmcontroller "github.com/fluxcd/helm-controller/api/v2"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
)

const (
	DefaultInterval = 5 * time.Minute
	DefaultTimeout  = 5 * time.Minute // TODO: make this configurable via annotations on plugin / environment variable (Test scenarios)
	DefaultRetry    = 3               // TODO: make this also configurable via annotations on plugin
)

type HelmReleaseBuilder interface {
	WithChart(specRef helmcontroller.HelmChartTemplateSpec) *helmReleaseBuilder
	WithMaxHistory(num int) *helmReleaseBuilder
	WithInterval(duration time.Duration) *helmReleaseBuilder
	WithTimeout(timeout time.Duration) *helmReleaseBuilder
	WithValues(byteValues []byte) *helmReleaseBuilder
	WithValuesFrom(ref []helmcontroller.ValuesReference) *helmReleaseBuilder
	WithReleaseName(name string) *helmReleaseBuilder
	WithTargetNamespace(namespace string) *helmReleaseBuilder
	WithInstall(install *helmcontroller.Install) *helmReleaseBuilder
	WithUpgrade(upgrade *helmcontroller.Upgrade) *helmReleaseBuilder
	WithRollback(rollback *helmcontroller.Rollback) *helmReleaseBuilder
	WithDriftDetection(driftDetection *helmcontroller.DriftDetection) *helmReleaseBuilder
	WithTest(test *helmcontroller.Test) *helmReleaseBuilder
	WithUninstall(uninstall *helmcontroller.Uninstall) *helmReleaseBuilder
	WithDependsOn(dependencies []fluxmeta.NamespacedObjectReference) *helmReleaseBuilder
	WithKubeConfig(kc fluxmeta.SecretKeyReference) *helmReleaseBuilder
	Build() (helmcontroller.HelmReleaseSpec, error)
}

type helmReleaseBuilder struct {
	spec helmcontroller.HelmReleaseSpec
}

func NewHelmReleaseSpecBuilder() HelmReleaseBuilder {
	return &helmReleaseBuilder{
		spec: helmcontroller.HelmReleaseSpec{
			Install: &helmcontroller.Install{
				Remediation: &helmcontroller.InstallRemediation{},
			},
			Upgrade: &helmcontroller.Upgrade{
				Remediation: &helmcontroller.UpgradeRemediation{},
			},
			DriftDetection: &helmcontroller.DriftDetection{},
			Test:           &helmcontroller.Test{},
			KubeConfig:     &fluxmeta.KubeConfigReference{},
			Values:         &v1.JSON{},
		},
	}
}

// WithChart sets the chart specification for the Helm release.
func (b *helmReleaseBuilder) WithChart(specRef helmcontroller.HelmChartTemplateSpec) *helmReleaseBuilder {
	b.spec.Chart = &helmcontroller.HelmChartTemplate{
		Spec: specRef,
	}
	return b
}

// WithMaxHistory sets the maximum history for the Helm release.
func (b *helmReleaseBuilder) WithMaxHistory(num int) *helmReleaseBuilder {
	if num < 0 {
		return b
	}
	b.spec.MaxHistory = ptr.To[int](num)
	return b
}

// WithInterval sets the interval for the Helm release.
func (b *helmReleaseBuilder) WithInterval(duration time.Duration) *helmReleaseBuilder {
	if duration <= 0 {
		return b
	}
	b.spec.Interval = metav1.Duration{Duration: duration}
	return b
}

// WithTimeout sets the timeout for the Helm release.
func (b *helmReleaseBuilder) WithTimeout(timeout time.Duration) *helmReleaseBuilder {
	if timeout <= 0 {
		return b
	}
	b.spec.Timeout = &metav1.Duration{Duration: timeout}
	return b
}

// WithValues sets the values for the Helm release.
func (b *helmReleaseBuilder) WithValues(byteValues []byte) *helmReleaseBuilder {
	if byteValues == nil {
		return b
	}
	b.spec.Values = &v1.JSON{Raw: byteValues}
	return b
}

// WithValuesFrom sets the values references for the Helm release. Only secret references are supported on the plugin side
func (b *helmReleaseBuilder) WithValuesFrom(ref []helmcontroller.ValuesReference) *helmReleaseBuilder {
	if len(ref) == 0 {
		return b
	}
	b.spec.ValuesFrom = ref
	return b
}

// WithReleaseName sets the release name for the Helm release.
func (b *helmReleaseBuilder) WithReleaseName(name string) *helmReleaseBuilder {
	if name == "" {
		return b
	}
	b.spec.ReleaseName = name
	return b
}

// WithTargetNamespace sets the target namespace for the Helm release.
func (b *helmReleaseBuilder) WithTargetNamespace(namespace string) *helmReleaseBuilder {
	if namespace == "" {
		return b
	}
	b.spec.TargetNamespace = namespace
	return b
}

// WithInstall sets the installation configuration for the Helm release.
func (b *helmReleaseBuilder) WithInstall(install *helmcontroller.Install) *helmReleaseBuilder {
	if install == nil {
		install = &helmcontroller.Install{
			Timeout: &metav1.Duration{Duration: DefaultTimeout},
			Remediation: &helmcontroller.InstallRemediation{
				Retries: DefaultRetry,
			},
		}
	}
	b.spec.Install = install
	return b
}

// WithUpgrade sets the upgrade configuration for the Helm release.
func (b *helmReleaseBuilder) WithUpgrade(upgrade *helmcontroller.Upgrade) *helmReleaseBuilder {
	if upgrade == nil {
		upgrade = &helmcontroller.Upgrade{
			Timeout: &metav1.Duration{Duration: DefaultTimeout},
			Remediation: &helmcontroller.UpgradeRemediation{
				Retries: DefaultRetry,
			},
		}
	}
	b.spec.Upgrade = upgrade
	return b
}

// WithRollback sets the rollback configuration for the Helm release.
func (b *helmReleaseBuilder) WithRollback(rollback *helmcontroller.Rollback) *helmReleaseBuilder {
	if rollback == nil {
		rollback = &helmcontroller.Rollback{
			Timeout: &metav1.Duration{Duration: DefaultTimeout},
		}
	}
	b.spec.Rollback = rollback
	return b
}

// WithDriftDetection sets the drift detection configuration for the Helm release.
func (b *helmReleaseBuilder) WithDriftDetection(driftDetection *helmcontroller.DriftDetection) *helmReleaseBuilder {
	if driftDetection == nil {
		driftDetection = &helmcontroller.DriftDetection{
			Mode: helmcontroller.DriftDetectionEnabled,
		}
	}
	b.spec.DriftDetection = driftDetection
	return b
}

// WithTest sets the test configuration for the Helm release.
func (b *helmReleaseBuilder) WithTest(test *helmcontroller.Test) *helmReleaseBuilder {
	if test == nil {
		test = &helmcontroller.Test{
			Enable:  true,
			Timeout: &metav1.Duration{Duration: DefaultTimeout},
		}
	}
	b.spec.Test = test
	return b
}

// WithUninstall sets the uninstallation configuration for the Helm release.
func (b *helmReleaseBuilder) WithUninstall(uninstall *helmcontroller.Uninstall) *helmReleaseBuilder {
	if uninstall == nil {
		uninstall = &helmcontroller.Uninstall{
			Timeout: &metav1.Duration{Duration: DefaultTimeout},
		}
	}
	b.spec.Uninstall = uninstall
	return b
}

// WithDependsOn sets the dependencies for the Helm release.
func (b *helmReleaseBuilder) WithDependsOn(dependencies []fluxmeta.NamespacedObjectReference) *helmReleaseBuilder {
	if len(dependencies) == 0 {
		return b
	}
	b.spec.DependsOn = dependencies
	return b
}

// WithKubeConfig sets the kubeconfig reference for the Helm release.
func (b *helmReleaseBuilder) WithKubeConfig(kc fluxmeta.SecretKeyReference) *helmReleaseBuilder {
	if kc == (fluxmeta.SecretKeyReference{}) {
		return b
	}
	b.spec.KubeConfig.SecretRef = kc
	return b
}

// Build validates the HelmRelease and returns it.
func (b *helmReleaseBuilder) Build() (helmcontroller.HelmReleaseSpec, error) {
	if b.spec.Chart.Spec.Chart == "" {
		return helmcontroller.HelmReleaseSpec{}, errors.New("chart name is required")
	}

	if b.spec.Chart.Spec.Version == "" {
		return helmcontroller.HelmReleaseSpec{}, errors.New("chart version is required")
	}
	return b.spec, nil
}
