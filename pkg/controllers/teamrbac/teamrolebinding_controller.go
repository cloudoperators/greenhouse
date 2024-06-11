// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package teamrbac

import (
	"context"
	"fmt"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
)

var exposedConditions = []greenhousev1alpha1.ConditionType{
	greenhousev1alpha1.ReadyCondition,
	greenhousev1alpha1.ClusterListEmpty,
}

// TeamRoleBindingReconciler reconciles a TeamRole object
type TeamRoleBindingReconciler struct {
	client.Client
	recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=teamrolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greenhouse.sap,resources=teamroles,verbs=get;list;watch;
//+kubebuilder:rbac:groups=greenhouse.sap,resources=teamrolebindings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=teamrolebindings/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// SetupWithManager sets up the controller with the Manager.
func (r *TeamRoleBindingReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorderFor(name)

	// index RoleBindings by the RoleRef field for faster lookups
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &greenhousev1alpha1.TeamRoleBinding{}, greenhouseapis.RolebindingRoleRefField, func(rawObj client.Object) []string {
		// Extract the TeamRole name from the TeamRoleBinding Spec, if one is provided
		teamRoleBinding, ok := rawObj.(*greenhousev1alpha1.TeamRoleBinding)
		if teamRoleBinding.Spec.TeamRoleRef == "" || !ok {
			return nil
		}
		return []string{teamRoleBinding.Spec.TeamRoleRef}
	}); err != nil {
		return err
	}

	// index TeamRoleBindings by the TeamRef field for faster lookups
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &greenhousev1alpha1.TeamRoleBinding{}, greenhouseapis.RolebindingTeamRefField, func(rawObj client.Object) []string {
		// Extract the TeamRole name from the TeamRoleBinding Spec, if one is provided
		teamRoleBinding, ok := rawObj.(*greenhousev1alpha1.TeamRoleBinding)
		if teamRoleBinding.Spec.TeamRef == "" || !ok {
			return nil
		}
		return []string{teamRoleBinding.Spec.TeamRef}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&greenhousev1alpha1.TeamRoleBinding{}).
		Watches(&greenhousev1alpha1.TeamRole{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueTeamRoleBindingsFor)).
		Watches(&greenhousev1alpha1.Team{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueTeamRoleBindingsFor)).
		// Reconcile TeamRoleBindings for all Cluster label changes in the same namespace
		Watches(&greenhousev1alpha1.Cluster{}, handler.EnqueueRequestsFromMapFunc(r.enqueueAllTeamRoleBindingsInNamespace),
			builder.WithPredicates(predicate.LabelChangedPredicate{})).
		Complete(r)
}

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *TeamRoleBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	trb := &greenhousev1alpha1.TeamRoleBinding{}
	if err := r.Get(ctx, req.NamespacedName, trb); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	_ = log.FromContext(ctx)

	trbStatus := initTeamRoleBindingStatus(trb)

	defer func() {
		if statusErr := r.setStatus(ctx, trb, trbStatus); statusErr != nil {
			log.FromContext(ctx).Error(statusErr, "Error setting status for TeamRoleBinding", "TeamRoleBinding", trb.GetName())
		}
	}()

	if trb.DeletionTimestamp != nil && controllerutil.ContainsFinalizer(trb, greenhouseapis.FinalizerCleanupTeamRoleBinding) {
		return r.deleteTeamRoleBinding(ctx, trb, trbStatus)
	}

	if err := clientutil.EnsureFinalizer(ctx, r.Client, trb, greenhouseapis.FinalizerCleanupTeamRoleBinding); err != nil {
		return ctrl.Result{}, err
	}

	teamRole, err := getTeamRole(ctx, r.Client, r.recorder, trb)
	if err != nil {
		return ctrl.Result{}, err
	}

	// list the clusters that either match the ClusterName or the ClusterSelector
	clusters, err := r.listClusters(ctx, trb)
	if err != nil {
		r.recorder.Eventf(trb, corev1.EventTypeNormal, greenhousev1alpha1.ClusterNotFoundReason, "Error retrieving clusters for TeamRoleBinding %s", trb.GetName)
		return ctrl.Result{}, err
	}

	trbStatus, err = r.cleanupResources(ctx, trbStatus, trb, clusters)
	if err != nil {
		return ctrl.Result{}, err
	}

	team, err := getTeam(ctx, r.Client, trb)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.recorder.Eventf(trb, corev1.EventTypeNormal, greenhousev1alpha1.TeamNotFoundReason, "Team %s not found in Namespace %s for TeamRoleBinding %s", trb.Spec.TeamRef, trb.GetNamespace(), trb.GetName())
		}
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, r.doReconcile(ctx, teamRole, clusters, trb, trbStatus, team)
}

