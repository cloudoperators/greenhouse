// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"fmt"
	"time"

	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudoperators/greenhouse/internal/controller/cluster/utils"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"

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
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups="rbac",resources=clusterrolebindings,verbs=get;list;watch;update;patch;create

// SetupWithManager sets up the controller with the Manager.
func (r *RemoteClusterReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorder(name)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousev1alpha1.Cluster{}, builder.WithPredicates(
			clientutil.PredicateClusterByAccessMode(greenhousev1alpha1.ClusterAccessModeDirect),
		)).
		// Watch the secret owned by this cluster.
		Watches(&corev1.Secret{}, handler.EnqueueRequestForOwner(mgr.GetScheme(), mgr.GetRESTMapper(), &greenhousev1alpha1.Cluster{})).
		WithOptions(controller.Options{MaxConcurrentReconciles: 5}).
		Complete(r)
}

func (r *RemoteClusterReconciler) GetEventRecorder() events.EventRecorder {
	return r.recorder
}

func (r *RemoteClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return lifecycle.Reconcile(ctx, r.Client, req.NamespacedName, &greenhousev1alpha1.Cluster{}, r, r.setConditions())
}

func (r *RemoteClusterReconciler) EnsureCreated(ctx context.Context, resource lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	cluster := resource.(*greenhousev1alpha1.Cluster) //nolint:errcheck
	if cluster.Spec.AccessMode != greenhousev1alpha1.ClusterAccessModeDirect {
		return ctrl.Result{}, lifecycle.Failed, nil
	}
	// Deletion Schedule mechanism
	isScheduled, schedule, err := clientutil.ExtractDeletionSchedule(cluster.GetAnnotations())
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}
	if isScheduled && cluster.DeletionTimestamp == nil {
		if ok, err := clientutil.ShouldProceedDeletion(time.Now(), schedule); ok && err == nil {
			err = r.Delete(ctx, cluster)
			if err != nil {
				return ctrl.Result{}, lifecycle.Failed, err
			}
			return ctrl.Result{}, lifecycle.Success, nil
		}
	}
	defer UpdateClusterMetrics(cluster)
	clusterSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: cluster.GetSecretName(), Namespace: cluster.GetNamespace()}, clusterSecret); err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}

	restClientGetter, err := clientutil.NewRestClientGetterFromSecret(clusterSecret, cluster.Namespace)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}

	remoteClient, err := clientutil.NewK8sClientFromRestClientGetter(restClientGetter)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}

	// check token validity and renew if needed
	// for OIDC kubeconfig this needs to be done first before any other operations for OIDC clusters
	if clusterSecret.Type == greenhouseapis.SecretTypeOIDCConfig {
		if err := r.reconcileServiceAccountToken(ctx, restClientGetter, remoteClient, cluster, clusterSecret); err != nil {
			return ctrl.Result{}, lifecycle.Failed, err
		}
	}

	var crb *rbacv1.ClusterRoleBinding
	if clusterSecret.Type != greenhouseapis.SecretTypeOIDCConfig {
		// Create ClusterRoleBinding first so it can be added as an owner to namespace
		var err error
		crb, err = r.reconcileClusterRoleBindingInRemoteCluster(ctx, remoteClient, cluster)
		if err != nil {
			return ctrl.Result{}, lifecycle.Failed, err
		}
	}

	// Create the namespace in the remote cluster
	if err := r.reconcileNamespaceInRemoteCluster(ctx, remoteClient, cluster); err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}

	// Create greenhouse service account in managed namespace only for non-OIDC clusters
	// use crb as owner reference if applicable
	if clusterSecret.Type != greenhouseapis.SecretTypeOIDCConfig {
		if err := r.reconcileServiceAccountInRemoteCluster(ctx, remoteClient, crb, cluster); err != nil {
			return ctrl.Result{}, lifecycle.Failed, err
		}
	}
	// reconcile the service account token in the remote cluster
	// for OIDC this will early exit as it is already done above
	if err := r.reconcileServiceAccountToken(ctx, restClientGetter, remoteClient, cluster, clusterSecret); err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}
	return ctrl.Result{RequeueAfter: utils.DefaultRequeueInterval}, lifecycle.Success, nil
}

