// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"time"

	"github.com/cloudoperators/greenhouse/pkg/lifecycle"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
)

const serviceAccountName = "greenhouse"

// RemoteClusterReconciler reconciles a Cluster object with accessMode=direct set.
type RemoteClusterReconciler struct {
	client.Client
	recorder                           record.EventRecorder
	RemoteClusterBearerTokenValidity   time.Duration
	RenewRemoteClusterBearerTokenAfter time.Duration
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusters/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;update;patch;create
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;update;patch;create
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;update;patch;create;delete
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups="rbac",resources=clusterrolebindings,verbs=get;list;watch;update;patch;create

// SetupWithManager sets up the controller with the Manager.
func (r *RemoteClusterReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorderFor(name)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousev1alpha1.Cluster{}, builder.WithPredicates(
			clientutil.PredicateClusterByAccessMode(greenhousev1alpha1.ClusterAccessModeDirect),
		)).
		// Watch the secret owned by this cluster.
		Watches(&corev1.Secret{}, handler.EnqueueRequestForOwner(mgr.GetScheme(), mgr.GetRESTMapper(), &greenhousev1alpha1.Cluster{})).
		Complete(r)
}

func (r *RemoteClusterReconciler) GetEventRecorder() record.EventRecorder {
	return r.recorder
}

func (r *RemoteClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return lifecycle.Reconcile(ctx, r.Client, req.NamespacedName, &greenhousev1alpha1.Cluster{}, r, r.setConditions())
}

func (r *RemoteClusterReconciler) EnsureCreated(ctx context.Context, resource lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	cluster, ok := resource.(*greenhousev1alpha1.Cluster)
	if !ok {
		return ctrl.Result{}, lifecycle.Failed, errors.New("object is not of cluster type")
	}
	if cluster.Spec.AccessMode != greenhousev1alpha1.ClusterAccessModeDirect {
		return ctrl.Result{}, lifecycle.Success, nil
	}

	isScheduled, schedule, err := clientutil.ExtractDeletionSchedule(cluster.GetAnnotations())
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}
	if isScheduled && cluster.DeletionTimestamp == nil {
		if ok, err := clientutil.ShouldProceedDeletion(time.Now(), schedule); ok && err == nil {
			err = r.Client.Delete(ctx, cluster)
			if err != nil {
				return ctrl.Result{}, lifecycle.Failed, err
			}
			return ctrl.Result{}, lifecycle.Success, nil
		}
	}
	var clusterSecret = new(corev1.Secret)
	if err := r.Get(ctx, types.NamespacedName{Name: cluster.GetSecretName(), Namespace: cluster.GetNamespace()}, clusterSecret); err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}

	restClientGetter, err := clientutil.NewRestClientGetterFromSecret(clusterSecret, cluster.Namespace)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}

	k8sClientForRemoteCluster, err := clientutil.NewK8sClientFromRestClientGetter(restClientGetter)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}

	if err := reconcileNamespaceInRemoteCluster(ctx, k8sClientForRemoteCluster, cluster); err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}
	if err := reconcileServiceAccountInRemoteCluster(ctx, k8sClientForRemoteCluster, cluster); err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}
	if err := reconcileClusterRoleBindingInRemoteCluster(ctx, k8sClientForRemoteCluster, cluster); err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}

	var tokenRequestor = &tokenHelper{
		Client:                             r.Client,
		RemoteClusterBearerTokenValidity:   r.RemoteClusterBearerTokenValidity,
		RenewRemoteClusterBearerTokenAfter: r.RenewRemoteClusterBearerTokenAfter,
	}
	if err := tokenRequestor.ReconcileServiceAccountToken(ctx, restClientGetter, cluster); err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}

	if err := reconcileRemoteAPIServerVersion(ctx, restClientGetter, r.Client, cluster); err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}
	updateMetrics(cluster)
	return ctrl.Result{RequeueAfter: defaultRequeueInterval}, lifecycle.Success, nil
}

// EnsureDeleted - handles the deletion / cleanup of cluster resource
func (r *RemoteClusterReconciler) EnsureDeleted(ctx context.Context, resource lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	cluster, ok := resource.(*greenhousev1alpha1.Cluster)
	if !ok {
		return ctrl.Result{}, lifecycle.Failed, errors.New("object is not of cluster type")
	}
	// delete all plugins that are bound to this cluster
	deletionCount, err := deletePlugins(ctx, r.Client, cluster)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}
	if deletionCount > 0 {
		return ctrl.Result{RequeueAfter: 10 * time.Second}, lifecycle.Pending, nil
	}

	kubeConfigSecret := &corev1.Secret{}
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: cluster.GetNamespace(), Name: cluster.GetSecretName()}, kubeConfigSecret); err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}
	restClientGetter, err := clientutil.NewRestClientGetterFromSecret(kubeConfigSecret, cluster.Namespace)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}
	remoteClient, err := clientutil.NewK8sClientFromRestClientGetter(restClientGetter)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}
	// Delete namespace in remote cluster before the secret.
	// All remote resources are bound by owner-reference to the namespace
	if err := deleteNamespaceInRemoteCluster(ctx, remoteClient, cluster); err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}
	updateMetrics(cluster)
	return ctrl.Result{}, lifecycle.Success, nil
}

// generateNewClientKubeConfig generates a kubeconfig for the client to access the cluster from REST config coming from the secret
func generateNewClientKubeConfig(_ context.Context, restConfigGetter *clientutil.RestClientGetter, bearerToken string, cluster *greenhousev1alpha1.Cluster) ([]byte, error) {
	restConfig, err := restConfigGetter.ToRawKubeConfigLoader().ClientConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load kube clientConfig for cluster %s", cluster.GetName())
	}
	// TODO: replace overwrite with https://github.com/kubernetes/kubernetes/pull/119398 after 1.30 upgrade
	kubeConfigGenerator := &KubeConfigHelper{
		Host:        restConfig.Host,
		CAData:      restConfig.CAData,
		BearerToken: bearerToken,
		Username:    serviceAccountName,
		Namespace:   cluster.GetNamespace(),
	}
	kubeconfigByte, err := clientcmd.Write(kubeConfigGenerator.RestConfigToAPIConfig(cluster.Name))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate kubeconfig for cluster %s", cluster.GetName())
	}
	return kubeconfigByte, nil
}

func deletePlugins(ctx context.Context, c client.Client, cluster *greenhousev1alpha1.Cluster) (count int, err error) {
	pluginList := &greenhousev1alpha1.PluginList{}
	err = c.List(
		ctx,
		pluginList,
		client.InNamespace(cluster.GetNamespace()),
		client.MatchingLabels{greenhouseapis.LabelKeyCluster: cluster.GetName()},
	)
	if err != nil {
		return
	}
	for _, plugin := range pluginList.Items {
		if err = c.Delete(ctx, &plugin); client.IgnoreNotFound(err) != nil {
			return
		}
		count++
	}
	return
}