// doReconcile reconciles the TeamRoleBinding's rbacv1 resources on all relevant clusters
func (r *TeamRoleBindingReconciler) doReconcile(ctx context.Context, teamRole *greenhousev1alpha1.TeamRole, clusters *greenhousev1alpha1.ClusterList, trb *greenhousev1alpha1.TeamRoleBinding, trbStatus greenhousev1alpha1.TeamRoleBindingStatus, team *greenhousev1alpha1.Team) error {
	failedClusters := []string{}
	cr := initRBACClusterRole(teamRole)
	for _, cluster := range clusters.Items {
		remoteRestClient, err := clientutil.NewK8sClientFromCluster(ctx, r.Client, &cluster)
		if err != nil {
			r.recorder.Eventf(trb, corev1.EventTypeWarning, "ClusterClientError", "Error getting client for cluster %s to replicate %s", cluster.GetName(), trb.GetName())
			setPropagationStatus(&trbStatus, cluster.GetName(), metav1.ConditionFalse, greenhousev1alpha1.ClusterConnectionFailed)
			failedClusters = append(failedClusters, cluster.GetName())
			continue
		}

		if err := reconcileClusterRole(ctx, remoteRestClient, &cluster, cr); err != nil {
			r.recorder.Eventf(trb, corev1.EventTypeWarning, greenhousev1alpha1.FailedReconcileClusterRoleReason, "Error reconciling ClusterRole %s in cluster %s", cr.GetName(), cluster.GetName())
			setPropagationStatus(&trbStatus, cluster.GetName(), metav1.ConditionFalse, greenhousev1alpha1.ClusterRoleFailed)
			failedClusters = append(failedClusters, cluster.GetName())
			continue
		}

		switch isClusterScoped(trb) {
		case true:
			crb := rbacClusterRoleBinding(trb, cr, team)
			if err := reconcileClusterRoleBinding(ctx, remoteRestClient, &cluster, crb); err != nil {
				r.recorder.Eventf(trb, corev1.EventTypeWarning, greenhousev1alpha1.FailedReconcileClusterRoleBindingReason, "Error reconciling ClusterRoleBinding %s in cluster %s: %s", crb.GetName(), cluster.GetName(), err.Error())
				setPropagationStatus(&trbStatus, cluster.GetName(), metav1.ConditionFalse, greenhousev1alpha1.RoleBindingFailed)
				failedClusters = append(failedClusters, cluster.GetName())
				continue
			}
			setPropagationStatus(&trbStatus, cluster.GetName(), metav1.ConditionTrue, greenhousev1alpha1.RBACReconciled)
		default:
			hasFailed := false
			for _, namespace := range trb.Spec.Namespaces {
				rbacRoleBinding := rbacRoleBinding(trb, cr, team, namespace)

				if err := reconcileRoleBinding(ctx, remoteRestClient, &cluster, rbacRoleBinding); err != nil {
					r.recorder.Eventf(trb, corev1.EventTypeWarning, greenhousev1alpha1.FailedReconcileRoleBindingReason, "Error reconciling RoleBinding %s in cluster %s: %s", rbacRoleBinding.GetName(), cluster.GetName(), err.Error())
					setPropagationStatus(&trbStatus, cluster.GetName(), metav1.ConditionFalse, greenhousev1alpha1.RoleBindingFailed)
					if !hasFailed {
						hasFailed = true
						failedClusters = append(failedClusters, cluster.GetName())
					}
				}
			}
			if hasFailed {
				continue
			}
		}
		setPropagationStatus(&trbStatus, cluster.GetName(), metav1.ConditionTrue, greenhousev1alpha1.RBACReconciled)
	}

	if len(failedClusters) > 0 {
		trbStatus.SetConditions(greenhousev1alpha1.FalseCondition(greenhousev1alpha1.RBACReady, greenhousev1alpha1.RBACReconcileFailed, "Error reconciling TeamRoleBindiding for clusters: "+strings.Join(failedClusters, ", ")))
		return fmt.Errorf("error reconciling TeamRoleBinding for clusters: %v", strings.Join(failedClusters, ", "))
	}

	trbStatus.SetConditions(greenhousev1alpha1.TrueCondition(greenhousev1alpha1.RBACReady, greenhousev1alpha1.RBACReconciled, ""))
	return nil
}

