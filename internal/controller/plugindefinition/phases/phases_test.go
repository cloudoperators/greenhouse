// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"
	"testing"

	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/common"
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
func testClusterPluginDefinition() *greenhousev1alpha1.ClusterPluginDefinition {
	cpd := &greenhousev1alpha1.ClusterPluginDefinition{}
	cpd.Name = "test-cluster-plugin"
	cpd.Spec.HelmChart = &greenhousev1alpha1.HelmChartReference{
		Name:       testHelmChart,
		Repository: testHelmRepo,
		Version:    testChartVersion,
	}
	return cpd
}

func TestEnsureHelmRepository(t *testing.T) {
	tests := []struct {
		name      string
		pluginDef func() common.GenericPluginDefinition
		namespace string
	}{
		{
			name:      "PluginDefinition",
			pluginDef: func() common.GenericPluginDefinition { return testPluginDefinition() },
			namespace: testNamespace,
		},
		{
			name:      "ClusterPluginDefinition",
			pluginDef: func() common.GenericPluginDefinition { return testClusterPluginDefinition() },
			namespace: flux.HelmRepositoryDefaultNamespace,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			pd := tt.pluginDef()
			c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(pd).Build()
			p := &Phase{Client: c, PluginDef: pd, NamespaceName: tt.namespace, Recorder: noopRecorder{}}

			res, err := p.ensureHelmRepository()(ctx)

			require.NoError(t, err)
			require.Equal(t, lifecycle.Continue(), res)

			repo := &sourcev1.HelmRepository{}
			require.NoError(t, c.Get(ctx, client.ObjectKey{Name: flux.ChartURLToName(testHelmRepo), Namespace: tt.namespace}, repo))
			require.Equal(t, testHelmRepo, repo.Spec.URL)
			require.NotNil(t, p.helmRepo)
		})
	}
}

func TestEnsureChartReplication(t *testing.T) {
	tests := []struct {
		name                string
		ociMirroringEnabled bool
		pluginDef           func() *greenhousev1alpha1.PluginDefinition
	}{
		{
			name:                "disabled",
			ociMirroringEnabled: false,
			pluginDef:           testPluginDefinition,
		},
		{
			name:                "non-OCI repository",
			ociMirroringEnabled: true,
			pluginDef:           testPluginDefinition, // uses https://, not oci://
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pd := tt.pluginDef()
			p := &Phase{PluginDef: pd, OCIMirroringEnabled: tt.ociMirroringEnabled}

			res, err := p.ensureChartReplication()(context.Background())

			require.NoError(t, err)
			require.Equal(t, lifecycle.Continue(), res)
			condition := pd.Status.GetConditionByType(greenhousev1alpha1.OCIReplicationReadyCondition)
			require.NotNil(t, condition)
			require.Equal(t, greenhousev1alpha1.OCIReplicationNotConfiguredReason, condition.Reason)
		})
	}
}

const (
	testRegistry  = "keppel.eu-de-1.cloud.sap"
	testChartName = "ccloud-ghcr-io-mirror/cloudoperators/greenhouse-extensions/charts/audit-logs"
	testVersion   = "0.0.21"
)

func replicatedPluginDefinition() *greenhousev1alpha1.PluginDefinition {
	return &greenhousev1alpha1.PluginDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: "audit-logs-compute", Namespace: "sci"},
		Status: greenhousev1alpha1.PluginDefinitionStatus{
			LastSyncedArtifact: &greenhousev1alpha1.LastSyncedArtifact{
				Registry:          testRegistry,
				ChartName:         testChartName,
				Version:           testVersion,
				ReplicationStatus: greenhousev1alpha1.ReplicationStatusReplicated,
			},
		},
	}
}

