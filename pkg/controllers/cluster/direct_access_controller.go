// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
)

const serviceAccountName = "greenhouse"

// DirectAccessReconciler reconciles a Cluster object with accessMode=direct set.
type DirectAccessReconciler struct {
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
func (r *DirectAccessReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
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

func (r *DirectAccessReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var cluster = new(greenhousev1alpha1.Cluster)
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if cluster.Spec.AccessMode != greenhousev1alpha1.ClusterAccessModeDirect {
		return ctrl.Result{}, nil
	}

	// Cleanup logic
	if cluster.DeletionTimestamp != nil && controllerutil.ContainsFinalizer(cluster, greenhouseapis.FinalizerCleanupCluster) {
		// TODO: Delete the pluginDefinitions first then the rest of the resources.
		var kubeConfigSecret = new(corev1.Secret)
		if err := r.Client.Get(ctx, types.NamespacedName{Namespace: cluster.GetNamespace(), Name: cluster.GetSecretName()}, kubeConfigSecret); err != nil {
			return ctrl.Result{}, err
		}
		restClientGetter, err := clientutil.NewRestClientGetterFromSecret(kubeConfigSecret, cluster.Namespace)
		if err != nil {
			return ctrl.Result{}, err
		}
		remoteClient, err := clientutil.NewK8sClientFromRestClientGetter(restClientGetter)
		if err != nil {
			return ctrl.Result{}, err
		}
		// Delete namespace in remote cluster before the secret.
		// All remote resources are bound by owner-reference to the namespace
		if err := deleteNamespaceInRemoteCluster(ctx, remoteClient, cluster); err != nil {
			return ctrl.Result{}, err
		}
		// A simple Delete won't do. The logic should take into consideration the order, that only a portion of the resources have been deleted, etc.
		err = clientutil.RemoveFinalizer(ctx, r.Client, cluster, greenhouseapis.FinalizerCleanupCluster)
		return ctrl.Result{}, err
	}

	// Add finalizer before starting any work.
	if err := clientutil.EnsureFinalizer(ctx, r.Client, cluster, greenhouseapis.FinalizerCleanupCluster); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var clusterSecret = new(corev1.Secret)
	if err := r.Get(ctx, types.NamespacedName{Name: cluster.GetSecretName(), Namespace: cluster.GetNamespace()}, clusterSecret); err != nil {
		return ctrl.Result{}, err
	}

	restClientGetter, err := clientutil.NewRestClientGetterFromSecret(clusterSecret, cluster.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	k8sClientForRemoteCluster, err := clientutil.NewK8sClientFromRestClientGetter(restClientGetter)
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := reconcileNamespaceInRemoteCluster(ctx, k8sClientForRemoteCluster, cluster); err != nil {
		return ctrl.Result{}, err
	}
	if err := reconcileServiceAccountInRemoteCluster(ctx, k8sClientForRemoteCluster, cluster); err != nil {
		return ctrl.Result{}, err
	}
	if err := reconcileClusterRoleBindingInRemoteCluster(ctx, k8sClientForRemoteCluster, cluster); err != nil {
		return ctrl.Result{}, err
	}

	var tokenRequestor = &tokenHelper{
		Client:                             r.Client,
		RemoteClusterBearerTokenValidity:   r.RemoteClusterBearerTokenValidity,
		RenewRemoteClusterBearerTokenAfter: r.RenewRemoteClusterBearerTokenAfter,
	}
	if err := tokenRequestor.ReconcileServiceAccountToken(ctx, restClientGetter, cluster); err != nil {
		return ctrl.Result{}, err
	}

	if err := reconcileRemoteAPIServerVersion(ctx, restClientGetter, r.Client, cluster); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: defaultRequeueInterval}, nil
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