// reconcileClusterRoleBindingInRemoteCluster - creates or updates the cluster role binding in the remote cluster
func (r *RemoteClusterReconciler) reconcileClusterRoleBindingInRemoteCluster(ctx context.Context, k8sClient client.Client, cluster *greenhousev1alpha1.Cluster) (*rbacv1.ClusterRoleBinding, error) {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: utils.ServiceAccountName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      utils.ServiceAccountName,
				Namespace: cluster.GetNamespace(),
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     utils.CRoleKind,
			Name:     utils.CRoleRef,
			APIGroup: rbacv1.GroupName,
		},
	}

	result, err := clientutil.CreateOrPatch(ctx, k8sClient, clusterRoleBinding, func() error {
		return nil
	})
	if err != nil {
		return nil, err
	}
	switch result {
	// TODO: emit event on cluster
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created clusterRoleBinding", "cluster", clusterRoleBinding.Name)
	// TODO: emit event on cluster
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated clusterRoleBinding", "cluster", clusterRoleBinding.Name)
	}
	return clusterRoleBinding, nil
}

// reconcileNamespaceInRemoteCluster - creates or updates the namespace in the remote cluster
func (r *RemoteClusterReconciler) reconcileNamespaceInRemoteCluster(ctx context.Context, k8sClient client.Client, cluster *greenhousev1alpha1.Cluster) error {
	var namespace = new(corev1.Namespace)
	namespace.Name = cluster.GetNamespace()
	result, err := clientutil.CreateOrPatch(ctx, k8sClient, namespace, func() error {
		return nil
	})
	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created namespace", "cluster", cluster.Name, "namespace", namespace.Name)
		// TODO: emit event on cluster
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated namespace", "cluster", cluster.Name, "namespace", namespace.Name)
		// TODO: emit event on cluster
	}
	return nil
}

// reconcileServiceAccountInRemoteCluster - creates or updates the service account in the remote cluster
func (r *RemoteClusterReconciler) reconcileServiceAccountInRemoteCluster(ctx context.Context, k8sClient client.Client, crb *rbacv1.ClusterRoleBinding, cluster *greenhousev1alpha1.Cluster) error {
	serviceAccount := utils.NewServiceAccount(utils.ServiceAccountName, cluster.GetNamespace())
	result, err := clientutil.CreateOrPatch(ctx, k8sClient, serviceAccount, func() error {
		if crb != nil {
			return controllerutil.SetOwnerReference(crb, serviceAccount, k8sClient.Scheme())
		}
		return nil
	})
	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created serviceAccount", "cluster", serviceAccount.Name)
		// TODO: emit event on cluster
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated serviceAccount", "cluster", serviceAccount.Name)
		// TODO: emit event on cluster
	}
	return nil
}

