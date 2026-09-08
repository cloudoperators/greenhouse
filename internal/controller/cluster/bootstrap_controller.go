// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"sync/atomic"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	clusterphases "github.com/cloudoperators/greenhouse/internal/controller/cluster/phases"
	"github.com/cloudoperators/greenhouse/internal/features"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

type BootstrapReconciler struct {
	client.Client
	recorder                events.EventRecorder
	WorkloadIdentityEnabled bool
	FeatureFlagsName        string
	FeatureFlagsNamespace   string
	workloadIdentityEnabled atomic.Bool
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusters/finalizers,verbs=update
//+kubebuilder:rbac:groups="events.k8s.io",resources=events,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;update;patch;delete
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;update;patch;create
//+kubebuilder:rbac:groups="",resources=serviceaccounts/token,verbs=create
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;update;patch;create

// SetupWithManager sets up the controller with the Manager.
func (r *BootstrapReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorder(name)
	r.workloadIdentityEnabled.Store(r.WorkloadIdentityEnabled)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&corev1.Secret{}, builder.WithPredicates(
			clientutil.PredicateFilterBySecretTypes(greenhouseapis.SecretTypeKubeConfig, greenhouseapis.SecretTypeOIDCConfig),
		)).
		// Watch clusters and enqueue its secret.
		Watches(&greenhousev1alpha1.Cluster{}, handler.EnqueueRequestsFromMapFunc(enqueueSecretForCluster)).
		// Watch the feature flags ConfigMap to hot-reload the workload identity feature gate.
		Watches(&corev1.ConfigMap{}, handler.EnqueueRequestsFromMapFunc(r.reloadFeatureFlags), builder.WithPredicates(
			clientutil.PredicateHasLabelWithValue(greenhouseapis.LabelKeyFeatureFlags, "true"),
		)).
		// (WI feature gate) Owns the workload identity ConfigMap rendered for OIDC clusters.
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

func (r *BootstrapReconciler) reloadFeatureFlags(ctx context.Context, _ client.Object) []ctrl.Request {
	featureFlags, err := features.NewFeatures(ctx, r.Client, r.FeatureFlagsName, r.FeatureFlagsNamespace)
	if err != nil {
		return nil
	}
	r.workloadIdentityEnabled.Store(featureFlags.IsWorkloadIdentityEnabled())
	return nil
}

func (r *BootstrapReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return lifecycle.ReconcileObject(ctx, r.Client, req.NamespacedName, &corev1.Secret{}, r)
}

func (r *BootstrapReconciler) EnsureCreated(ctx context.Context, obj client.Object) (ctrl.Result, error) {
	p := &clusterphases.BootstrapPhase{
		Client:                  r.Client,
		Scheme:                  r.Scheme(),
		Secret:                  obj.(*corev1.Secret),
		WorkloadIdentityEnabled: r.workloadIdentityEnabled.Load(),
	}
	result, _, err := lifecycle.ExecuteSubRoutine(ctx, p.EnsureCreatePhases())
	return result, err
}

func (r *BootstrapReconciler) EnsureDeleted(ctx context.Context, obj client.Object) (ctrl.Result, error) {
	p := &clusterphases.BootstrapPhase{
		Client:                  r.Client,
		Scheme:                  r.Scheme(),
		Secret:                  obj.(*corev1.Secret),
		WorkloadIdentityEnabled: r.workloadIdentityEnabled.Load(),
	}
	result, _, err := lifecycle.ExecuteSubRoutine(ctx, p.EnsureDeletePhases())
	return result, err
}

func enqueueSecretForCluster(_ context.Context, o client.Object) []ctrl.Request {
	cluster, ok := o.(*greenhousev1alpha1.Cluster)
	if !ok {
		return nil
	}
	return []ctrl.Request{{NamespacedName: types.NamespacedName{Namespace: cluster.GetNamespace(), Name: cluster.GetSecretName()}}}
}
