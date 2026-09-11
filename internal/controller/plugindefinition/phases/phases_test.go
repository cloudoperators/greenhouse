// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"testing"

	"github.com/stretchr/testify/require"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

const (
	testNamespace    = "test-org"
	testHelmRepo     = "https://my.dummy.io"
	testHelmChart    = "mychart"
	testChartVersion = "1.0.0"
)

func testPluginDefinition() *greenhousev1alpha1.PluginDefinition {
	pd := &greenhousev1alpha1.PluginDefinition{}
	pd.Name = "test-plugin"
	pd.Namespace = testNamespace
	pd.Spec.HelmChart = &greenhousev1alpha1.HelmChartReference{
		Name:       testHelmChart,
		Repository: testHelmRepo,
		Version:    testChartVersion,
	}
	return pd
}

func TestEnsureCreatePhases(t *testing.T) {
	p := &Phase{PluginDef: testPluginDefinition(), NamespaceName: testNamespace}
	require.Len(t, p.EnsureCreatePhases(), 4)
}

func TestEnsureDeletePhases(t *testing.T) {
	p := &Phase{PluginDef: testPluginDefinition(), NamespaceName: testNamespace}
	require.Empty(t, p.EnsureDeletePhases())
}