func (r *RemoteClusterReconciler) reconcileServiceAccountToken(
	ctx context.Context,
	restClientGetter *clientutil.RestClientGetter,
	remoteClient client.Client,
	cluster *greenhousev1alpha1.Cluster,
	clusterSecret *corev1.Secret,
) error {

	cluster.SetDefaultTokenValidityIfNeeded()
	t := &utils.TokenHelper{
		InClusterClient:                    r.Client,
		RemoteClusterClient:                remoteClient,
		RemoteClusterBearerTokenValidity:   time.Duration(cluster.Spec.KubeConfig.MaxTokenValidity) * time.Hour,
		RenewRemoteClusterBearerTokenAfter: r.RenewRemoteClusterBearerTokenAfter,
		SecretType:                         clusterSecret.Type,
	}
	tokenRequest, err := t.GenerateTokenRequest(ctx, restClientGetter, cluster, clusterSecret)
	if err != nil {
		log.FromContext(ctx).Error(err, "failed to generate token", "cluster", cluster.Name)
		return err
	}
	if tokenRequest == nil {
		// early return as the token is still valid and no new token is needed
		return nil
	}
	var generatedKubeConfig []byte
	switch cluster.Spec.AccessMode {
	case greenhousev1alpha1.ClusterAccessModeDirect:
		generatedKubeConfig, err = utils.GenerateNewClientKubeConfig(restClientGetter, tokenRequest.Status.Token, cluster)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown access mode %s", cluster.Spec.AccessMode)
	}

	kubeConfigSecret := &corev1.Secret{}
	if err := t.InClusterClient.Get(ctx, types.NamespacedName{Namespace: cluster.GetNamespace(), Name: cluster.GetName()}, kubeConfigSecret); err != nil {
		return err
	}
	result, err := clientutil.CreateOrPatch(ctx, t.InClusterClient, kubeConfigSecret, func() error {
		if clusterSecret.Type == greenhouseapis.SecretTypeOIDCConfig {
			kubeConfigSecret.Annotations[greenhouseapis.SecretOIDCConfigGeneratedOnAnnotation] = metav1.Now().Format(time.DateTime)
		}
		kubeConfigSecret.Data[greenhouseapis.GreenHouseKubeConfigKey] = generatedKubeConfig
		return nil
	})
	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created secret", "namespace", kubeConfigSecret.GetNamespace(), "name", kubeConfigSecret.GetName())
		// TODO: emit event on cluster
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated secret", "namespace", kubeConfigSecret.GetNamespace(), "name", kubeConfigSecret.GetName())
		// TODO: emit event on cluster
	}
	cluster.Status.BearerTokenExpirationTimestamp = tokenRequest.Status.ExpirationTimestamp

	return nil
}

// EnsureDeleted - handles the deletion / cleanup of cluster resource
func (r *RemoteClusterReconciler) EnsureDeleted(ctx context.Context, resource lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	cluster := resource.(*greenhousev1alpha1.Cluster) //nolint:errcheck
	c := cluster.Status.GetConditionByType(greenhousev1alpha1.KubeConfigValid)
	if c != nil && c.IsFalse() {
		return ctrl.Result{}, lifecycle.Success, nil
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
	if err := r.Get(ctx, types.NamespacedName{Namespace: cluster.GetNamespace(), Name: cluster.GetSecretName()}, kubeConfigSecret); err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}
	// early return if the cluster connectivity is via OIDC
	if kubeConfigSecret.Type == greenhouseapis.SecretTypeOIDCConfig {
		log.FromContext(ctx).Info("no resources to clean up", "secretType", kubeConfigSecret.Type, "cluster", cluster.Name)
		return ctrl.Result{}, lifecycle.Success, nil
	}
	restClientGetter, err := clientutil.NewRestClientGetterFromSecret(kubeConfigSecret, cluster.Namespace)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}
	remoteClient, err := clientutil.NewK8sClientFromRestClientGetter(restClientGetter)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}

	// deleting the cluster role binding in the remote cluster will delete
	// greenhouse service account and namespace due to owner reference
	if err := r.deleteClusterRoleBindingInRemoteCluster(ctx, remoteClient); err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}
	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *RemoteClusterReconciler) EnsureSuspended(_ context.Context, _ lifecycle.RuntimeObject) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

// deleteClusterRoleBindingInRemoteCluster - deletes the cluster role binding in the remote cluster
func (r *RemoteClusterReconciler) deleteClusterRoleBindingInRemoteCluster(ctx context.Context, k8sClient client.Client) error {
	crb := &rbacv1.ClusterRoleBinding{}
	err := k8sClient.Get(ctx, client.ObjectKey{Name: utils.ServiceAccountName}, crb)
	if err != nil {
		if apierrors.IsUnauthorized(err) || apierrors.IsNotFound(err) || apierrors.IsForbidden(err) {
			return nil
		}
		ctrl.LoggerFrom(ctx).V(5).Error(err, "err getting clusterRoleBinding")
		return err
	}
	err = k8sClient.Delete(ctx, crb)
	// ignore not found and forbidden errors
	if err != nil {
		if !apierrors.IsUnauthorized(err) && !apierrors.IsNotFound(err) && !apierrors.IsForbidden(err) {
			return err
		}
		ctrl.LoggerFrom(ctx).V(5).Error(err, "err deleting clusterRoleBinding")
	}
	return nil
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
