// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	"github.com/cloudoperators/greenhouse/api/v1alpha2"
)

// ConvertTo converts this version to the Hub version.
func (p *Plugin) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha2.Plugin) //nolint:errcheck

	// Convert the PluginDefinitionRef.
	dst.Spec.PluginDefinitionRef = greenhousemetav1alpha1.PluginDefinitionReference{
		Name:      p.Spec.PluginDefinition,
		Namespace: p.Spec.PluginDefinitionNamespace,
	}

	// Rote conversion.

	// ObjectMeta
	dst.ObjectMeta = p.ObjectMeta

	// Spec
	dst.Spec.DisplayName = p.Spec.DisplayName
	dst.Spec.OptionValues = p.Spec.OptionValues
	dst.Spec.ClusterName = p.Spec.ClusterName
	dst.Spec.ReleaseNamespace = p.Spec.ReleaseNamespace
	dst.Spec.ReleaseName = p.Spec.ReleaseName

	// Status
	dst.Status.HelmReleaseStatus = (*v1alpha2.HelmReleaseStatus)(p.Status.HelmReleaseStatus)
	dst.Status.Version = p.Status.Version
	dst.Status.HelmChart = (*v1alpha2.HelmChartReference)(p.Status.HelmChart)
	dst.Status.UIApplication = (*v1alpha2.UIApplicationReference)(p.Status.UIApplication)
	dst.Status.Weight = p.Status.Weight
	dst.Status.Description = p.Status.Description
	dstExposedServices := make(map[string]v1alpha2.Service, len(p.Status.ExposedServices))
	for i, v := range p.Status.ExposedServices {
		dstExposedServices[i] = v1alpha2.Service(v)
	}
	dst.Status.ExposedServices = dstExposedServices
	dst.Status.StatusConditions = p.Status.StatusConditions

	return nil
}

// ConvertFrom converts from the Hub version to this version.
func (p *Plugin) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha2.Plugin) //nolint:errcheck

	// Convert the PluginDefinitionRef.
	p.Spec.PluginDefinition = src.Spec.PluginDefinitionRef.Name
	p.Spec.PluginDefinitionNamespace = src.Spec.PluginDefinitionRef.Namespace

	// Rote conversion.

	// ObjectMeta
	p.ObjectMeta = src.ObjectMeta

	// Spec
	p.Spec.DisplayName = src.Spec.DisplayName
	p.Spec.OptionValues = src.Spec.OptionValues
	p.Spec.ClusterName = src.Spec.ClusterName
	p.Spec.ReleaseNamespace = src.Spec.ReleaseNamespace
	p.Spec.ReleaseName = src.Spec.ReleaseName

	// Status
	p.Status.HelmReleaseStatus = (*HelmReleaseStatus)(src.Status.HelmReleaseStatus)
	p.Status.Version = src.Status.Version
	p.Status.HelmChart = (*HelmChartReference)(src.Status.HelmChart)
	p.Status.UIApplication = (*UIApplicationReference)(src.Status.UIApplication)
	p.Status.Weight = src.Status.Weight
	p.Status.Description = src.Status.Description
	dstExposedServices := make(map[string]Service, len(src.Status.ExposedServices))
	for i, v := range src.Status.ExposedServices {
		dstExposedServices[i] = Service(v)
	}
	p.Status.ExposedServices = dstExposedServices
	p.Status.StatusConditions = src.Status.StatusConditions

	return nil
}