func TestShouldSkipChartReplication(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(*greenhousev1alpha1.PluginDefinition)
		expectSkip bool
	}{
		{
			name:       "skips when the recorded artifact matches the desired chart",
			mutate:     func(*greenhousev1alpha1.PluginDefinition) {},
			expectSkip: true,
		},
		{
			name:       "replicates when nothing was recorded yet",
			mutate:     func(pd *greenhousev1alpha1.PluginDefinition) { pd.Status.LastSyncedArtifact = nil },
			expectSkip: false,
		},
		{
			name:       "replicates when the recorded version differs",
			mutate:     func(pd *greenhousev1alpha1.PluginDefinition) { pd.Status.LastSyncedArtifact.Version = "0.0.20" },
			expectSkip: false,
		},
		{
			name: "replicates when a reconcile is requested via annotation",
			mutate: func(pd *greenhousev1alpha1.PluginDefinition) {
				pd.SetAnnotations(map[string]string{lifecycle.ReconcileAnnotation: "2026-08-25T13:34:13Z"})
			},
			expectSkip: false,
		},
		{
			name: "skips again once the annotation is removed",
			mutate: func(pd *greenhousev1alpha1.PluginDefinition) {
				pd.SetAnnotations(map[string]string{"unrelated": "value"})
			},
			expectSkip: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pd := replicatedPluginDefinition()
			tt.mutate(pd)
			require.Equal(t, tt.expectSkip, ShouldSkipChartReplication(pd, testRegistry, testChartName, testVersion))
		})
	}
}

func TestCreateUpdateHelmChart(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		expectValue string
	}{
		{
			name:        "propagates the reconcile request value to the HelmChart",
			annotations: map[string]string{lifecycle.ReconcileAnnotation: "2026-08-25T13:34:13Z"},
			expectValue: "2026-08-25T13:34:13Z",
		},
		{
			name:        "leaves the HelmChart untouched when no reconcile was requested",
			annotations: nil,
			expectValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pd := replicatedPluginDefinition()
			pd.SetAnnotations(tt.annotations)
			pd.Spec.HelmChart = &greenhousev1alpha1.HelmChartReference{
				Name:       "audit-logs",
				Repository: "oci://" + testRegistry + "/ccloud-ghcr-io-mirror/cloudoperators/greenhouse-extensions/charts",
				Version:    testVersion,
			}
			c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(pd).Build()
			p := &Phase{Client: c, Recorder: noopRecorder{}, PluginDef: pd, NamespaceName: pd.Namespace}

			helmChart, err := p.CreateUpdateHelmChart(context.Background(), &sourcev1.HelmRepository{
				ObjectMeta: metav1.ObjectMeta{Name: "keppel-repo", Namespace: pd.Namespace},
			})
			require.NoError(t, err)
			require.Equal(t, tt.expectValue, helmChart.GetAnnotations()[fluxmeta.ReconcileRequestAnnotation])
		})
	}
}

func TestEnsureHelmChart(t *testing.T) {
	tests := []struct {
		name      string
		pluginDef func() common.GenericPluginDefinition
		namespace string
	}{
		{
			name:      "PluginDefinition",
			pluginDef: func() common.GenericPluginDefinition { return testPluginDefinition() },
			namespace: testNamespace,
		},
		{
			name:      "ClusterPluginDefinition",
			pluginDef: func() common.GenericPluginDefinition { return testClusterPluginDefinition() },
			namespace: flux.HelmRepositoryDefaultNamespace,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			pd := tt.pluginDef()
			c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(pd).Build()
			repo := &sourcev1.HelmRepository{}
			repo.Name = flux.ChartURLToName(testHelmRepo)
			repo.Namespace = tt.namespace
			p := &Phase{Client: c, PluginDef: pd, NamespaceName: tt.namespace, Recorder: noopRecorder{}, helmRepo: repo}

			res, err := p.ensureHelmChart()(ctx)

			require.NoError(t, err)
			require.Equal(t, lifecycle.Continue(), res)

			helmChart := &sourcev1.HelmChart{}
			require.NoError(t, c.Get(ctx, client.ObjectKey{Name: pd.FluxHelmChartResourceName(), Namespace: tt.namespace}, helmChart))
			require.Equal(t, testHelmChart, helmChart.Spec.Chart)
			require.Equal(t, testChartVersion, helmChart.Spec.Version)

			// Without Flux source-controller, HelmChart has no Ready condition — status should be Unknown.
			conditions := pd.GetConditions()
			condition := conditions.GetConditionByType(greenhousev1alpha1.HelmChartReadyCondition)
			require.NotNil(t, condition)
			require.Equal(t, metav1.ConditionUnknown, condition.Status)
		})
	}
}

