// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"
	"testing"

	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/flux"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

type noopRecorder struct{}

func (noopRecorder) Eventf(_, _ runtime.Object, _, _, _, _ string, _ ...any) {}

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, greenhousev1alpha1.AddToScheme(scheme))
	require.NoError(t, sourcev1.AddToScheme(scheme))
	return scheme
}

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

func TestEnsureHelmRepository(t *testing.T) {
	ctx := context.Background()
	pd := testPluginDefinition()
	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(pd).Build()
	p := &Phase{Client: c, PluginDef: pd, NamespaceName: testNamespace, Recorder: noopRecorder{}}

	res, err := p.ensureHelmRepository()(ctx)

	require.NoError(t, err)
	require.Equal(t, lifecycle.Continue(), res)

	// HelmRepository should exist with the right URL.
	repo := &sourcev1.HelmRepository{}
	require.NoError(t, c.Get(ctx, client.ObjectKey{Name: flux.ChartURLToName(testHelmRepo), Namespace: testNamespace}, repo))
	require.Equal(t, testHelmRepo, repo.Spec.URL)
	// helmRepo field on Phase is populated for the next subroutine.
	require.NotNil(t, p.helmRepo)
}