// deleteTeamRoleBinding removes the TeamRoleBinding's rbacv1 resources from all clusters and removes the finalizer if all resources have been deleted
func (r *TeamRoleBindingReconciler) deleteTeamRoleBinding(ctx context.Context, trb *greenhousev1alpha1.TeamRoleBinding, trbStatus greenhousev1alpha1.TeamRoleBindingStatus) (ctrl.Result, error) {
	clusters, err := r.listClusters(ctx, trb)
	if err != nil {
		r.recorder.Eventf(trb, corev1.EventTypeNormal, greenhousev1alpha1.ClusterNotFoundReason, "Error retrieving cluster for TeamRoleBinding %s", trb.GetName)
		return ctrl.Result{}, err
	}

	// add missing clusters from the Status to the list of clusters to be processed
	if err = r.combineClusterLists(ctx, trb.Namespace, clusters, trbStatus); err != nil {
		return ctrl.Result{}, err
	}

	for _, cluster := range clusters.Items {
		if err := r.cleanupCluster(ctx, trb, &cluster); err != nil {
			continue
		}
		trbStatus = removePropagationStatus(trbStatus, cluster.GetName())
	}

	// all clusters have been processed, finalizer can be removed
	if len(trbStatus.PropagationStatus) == 0 {
		err = clientutil.RemoveFinalizer(ctx, r.Client, trb, greenhouseapis.FinalizerCleanupTeamRoleBinding)
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// combineClusterLists appends clusters from the TeamRoleBinding's status not contained in the given ClusterList. This is an in place operation.
func (r *TeamRoleBindingReconciler) combineClusterLists(ctx context.Context, namespace string, clusters *greenhousev1alpha1.ClusterList, trbStatus greenhousev1alpha1.TeamRoleBindingStatus) error {
	for _, ps := range trbStatus.PropagationStatus {
		if slices.ContainsFunc(clusters.Items, func(c greenhousev1alpha1.Cluster) bool { return c.GetName() == ps.ClusterName }) {
			continue
		}

		cluster := &greenhousev1alpha1.Cluster{}
		err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: ps.ClusterName}, cluster)
		if apierrors.IsNotFound(err) {
			// cluster has been removed, nothing to be done
			trbStatus = removePropagationStatus(trbStatus, cluster.GetName())
			continue
		}
		if err != nil {
			return err
		}
		clusters.Items = append(clusters.Items, *cluster)
	}
	return nil
}

// cleanupResources removes rbacv1 resources from all clusters that are no longer matching the the TeamRoleBinding's clusterSelector/clusterName
func (r *TeamRoleBindingReconciler) cleanupResources(ctx context.Context, trbStatus greenhousev1alpha1.TeamRoleBindingStatus, trb *greenhousev1alpha1.TeamRoleBinding, clusters *greenhousev1alpha1.ClusterList) (greenhousev1alpha1.TeamRoleBindingStatus, error) {
	for _, s := range trbStatus.PropagationStatus {
		// remove rbac for all clusters no longer matching the clusterSelector
		if !slices.ContainsFunc(clusters.Items, func(c greenhousev1alpha1.Cluster) bool { return c.GetName() == s.ClusterName }) {
			cluster := &greenhousev1alpha1.Cluster{}
			err := r.Get(ctx, types.NamespacedName{Namespace: trb.Namespace, Name: s.ClusterName}, cluster)
			if apierrors.IsNotFound(err) {
				// cluster has been removed, nothing to be done
				trbStatus = removePropagationStatus(trbStatus, s.ClusterName)
				continue
			}
			if err != nil {
				return trbStatus, err
			}
			if err = r.cleanupCluster(ctx, trb, cluster); err != nil {
				return trbStatus, err
			}
			trbStatus = removePropagationStatus(trbStatus, s.ClusterName)
		}
	}
	return trbStatus, nil
}

