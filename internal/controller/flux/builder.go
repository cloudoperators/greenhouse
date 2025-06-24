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
	defaultInterval = 5 * time.Minute
	defaultTimeout  = 5 * time.Minute
	defaultRetry    = 3
)

type HelmReleaseBuilder interface {
	New(name, namespace string) *helmReleaseBuilder
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
	Build() (*helmcontroller.HelmRelease, error)
}

type helmReleaseBuilder struct {
	hr *helmcontroller.HelmRelease
}

func NewHelmReleaseBuilder() HelmReleaseBuilder {
	return &helmReleaseBuilder{}
}

// New creates a new HelmReleaseBuilder with the specified name and namespace.
func (b *helmReleaseBuilder) New(name, namespace string) *helmReleaseBuilder {
	return &helmReleaseBuilder{
		hr: &helmcontroller.HelmRelease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: helmcontroller.HelmReleaseSpec{
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
		},
	}
}

// WithChart sets the chart specification for the Helm release.
func (b *helmReleaseBuilder) WithChart(specRef helmcontroller.HelmChartTemplateSpec) *helmReleaseBuilder {
	b.hr.Spec.Chart = &helmcontroller.HelmChartTemplate{
		Spec: specRef,
	}
	return b
}

// WithMaxHistory sets the maximum history for the Helm release.
func (b *helmReleaseBuilder) WithMaxHistory(num int) *helmReleaseBuilder {
	if num < 0 {
		return b
	}
	b.hr.Spec.MaxHistory = ptr.To[int](num)
	return b
}

// WithInterval sets the interval for the Helm release.
func (b *helmReleaseBuilder) WithInterval(duration time.Duration) *helmReleaseBuilder {
	if duration <= 0 {
		return b
	}
	b.hr.Spec.Interval = metav1.Duration{Duration: duration}
	return b
}

// WithTimeout sets the timeout for the Helm release.
func (b *helmReleaseBuilder) WithTimeout(timeout time.Duration) *helmReleaseBuilder {
	if timeout <= 0 {
		return b
	}
	b.hr.Spec.Timeout = &metav1.Duration{Duration: timeout}
	return b
}

// WithValues sets the values for the Helm release.
func (b *helmReleaseBuilder) WithValues(byteValues []byte) *helmReleaseBuilder {
	if byteValues == nil {
		return b
	}
	b.hr.Spec.Values = &v1.JSON{Raw: byteValues}
	return b
}

// WithValuesFrom sets the values references for the Helm release. Only secret references are supported on the plugin side
func (b *helmReleaseBuilder) WithValuesFrom(ref []helmcontroller.ValuesReference) *helmReleaseBuilder {
	if len(ref) == 0 {
		return b
	}
	b.hr.Spec.ValuesFrom = ref
	return b
}

// WithReleaseName sets the release name for the Helm release.
func (b *helmReleaseBuilder) WithReleaseName(name string) *helmReleaseBuilder {
	if name == "" {
		return b
	}
	b.hr.Spec.ReleaseName = name
	return b
}

// WithTargetNamespace sets the target namespace for the Helm release.
func (b *helmReleaseBuilder) WithTargetNamespace(namespace string) *helmReleaseBuilder {
	if namespace == "" {
		return b
	}
	b.hr.Spec.TargetNamespace = namespace
	return b
}

// WithInstall sets the installation configuration for the Helm release.
func (b *helmReleaseBuilder) WithInstall(install *helmcontroller.Install) *helmReleaseBuilder {
	if install == nil {
		install = &helmcontroller.Install{
			Timeout: &metav1.Duration{Duration: defaultTimeout},
			Remediation: &helmcontroller.InstallRemediation{
				Retries: defaultRetry,
			},
		}
	}
	b.hr.Spec.Install = install
	return b
}

// WithUpgrade sets the upgrade configuration for the Helm release.
func (b *helmReleaseBuilder) WithUpgrade(upgrade *helmcontroller.Upgrade) *helmReleaseBuilder {
	if upgrade == nil {
		upgrade = &helmcontroller.Upgrade{
			Timeout: &metav1.Duration{Duration: defaultTimeout},
			Remediation: &helmcontroller.UpgradeRemediation{
				Retries: defaultRetry,
			},
		}
	}
	b.hr.Spec.Upgrade = upgrade
	return b
}

// WithRollback sets the rollback configuration for the Helm release.
func (b *helmReleaseBuilder) WithRollback(rollback *helmcontroller.Rollback) *helmReleaseBuilder {
	if rollback == nil {
		rollback = &helmcontroller.Rollback{
			Timeout: &metav1.Duration{Duration: defaultTimeout},
		}
	}
	b.hr.Spec.Rollback = rollback
	return b
}

// WithDriftDetection sets the drift detection configuration for the Helm release.
func (b *helmReleaseBuilder) WithDriftDetection(driftDetection *helmcontroller.DriftDetection) *helmReleaseBuilder {
	if driftDetection == nil {
		driftDetection = &helmcontroller.DriftDetection{
			Mode: helmcontroller.DriftDetectionEnabled,
		}
	}
	b.hr.Spec.DriftDetection = driftDetection
	return b
}

// WithTest sets the test configuration for the Helm release.
func (b *helmReleaseBuilder) WithTest(test *helmcontroller.Test) *helmReleaseBuilder {
	if test == nil {
		test = &helmcontroller.Test{
			Enable:  true,
			Timeout: &metav1.Duration{Duration: defaultTimeout},
		}
	}
	b.hr.Spec.Test = test
	return b
}

// WithUninstall sets the uninstallation configuration for the Helm release.
func (b *helmReleaseBuilder) WithUninstall(uninstall *helmcontroller.Uninstall) *helmReleaseBuilder {
	if uninstall == nil {
		uninstall = &helmcontroller.Uninstall{
			Timeout: &metav1.Duration{Duration: defaultTimeout},
		}
	}
	b.hr.Spec.Uninstall = uninstall
	return b
}

// WithDependsOn sets the dependencies for the Helm release.
func (b *helmReleaseBuilder) WithDependsOn(dependencies []fluxmeta.NamespacedObjectReference) *helmReleaseBuilder {
	if len(dependencies) == 0 {
		return b
	}
	b.hr.Spec.DependsOn = dependencies
	return b
}

// WithKubeConfig sets the kubeconfig reference for the Helm release.
func (b *helmReleaseBuilder) WithKubeConfig(kc fluxmeta.SecretKeyReference) *helmReleaseBuilder {
	if kc == (fluxmeta.SecretKeyReference{}) {
		return b
	}
	b.hr.Spec.KubeConfig.SecretRef = kc
	return b
}

// Build validates the HelmRelease and returns it.
func (b *helmReleaseBuilder) Build() (*helmcontroller.HelmRelease, error) {
	if b.hr.Spec.Chart.Spec.Chart == "" {
		return nil, errors.New("chart name is required")
	}

	if b.hr.Spec.Chart.Spec.Version == "" {
		return nil, errors.New("chart version is required")
	}

	return b.hr, nil
}
