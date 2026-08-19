// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	corev1 "k8s.io/api/core/v1"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	clusterphases "github.com/cloudoperators/greenhouse/internal/controller/cluster/phases"
	"github.com/cloudoperators/greenhouse/internal/util"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

// RemoteClusterReconciler reconciles a Cluster object with accessMode=direct set.
type RemoteClusterReconciler struct {
	client.Client
	recorder                           events.EventRecorder
	RemoteClusterBearerTokenValidity   time.Duration
	RenewRemoteClusterBearerTokenAfter time.Duration
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusters/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;update;patch;create
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;update;patch;create
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;update;patch;create;delete
//+kubebuilder:rbac:groups="events.k8s.io",resources=events,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterrolebindings,verbs=get;list;watch;update;patch;create

// SetupWithManager sets up the controller with the Manager.
func (r *RemoteClusterReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorder(name)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousev1alpha1.Cluster{}, builder.WithPredicates(
			clientutil.PredicateClusterByAccessMode(greenhousev1alpha1.ClusterAccessModeDirect),
		)).
		Watches(&corev1.Secret{}, handler.EnqueueRequestForOwner(mgr.GetScheme(), mgr.GetRESTMapper(), &greenhousev1alpha1.Cluster{})).
		WithOptions(controller.Options{MaxConcurrentReconciles: 5}).
		Complete(r)
}

func (r *RemoteClusterReconciler) GetEventRecorder() events.EventRecorder {
	return r.recorder
}

func (r *RemoteClusterReconciler) computeReady(ctx context.Context, resource lifecycle.RuntimeObject) {
	cluster := resource.(*greenhousev1alpha1.Cluster)

	kubeConfigValid := cluster.Status.GetConditionByType(greenhousev1alpha1.KubeConfigValid)
	if kubeConfigValid != nil && kubeConfigValid.IsFalse() {
		cluster.Status.DiscoveryCache = nil
		cluster.Status.Nodes = nil
		cluster.SetCondition(greenhousemetav1alpha1.UnknownCondition(
			greenhousev1alpha1.AllNodesReady, "", "kubeconfig not valid - cannot know node status",
		))
		cluster.SetCondition(greenhousemetav1alpha1.UnknownCondition(
			greenhousev1alpha1.PermissionsVerified, "", "kubeconfig not valid - cannot validate cluster access",
		))
		cluster.SetCondition(greenhousemetav1alpha1.FalseCondition(
			greenhousev1alpha1.PayloadSchedulable, "", "kubeconfig not valid - payloads cannot be scheduled",
		))
	}

	conditionsToAggregate := []greenhousemetav1alpha1.ConditionType{
		greenhousev1alpha1.KubeConfigValid,
		greenhousev1alpha1.PermissionsVerified,
	}
	switch cluster.Annotations[greenhouseapis.ClusterConnectivityAnnotation] {
	case greenhouseapis.ClusterConnectivityOIDC:
		cluster.RemoveCondition(greenhousev1alpha1.ManagedResourcesDeployed) // no managed resources for OIDC clusters
	default:
		conditionsToAggregate = append(conditionsToAggregate, greenhousev1alpha1.ManagedResourcesDeployed)
	}

	ready := greenhousemetav1alpha1.UnknownCondition(greenhousemetav1alpha1.ReadyCondition, "", "")
	allSet := true
	for _, ct := range conditionsToAggregate {
		c := cluster.Status.GetConditionByType(ct)
		if c == nil {
			allSet = false
			continue
		}
		if c.IsFalse() {
			ready.Status = metav1.ConditionFalse
			ready.Message = c.Message
			if ready.Message == "" {
				ready.Message = "cannot access cluster"
			}
			break
		}
	}
	if ready.Status != metav1.ConditionFalse && allSet {
		ready.Status = metav1.ConditionTrue
	}
	cluster.SetCondition(ready)

	ownerLabelCondition := util.ComputeOwnerLabelCondition(ctx, r.Client, cluster)
	cluster.SetCondition(ownerLabelCondition)
	util.UpdateOwnedByLabelMissingMetric(cluster, ownerLabelCondition.IsFalse())
}

func (r *RemoteClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return lifecycle.Reconcile(ctx, r.Client, req.NamespacedName, &greenhousev1alpha1.Cluster{}, r, r.computeReady)
}

func (r *RemoteClusterReconciler) EnsureCreated(ctx context.Context, resource lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	cluster := resource.(*greenhousev1alpha1.Cluster)
	defer UpdateClusterMetrics(cluster)

	clusterSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: cluster.GetSecretName(), Namespace: cluster.GetNamespace()}, clusterSecret); err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}

	restClientGetter, remoteClient, err := clusterphases.CreateRemoteClient(clusterSecret, cluster.GetNamespace())
	if err != nil {
		cluster.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.KubeConfigValid, "", err.Error()))
		return ctrl.Result{}, lifecycle.Failed, err
	}

	p := &clusterphases.Phase{
		Client:                             r.Client,
		Recorder:                           r.recorder,
		RemoteClusterBearerTokenValidity:   r.RemoteClusterBearerTokenValidity,
		RenewRemoteClusterBearerTokenAfter: r.RenewRemoteClusterBearerTokenAfter,
		ClusterSecret:                      clusterSecret,
		RestClientGetter:                   restClientGetter,
		RemoteClient:                       remoteClient,
	}

	return lifecycle.ExecuteSubRoutine(ctx, p.EnsureCreatePhases(cluster))
}

// EnsureDeleted handles the deletion / cleanup of cluster resource.
func (r *RemoteClusterReconciler) EnsureDeleted(ctx context.Context, resource lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	cluster := resource.(*greenhousev1alpha1.Cluster)

	c := cluster.Status.GetConditionByType(greenhousev1alpha1.KubeConfigValid)
	if c != nil && c.IsFalse() {
		deleteClusterMetrics(cluster)
		return ctrl.Result{}, lifecycle.Success, nil
	}

	clusterSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: cluster.GetNamespace(), Name: cluster.GetSecretName()}, clusterSecret); err != nil {
		if client.IgnoreNotFound(err) == nil {
			deleteClusterMetrics(cluster)
			return ctrl.Result{}, lifecycle.Success, nil
		}
		return ctrl.Result{}, lifecycle.Failed, err
	}

	p := &clusterphases.Phase{
		Client:        r.Client,
		ClusterSecret: clusterSecret,
	}

	if clusterSecret.Type != greenhouseapis.SecretTypeOIDCConfig {
		restClientGetter, remoteClient, err := clusterphases.CreateRemoteClient(clusterSecret, cluster.GetNamespace())
		if err != nil {
			return ctrl.Result{}, lifecycle.Failed, err
		}
		p.RestClientGetter = restClientGetter
		p.RemoteClient = remoteClient
	}

	result, reconcileResult, err := lifecycle.ExecuteSubRoutine(ctx, p.EnsureDeletePhases(cluster))
	if reconcileResult == lifecycle.Success {
		deleteClusterMetrics(cluster)
	}
	return result, reconcileResult, err
}

func (r *RemoteClusterReconciler) EnsureSuspended(_ context.Context, _ lifecycle.RuntimeObject) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}