// cleanupCluster removes the TeamRoleBinding's rbacv1 resources from the cluster, returns an error if the cleanup fails.
// This will remove the rbacv1.ClusterRoleBinding if the TeamRoleBinding is not namespaced, otherwise it will remove all rbacv1.RoleBindings
// If the rbacv1.ClusterRole is no longer referenced, it will be removed as well
func (r *TeamRoleBindingReconciler) cleanupCluster(ctx context.Context, trb *greenhousev1alpha1.TeamRoleBinding, cluster *greenhousev1alpha1.Cluster) error {
	cl, err := clientutil.NewK8sClientFromCluster(ctx, r.Client, cluster)
	if err != nil {
		log.FromContext(ctx).Error(err, "Error getting client for cluster %s to delete TeamRoleBinding %s", cluster.GetName(), trb.GetName())
		return err
	}

	switch isClusterScoped(trb) {
	case true:
		if err := r.deleteClusterRoleBinding(ctx, cl, trb, cluster); err != nil {
			return err
		}
	default:
		if err := r.deleteRoleBindings(ctx, cl, trb, cluster); err != nil {
			return err
		}
	}

	// if all RBAC bindings for this TeamRoleBinding have been deleted, check if the Role is still needed
	// if the rbacv1.ClusterRole is not referenced by any other rbacv1.ClusterRole, rbacv1.Role, it can be removed from the remote cluster
	isReferenced, err := isRoleReferenced(ctx, cl, trb)
	switch {
	case err != nil:
		return err
	case isReferenced:
		return nil
	}

	// ClusterRole is no longer referenced, delete the ClusterRole
	remoteObject := &rbacv1.ClusterRole{}
	remoteObjectKey := types.NamespacedName{Name: greenhouseapis.RBACPrefix + trb.Spec.TeamRoleRef}
	err = cl.Get(ctx, remoteObjectKey, remoteObject)
	switch {
	case apierrors.IsNotFound(err):
		return nil
	case err != nil:
		return err
	}
	if err = cl.Delete(ctx, remoteObject); err != nil {
		return err
	}
	return nil
}

// initRBACClusterRole returns a ClusterRole that matches the spec defined by the Greenhouse Role
func initRBACClusterRole(teamRole *greenhousev1alpha1.TeamRole) *rbacv1.ClusterRole {
	clusterRole := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: rbacv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   teamRole.GetRBACName(),
			Labels: map[string]string{greenhouseapis.LabelKeyRole: teamRole.GetName()},
		},
		Rules: teamRole.DeepCopy().Spec.Rules,
	}
	return clusterRole
}

// rbacRoleBinding creates a rbacv1.RoleBinding for a rbacv1.ClusterRole, Team and Namespace
func rbacRoleBinding(trb *greenhousev1alpha1.TeamRoleBinding, clusterRole *rbacv1.ClusterRole, team *greenhousev1alpha1.Team, namespace string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      trb.GetRBACName(),
			Namespace: namespace,
			Labels:    map[string]string{greenhouseapis.LabelKeyRoleBinding: trb.GetName()},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: clusterRole.GroupVersionKind().Group,
			Kind:     clusterRole.Kind,
			Name:     clusterRole.GetName(),
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: rbacv1.GroupName,
				Kind:     rbacv1.GroupKind,
				Name:     team.Spec.MappedIDPGroup,
			},
		},
	}
}

