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

type HelmReleaseBuilder struct {
	spec helmcontroller.HelmReleaseSpec
}

func NewHelmReleaseSpecBuilder() *HelmReleaseBuilder {
	return &HelmReleaseBuilder{
		spec: helmcontroller.HelmReleaseSpec{
			Install: &helmcontroller.Install{
				Remediation: &helmcontroller.InstallRemediation{},
			},
			Upgrade: &helmcontroller.Upgrade{
				Remediation: &helmcontroller.UpgradeRemediation{},
			},
			DriftDetection: &helmcontroller.DriftDetection{},
			KubeConfig:     nil,
			Test:           &helmcontroller.Test{},
			Values:         &v1.JSON{},
		},
	}
}

// WithChart sets the chart specification for the Helm release.
func (b *HelmReleaseBuilder) WithChart(specRef helmcontroller.HelmChartTemplateSpec) *HelmReleaseBuilder {
	b.spec.Chart = &helmcontroller.HelmChartTemplate{
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
func (b *HelmReleaseBuilder) WithValuesFrom(ref []helmcontroller.ValuesReference) *HelmReleaseBuilder {
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

// WithTargetNamespace sets the target namespace for the Helm release.
func (b *HelmReleaseBuilder) WithTargetNamespace(namespace string) *HelmReleaseBuilder {
	if namespace == "" {
		return b
	}
	b.spec.TargetNamespace = namespace
	return b
}

// WithInstall sets the installation configuration for the Helm release.
func (b *HelmReleaseBuilder) WithInstall(install *helmcontroller.Install) *HelmReleaseBuilder {
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
func (b *HelmReleaseBuilder) WithUpgrade(upgrade *helmcontroller.Upgrade) *HelmReleaseBuilder {
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
func (b *HelmReleaseBuilder) WithRollback(rollback *helmcontroller.Rollback) *HelmReleaseBuilder {
	if rollback == nil {
		rollback = &helmcontroller.Rollback{
			Timeout: &metav1.Duration{Duration: DefaultTimeout},
		}
	}
	b.spec.Rollback = rollback
	return b
}

// WithDriftDetection sets the drift detection configuration for the Helm release.
func (b *HelmReleaseBuilder) WithDriftDetection(driftDetection *helmcontroller.DriftDetection) *HelmReleaseBuilder {
	if driftDetection == nil {
		driftDetection = &helmcontroller.DriftDetection{
			Mode: helmcontroller.DriftDetectionEnabled,
		}
	}
	b.spec.DriftDetection = driftDetection
	return b
}

// WithTest sets the test configuration for the Helm release.
func (b *HelmReleaseBuilder) WithTest(test *helmcontroller.Test) *HelmReleaseBuilder {
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
func (b *HelmReleaseBuilder) WithUninstall(uninstall *helmcontroller.Uninstall) *HelmReleaseBuilder {
	if uninstall == nil {
		uninstall = &helmcontroller.Uninstall{
			Timeout: &metav1.Duration{Duration: DefaultTimeout},
		}
	}
	b.spec.Uninstall = uninstall
	return b
}

// WithDependsOn sets the dependencies for the Helm release.
func (b *HelmReleaseBuilder) WithDependsOn(dependencies []fluxmeta.NamespacedObjectReference) *HelmReleaseBuilder {
	if len(dependencies) == 0 {
		return b
	}
	b.spec.DependsOn = dependencies
	return b
}

// WithKubeConfig sets the kubeconfig reference for the Helm release. If the fluxmeta.SecretKeyReference does not contain a name, the Plugin targets the central cluster and no specific kubeconfig is needed.
func (b *HelmReleaseBuilder) WithKubeConfig(kc fluxmeta.SecretKeyReference) *HelmReleaseBuilder {
	if kc.Name == "" { // Name is empty if Plugin is deployed in central cluster
		return b
	}
	b.spec.KubeConfig = &fluxmeta.KubeConfigReference{
		SecretRef: kc,
	}
	return b
}

// Build validates the HelmRelease and returns it.
func (b *HelmReleaseBuilder) Build() (helmcontroller.HelmReleaseSpec, error) {
	if b.spec.Chart.Spec.Chart == "" {
		return helmcontroller.HelmReleaseSpec{}, errors.New("chart name is required")
	}

	if b.spec.Chart.Spec.Version == "" {
		return helmcontroller.HelmReleaseSpec{}, errors.New("chart version is required")
	}
	return b.spec, nil
}