func TestEnsureOrphanedHelmChartsDeletedPluginDefinition(t *testing.T) {
	ctx := context.Background()
	pd := testPluginDefinition()
	isController := true

	// The orphaned chart has the old version name; the current chart has the new version name.
	orphanedChart := &sourcev1.HelmChart{}
	orphanedChart.Name = pd.Name + "-0.9.0"
	orphanedChart.Namespace = testNamespace
	orphanedChart.Labels = map[string]string{greenhouseapis.LabelKeyPluginDefinition: pd.Name}
	orphanedChart.OwnerReferences = []metav1.OwnerReference{{UID: pd.UID, Controller: &isController}}

	currentChart := &sourcev1.HelmChart{}
	currentChart.Name = pd.FluxHelmChartResourceName()
	currentChart.Namespace = testNamespace
	currentChart.Labels = map[string]string{greenhouseapis.LabelKeyPluginDefinition: pd.Name}
	currentChart.OwnerReferences = []metav1.OwnerReference{{UID: pd.UID, Controller: &isController}}

	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(pd, orphanedChart, currentChart).Build()
	p := &Phase{Client: c, PluginDef: pd, NamespaceName: testNamespace, Recorder: noopRecorder{}}

	res, err := p.ensureOrphanedHelmChartsDeleted()(ctx)

	require.NoError(t, err)
	require.Equal(t, lifecycle.Continue(), res)

	// Orphaned chart should be gone.
	err = c.Get(ctx, client.ObjectKey{Name: orphanedChart.Name, Namespace: testNamespace}, &sourcev1.HelmChart{})
	require.True(t, apierrors.IsNotFound(err), "orphaned HelmChart should be deleted")

	// Current chart should still exist.
	require.NoError(t, c.Get(ctx, client.ObjectKey{Name: currentChart.Name, Namespace: testNamespace}, &sourcev1.HelmChart{}))
}

func TestEnsureOrphanedHelmChartsDeletedClusterPluginDefinition(t *testing.T) {
	ctx := context.Background()
	cpd := testClusterPluginDefinition()
	ns := flux.HelmRepositoryDefaultNamespace
	isController := true

	orphanedChart := &sourcev1.HelmChart{}
	orphanedChart.Name = cpd.Name + "-0.9.0"
	orphanedChart.Namespace = ns
	orphanedChart.Labels = map[string]string{greenhouseapis.LabelKeyPluginDefinition: cpd.Name}
	orphanedChart.OwnerReferences = []metav1.OwnerReference{{UID: cpd.UID, Controller: &isController}}

	currentChart := &sourcev1.HelmChart{}
	currentChart.Name = cpd.FluxHelmChartResourceName()
	currentChart.Namespace = ns
	currentChart.Labels = map[string]string{greenhouseapis.LabelKeyPluginDefinition: cpd.Name}
	currentChart.OwnerReferences = []metav1.OwnerReference{{UID: cpd.UID, Controller: &isController}}

	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(cpd, orphanedChart, currentChart).Build()
	p := &Phase{Client: c, PluginDef: cpd, NamespaceName: ns, Recorder: noopRecorder{}}

	res, err := p.ensureOrphanedHelmChartsDeleted()(ctx)

	require.NoError(t, err)
	require.Equal(t, lifecycle.Continue(), res)

	err = c.Get(ctx, client.ObjectKey{Name: orphanedChart.Name, Namespace: ns}, &sourcev1.HelmChart{})
	require.True(t, apierrors.IsNotFound(err), "orphaned HelmChart should be deleted")

	require.NoError(t, c.Get(ctx, client.ObjectKey{Name: currentChart.Name, Namespace: ns}, &sourcev1.HelmChart{}))
}