// rbacClusterRoleBinding creates a rbacv1.ClusterRoleBinding for a rbacv1.ClusterRole, Team and Namespace
func rbacClusterRoleBinding(trb *greenhousev1alpha1.TeamRoleBinding, clusterRole *rbacv1.ClusterRole, team *greenhousev1alpha1.Team) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: trb.GetRBACName(),
			Labels: map[string]string{
				greenhouseapis.LabelKeyRoleBinding: trb.GetName(),
				greenhouseapis.LabelKeyRole:        trb.Spec.TeamRoleRef,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: clusterRole.GroupVersionKind().Group,
			Kind:     clusterRole.Kind,
			Name:     clusterRole.GetName(),
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: rbacv1.GroupName,
				Kind:     rbacv1.GroupKind,
				Name:     team.Spec.MappedIDPGroup,
			},
		},
	}
}

// getTeamRole retrieves the Role referenced by the given RoleBinding in the RoleBinding's Namespace
func getTeamRole(ctx context.Context, c client.Client, r record.EventRecorder, teamRoleBinding *greenhousev1alpha1.TeamRoleBinding) (*greenhousev1alpha1.TeamRole, error) {
	if teamRoleBinding.Spec.TeamRoleRef == "" {
		r.Eventf(teamRoleBinding, corev1.EventTypeNormal, "RoleReferenceMissing", "TeamRoleBinding %s does not reference a TeamRole", teamRoleBinding.GetName())
		return nil, fmt.Errorf("error missing teamrole reference for teamrole %s", teamRoleBinding.GetName())
	}

	teamRole := &greenhousev1alpha1.TeamRole{}
	if err := c.Get(ctx, types.NamespacedName{Name: teamRoleBinding.Spec.TeamRoleRef, Namespace: teamRoleBinding.GetNamespace()}, teamRole); err != nil {
		return nil, fmt.Errorf("error getting teamrole: %w", err)
	}
	return teamRole, nil
}

// getTeam retrieves the Team referenced by the given TeamRoleBinding in the TeamRoleBinding's Namespace
func getTeam(ctx context.Context, c client.Client, teamRoleBinding *greenhousev1alpha1.TeamRoleBinding) (*greenhousev1alpha1.Team, error) {
	if teamRoleBinding.Spec.TeamRef == "" {
		return nil, fmt.Errorf("error missing team reference")
	}

	team := &greenhousev1alpha1.Team{}
	if err := c.Get(ctx, types.NamespacedName{Name: teamRoleBinding.Spec.TeamRef, Namespace: teamRoleBinding.GetNamespace()}, team); err != nil {
		return nil, fmt.Errorf("error getting team: %w", err)
	}
	return team, nil
}

// reconcileClusterRole creates or updates a ClusterRole in the Cluster the given client.Client is created for
func reconcileClusterRole(ctx context.Context, cl client.Client, c *greenhousev1alpha1.Cluster, cr *rbacv1.ClusterRole) error {
	remoteCR := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: cr.Name,
		},
	}
	result, err := clientutil.CreateOrPatch(ctx, cl, remoteCR, func() error {
		remoteCR.Labels = cr.Labels
		remoteCR.Rules = cr.Rules
		return nil
	})

	if err != nil {
		return err
	}
	log.FromContext(ctx).Info(fmt.Sprintf("%s ClusterRoleBinding", result), "name", cr.GetName(), "cluster", c.GetName())
	return nil
}

// reconcileClusterRoleBinding creates or updates a ClusterRoleBinding in the Cluster the given client.Client is created for
func reconcileClusterRoleBinding(ctx context.Context, cl client.Client, c *greenhousev1alpha1.Cluster, crb *rbacv1.ClusterRoleBinding) error {
	remoteCRB := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: crb.Name,
		},
	}
	result, err := clientutil.CreateOrPatch(ctx, cl, remoteCRB, func() error {
		remoteCRB.Labels = crb.Labels
		remoteCRB.RoleRef = crb.RoleRef
		remoteCRB.Subjects = crb.Subjects
		return nil
	})

	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultNone:
		log.FromContext(ctx).Info("noop ClusterRoleBinding", "name", crb.GetName(), "cluster", c.GetName())
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created ClusterRoleBinding", "name", crb.GetName(), "cluster", c.GetName())
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated ClusterRoleBinding", "name", crb.GetName(), "cluster", c.GetName())
	}
	return nil
}

