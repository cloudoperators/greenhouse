// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/discovery"
)

// Override the default capabilities with the detected ones of the current cluster.
// FIXME: This is required as Helm detects a wrong kubernetes version.
func init() {
	cfg, err := newHelmAction(settings.RESTClientGetter(), corev1.NamespaceAll)
	if err != nil {
		return
	}
	caps, err := getCapabilities(cfg)
	if err != nil {
		return
	}
	chartutil.DefaultCapabilities = caps
}

func verifyKubeVersionIsCompatible(helmChart *chart.Chart, caps *chartutil.Capabilities) error {
	if helmChart.Metadata != nil && helmChart.Metadata.KubeVersion != "" {
		if !chartutil.IsCompatibleRange(helmChart.Metadata.KubeVersion, caps.KubeVersion.String()) {
			return errors.Errorf("chart requires kubeVersion: %s which is incompatible with Kubernetes %s", helmChart.Metadata.KubeVersion, caps.KubeVersion.String())
		}
	}
	return nil
}

func getCapabilities(cfg *action.Configuration) (*chartutil.Capabilities, error) {
	if cfg.Capabilities != nil {
		return cfg.Capabilities, nil
	}
	dc, err := cfg.RESTClientGetter.ToDiscoveryClient()
	if err != nil {
		return nil, errors.Wrap(err, "could not get Kubernetes discovery client")
	}
	// force a discovery cache invalidation to always fetch the latest server version/capabilities.
	dc.Invalidate()
	kubeVersion, err := dc.ServerVersion()
	if err != nil {
		return nil, errors.Wrap(err, "could not get server version from Kubernetes")
	}
	// Issue #6361:
	// Client-Go emits an error when an API service is registered but unimplemented.
	// We trap that error here and print a warning. But since the discovery client continues
	// building the API object, it is correctly populated with all valid APIs.
	// See https://github.com/kubernetes/kubernetes/issues/72051#issuecomment-521157642
	apiVersions, err := action.GetVersionSet(dc)
	if err != nil {
		if discovery.IsGroupDiscoveryFailedError(err) {
			cfg.Log("WARNING: The Kubernetes server has an orphaned API service. Server reports: %s", err)
			cfg.Log("WARNING: To fix this, kubectl delete apiservice <service-name>")
		} else {
			return nil, errors.Wrap(err, "could not get apiVersions from Kubernetes")
		}
	}

	cfg.Capabilities = &chartutil.Capabilities{
		APIVersions: apiVersions,
		KubeVersion: chartutil.KubeVersion{
			Version: kubeVersion.GitVersion,
			Major:   kubeVersion.Major,
			Minor:   kubeVersion.Minor,
		},
		HelmVersion: chartutil.DefaultCapabilities.HelmVersion,
	}
	return cfg.Capabilities, nil
}