// reconcileRoleBinding creates or updates a RoleBinding in the Cluster the given client.Client is created for
func reconcileRoleBinding(ctx context.Context, cl client.Client, c *greenhousev1alpha1.Cluster, rb *rbacv1.RoleBinding) error {
	remoteRB := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rb.Name,
			Namespace: rb.Namespace,
		},
	}

	result, err := clientutil.CreateOrPatch(ctx, cl, remoteRB, func() error {
		remoteRB.Labels = rb.Labels
		remoteRB.RoleRef = rb.RoleRef
		remoteRB.Subjects = rb.Subjects
		return nil
	})

	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultNone:
		log.FromContext(ctx).Info("noop RoleBinding", "name", rb.GetName(), "cluster", c.GetName(), "namespace", rb.GetNamespace())
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created RoleBinding", "name", rb.GetName(), "cluster", c.GetName(), "namespace", rb.GetNamespace())
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated RoleBinding", "name", rb.GetName(), "cluster", c.GetName(), "namespace", rb.GetNamespace())
	}
	return nil
}

func (r TeamRoleBindingReconciler) deleteRoleBindings(ctx context.Context, cl client.Client, trb *greenhousev1alpha1.TeamRoleBinding, cluster *greenhousev1alpha1.Cluster) error {
	for _, namespace := range trb.Spec.Namespaces {
		remoteObjectKey := types.NamespacedName{Name: trb.GetRBACName(), Namespace: namespace}
		remoteObject := &rbacv1.RoleBinding{}
		err := cl.Get(ctx, remoteObjectKey, remoteObject)
		switch {
		case apierrors.IsNotFound(err):
			continue
		case err != nil:
			r.recorder.Eventf(trb, corev1.EventTypeWarning, greenhousev1alpha1.FailedDeleteRoleBindingReason, "Error retrieving RoleBinding %s for cluster %s in namespace %s", trb.GetRBACName(), cluster.GetName(), namespace)
			// TODO: collect errors and return them
			return err
		}
		if err := cl.Delete(ctx, remoteObject); err != nil {
			r.recorder.Eventf(trb, corev1.EventTypeWarning, greenhousev1alpha1.FailedDeleteRoleBindingReason, "Error deleting RoleBinding %s for cluster %s in namespace %s", trb.GetRBACName(), cluster.GetName(), namespace)
			// TODO: collect errors and return them
			return err
		}
	}
	return nil
}

func (r TeamRoleBindingReconciler) deleteClusterRoleBinding(ctx context.Context, cl client.Client, trb *greenhousev1alpha1.TeamRoleBinding, cluster *greenhousev1alpha1.Cluster) error {
	remoteObjectKey := types.NamespacedName{Name: trb.GetRBACName()}
	remoteObject := &rbacv1.ClusterRoleBinding{}
	err := cl.Get(ctx, remoteObjectKey, remoteObject)
	switch {
	case apierrors.IsNotFound(err):
		return nil
	case err != nil:
		r.recorder.Eventf(trb, corev1.EventTypeWarning, greenhousev1alpha1.FailedDeleteRoleBindingReason, "Error retrieving ClusterRoleBinding %s for cluster %s", trb.GetRBACName(), cluster.GetName())
		// TODO: collect errors and return them
		return err
	}
	if err := cl.Delete(ctx, remoteObject); err != nil {
		r.recorder.Eventf(trb, corev1.EventTypeWarning, greenhousev1alpha1.FailedDeleteClusterRoleBindingReason, "Error deleting ClusterRoleBinding %s for cluster %s", trb.GetRBACName(), cluster.GetName())
		// TODO: collect errors and return them
		return err
	}

	return nil
}

// enqueueTeamRoleBindingsFor enqueues all TeamRoleBindings that are referenced by the given TeamRole or Team
func (r *TeamRoleBindingReconciler) enqueueTeamRoleBindingsFor(ctx context.Context, o client.Object) []ctrl.Request {
	fieldRef := ""
	// determine the field to select TeamRoleBindings by
	switch o.(type) {
	case *greenhousev1alpha1.TeamRole:
		fieldRef = greenhouseapis.RolebindingRoleRefField
	case *greenhousev1alpha1.Team:
		fieldRef = greenhouseapis.RolebindingTeamRefField
	default:
		return []ctrl.Request{}
	}
	listOpts := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(fieldRef, o.GetName()),
		Namespace:     o.GetNamespace(),
	}
	// list all referenced TeamRoleBindings
	teamRoleBindings := &greenhousev1alpha1.TeamRoleBindingList{}
	if err := r.Client.List(ctx, teamRoleBindings, listOpts); err != nil {
		return []ctrl.Request{}
	}

	// return a list of reconcile.Requests for the list of referenced TeamRoleBindings
	requests := make([]ctrl.Request, len(teamRoleBindings.Items))
	for i, trb := range teamRoleBindings.Items {
		requests[i] = ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      trb.GetName(),
				Namespace: trb.GetNamespace(),
			},
		}
	}
	return requests
}

// enqueueAllTeamRoleBindingsInNamespace returns a list of reconcile requests for all TeamRoleBindings in the same namespace as obj.
func (r *TeamRoleBindingReconciler) enqueueAllTeamRoleBindingsInNamespace(ctx context.Context, obj client.Object) []ctrl.Request {
	return listTeamRoleBindingsAsReconcileRequests(ctx, r.Client, client.InNamespace(obj.GetNamespace()))
}

// listTeamRoleBindingsAsReconcileRequests returns a list of reconcile requests for all PluginPresets that match the given list options.
func listTeamRoleBindingsAsReconcileRequests(ctx context.Context, c client.Client, listOpts ...client.ListOption) []ctrl.Request {
	var allTeamRoleBindings = new(greenhousev1alpha1.TeamRoleBindingList)
	if err := c.List(ctx, allTeamRoleBindings, listOpts...); err != nil {
		return nil
	}
	requests := make([]ctrl.Request, len(allTeamRoleBindings.Items))
	for i, trb := range allTeamRoleBindings.Items {
		requests[i] = ctrl.Request{NamespacedName: client.ObjectKeyFromObject(trb.DeepCopy())}
	}
	return requests
}

// isClusterScoped returns true if the TeamRoleBinding will create ClusterRoleBindings
func isClusterScoped(trb *greenhousev1alpha1.TeamRoleBinding) bool {
	return len(trb.Spec.Namespaces) == 0
}

// isRoleReferenced checks if the given TeamRoleBinding's TeamRole is still referenced by any Role or ClusterRole
func isRoleReferenced(ctx context.Context, c client.Client, teamRoleBinding *greenhousev1alpha1.TeamRoleBinding) (bool, error) {
	// list options to select Roles and ClusterRoles by "greenhouse.sap/role" set to TeamRoleBinding's TeamRoleRef
	listOpts := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{greenhouseapis.LabelKeyRole: teamRoleBinding.Spec.TeamRoleRef}),
	}

	roles := &rbacv1.RoleList{}
	if err := c.List(ctx, roles, listOpts); err != nil {
		return true, err
	}
	if len(roles.Items) > 0 { // if any Roles are found, the Role is still referenced
		return true, nil
	}

	clusterRoleBindings := &rbacv1.ClusterRoleBindingList{}
	if err := c.List(ctx, clusterRoleBindings, listOpts); err != nil {
		return true, err
	}
	if len(clusterRoleBindings.Items) > 0 { // if any ClusterRoles are found, the Role is still referenced
		return true, nil
	}
	return false, nil
}

// listClusters returns the list of Clusters that match the TeamRoleBinding's ClusterSelector or ClusterName
func (r *TeamRoleBindingReconciler) listClusters(ctx context.Context, trb *greenhousev1alpha1.TeamRoleBinding) (*greenhousev1alpha1.ClusterList, error) {
	if trb.Spec.ClusterName != "" {
		cluster := new(greenhousev1alpha1.Cluster)
		if err := r.Get(ctx, types.NamespacedName{Name: trb.Spec.ClusterName, Namespace: trb.GetNamespace()}, cluster); err != nil {
			return nil, err
		}
		return &greenhousev1alpha1.ClusterList{Items: []greenhousev1alpha1.Cluster{*cluster}}, nil
	}

	clusterSelector, err := v1.LabelSelectorAsSelector(&trb.Spec.ClusterSelector)
	if err != nil {
		return nil, err
	}
	var clusters = new(greenhousev1alpha1.ClusterList)
	if err := r.List(ctx, clusters, client.InNamespace(trb.GetNamespace()), client.MatchingLabelsSelector{Selector: clusterSelector}); err != nil {
		return nil, err
	}
	return clusters, nil
}

// initTeamRoleBindingStatus ensures that all required conditions are present in the TeamRoleBinding's Status
func initTeamRoleBindingStatus(trb *greenhousev1alpha1.TeamRoleBinding) greenhousev1alpha1.TeamRoleBindingStatus {
	status := trb.Status.DeepCopy()
	for _, ct := range exposedConditions {
		if status.GetConditionByType(ct) == nil {
			status.SetConditions(greenhousev1alpha1.UnknownCondition(ct, "", ""))
		}
	}
	return *status
}

// setStatus patches the Status of the TeamRoleBinding
func (r *TeamRoleBindingReconciler) setStatus(ctx context.Context, trb *greenhousev1alpha1.TeamRoleBinding, status greenhousev1alpha1.TeamRoleBindingStatus) error {
	readyCondition := computeReadyCondition(status)
	status.StatusConditions.SetConditions(readyCondition)
	_, err := clientutil.PatchStatus(ctx, r.Client, trb, func() error {
		trb.Status = status
		return nil
	})
	return err
}

// computeReadyCondition computes the ReadyCondition based on the TeamRoleBinding's StatusConditions
func computeReadyCondition(status greenhousev1alpha1.TeamRoleBindingStatus) greenhousev1alpha1.Condition {
	readyCondition := *status.GetConditionByType(greenhousev1alpha1.ReadyCondition)

	for _, condition := range status.PropagationStatus {
		if condition.IsTrue() {
			continue // skip if the RBACReady condition is true for this cluster
		}
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Message = "Team RBAC reconciliation failed"
		return readyCondition
	}

	if status.GetConditionByType(greenhousev1alpha1.ClusterListEmpty).IsTrue() {
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Message = "No cluster matches ClusterSelector or ClusterName"
		return readyCondition
	}
	readyCondition.Status = metav1.ConditionTrue
	readyCondition.Message = "ready"
	return readyCondition
}

// setPropagationStatus updates the TeamRoleBinding's PropagationStatus for the Cluster
func setPropagationStatus(status *greenhousev1alpha1.TeamRoleBindingStatus, cluster string, rbacReady metav1.ConditionStatus, reason greenhousev1alpha1.ConditionReason) {
	exists := false
	condition := greenhousev1alpha1.NewCondition(greenhousev1alpha1.RBACReady, rbacReady, reason, "")
	for i, ps := range status.PropagationStatus {
		if ps.ClusterName == cluster {
			if ps.Condition.Status == rbacReady {
				condition.LastTransitionTime = ps.Condition.LastTransitionTime
			}
			status.PropagationStatus[i].Condition = condition
			return
		}
	}
	if !exists {
		condition.LastTransitionTime = metav1.Now()
		status.PropagationStatus = append(status.PropagationStatus, greenhousev1alpha1.PropagationStatus{
			ClusterName: cluster,
			Condition:   condition,
		})
	}
}

func removePropagationStatus(status greenhousev1alpha1.TeamRoleBindingStatus, cluster string) greenhousev1alpha1.TeamRoleBindingStatus {
	updatedStatus := slices.DeleteFunc(status.PropagationStatus, func(ps greenhousev1alpha1.PropagationStatus) bool {
		return ps.ClusterName == cluster
	})
	status.PropagationStatus = updatedStatus
	return status
}
