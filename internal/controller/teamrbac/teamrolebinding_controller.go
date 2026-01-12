// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package teamrbac

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/util"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

var exposedConditions = []greenhousemetav1alpha1.ConditionType{
	greenhousemetav1alpha1.ReadyCondition,
	greenhousev1alpha2.RBACReady,
	greenhousemetav1alpha1.OwnerLabelSetCondition,
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

	// index TeamRoleBindings by the TeamRef field for faster lookups
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &greenhousev1alpha2.TeamRoleBinding{}, greenhouseapis.RolebindingTeamRefField, func(rawObj client.Object) []string {
		// Extract the TeamRole name from the TeamRoleBinding Spec, if one is provided
		teamRoleBinding, ok := rawObj.(*greenhousev1alpha2.TeamRoleBinding)
		if teamRoleBinding.Spec.TeamRef == "" || !ok {
			return nil
		}
		return []string{teamRoleBinding.Spec.TeamRef}
	}); clientutil.IgnoreIndexerConflict(err) != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&greenhousev1alpha2.TeamRoleBinding{}).
		Watches(&greenhousev1alpha1.TeamRole{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueTeamRoleBindingsFor)).
		Watches(&greenhousev1alpha1.Team{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueTeamRoleBindingsFor)).
		// Reconcile TeamRoleBindings for all Cluster label changes in the same namespace
		Watches(&greenhousev1alpha1.Cluster{}, handler.EnqueueRequestsFromMapFunc(r.enqueueAllTeamRoleBindingsInNamespace),
			builder.WithPredicates(predicate.LabelChangedPredicate{})).
		Complete(r)
}

func (r *TeamRoleBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return lifecycle.Reconcile(ctx, r.Client, req.NamespacedName, &greenhousev1alpha2.TeamRoleBinding{}, r, r.setConditions())
}

func (r *TeamRoleBindingReconciler) setConditions() lifecycle.Conditioner {
	return func(ctx context.Context, resource lifecycle.RuntimeObject) {
		logger := ctrl.LoggerFrom(ctx)
		trb, ok := resource.(*greenhousev1alpha2.TeamRoleBinding)
		if !ok {
			logger.Error(errors.New("resource is not a TeamRoleBinding"), "status setup failed")
			return
		}

		readyCondition := computeReadyCondition(trb.Status)
		ownerLabelCondition := util.ComputeOwnerLabelCondition(ctx, r.Client, trb)
		util.UpdateOwnedByLabelMissingMetric(trb, ownerLabelCondition.IsFalse())
		trb.Status.SetConditions(readyCondition, ownerLabelCondition)
		UpdateTeamrbacMetrics(trb)
	}
}

func (r *TeamRoleBindingReconciler) EnsureCreated(ctx context.Context, resource lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	trb, ok := resource.(*greenhousev1alpha2.TeamRoleBinding)
	if !ok {
		return ctrl.Result{}, lifecycle.Failed, errors.New("RuntimeObject has incompatible type")
	}

	_ = log.FromContext(ctx)

	initTeamRoleBindingStatus(trb)

	teamRole, err := getTeamRole(ctx, r.Client, r.recorder, trb)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}

	// list the clusters that either match the ClusterName or the ClusterSelector
	clusters, err := r.listClusters(ctx, trb)
	if err != nil {
		trb.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha2.RBACReady, greenhousev1alpha2.EmptyClusterList, "Failed to get clusters for TeamRoleBinding"))
		return ctrl.Result{}, lifecycle.Failed, err
	}

	err = r.cleanupResources(ctx, trb, clusters)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}

	// exit early if the cluster list is empty
	if len(clusters.Items) == 0 {
		trb.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha2.RBACReady, greenhousev1alpha2.EmptyClusterList, ""))
		r.recorder.Eventf(trb, corev1.EventTypeWarning, greenhousemetav1alpha1.FailedEvent, "No clusters found for %s", trb.GetName())
		return ctrl.Result{}, lifecycle.Success, nil
	}

	team, err := getTeam(ctx, r.Client, trb)
	if err != nil {
		if apierrors.IsNotFound(err) {
			message := fmt.Sprintf("Failed to get team %s in namespace %s", trb.Spec.TeamRef, trb.GetNamespace())
			trb.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha2.RBACReady, greenhousev1alpha2.TeamNotFound, message))
			r.recorder.Event(trb, corev1.EventTypeWarning, greenhousemetav1alpha1.FailedEvent, message)
		}
		return ctrl.Result{}, lifecycle.Failed, err
	}

	err = r.doReconcile(ctx, teamRole, clusters, trb, team)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}
	return ctrl.Result{}, lifecycle.Success, err
}

// EnsureDeleted - removes the TeamRoleBinding's rbacv1 resources from all clusters.
func (r *TeamRoleBindingReconciler) EnsureDeleted(ctx context.Context, resource lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	trb, ok := resource.(*greenhousev1alpha2.TeamRoleBinding)
	if !ok {
		return ctrl.Result{}, lifecycle.Failed, errors.New("RuntimeObject has incompatible type")
	}

	clusters, err := r.listClusters(ctx, trb)
	if err != nil {
		r.recorder.Eventf(trb, corev1.EventTypeWarning, greenhousemetav1alpha1.FailedDeleteEvent, "Failed to list clusters")
		return ctrl.Result{}, lifecycle.Failed, err
	}

	// add missing clusters from the Status to the list of clusters to be processed
	if err = r.combineClusterLists(ctx, trb.Namespace, clusters, trb); err != nil {
		r.recorder.Eventf(trb, corev1.EventTypeWarning, greenhousemetav1alpha1.FailedDeleteEvent, "Failed to list clusters")
		return ctrl.Result{}, lifecycle.Failed, err
	}

	for _, cluster := range clusters.Items {
		if err := r.cleanupCluster(ctx, trb, &cluster); err != nil {
			r.recorder.Eventf(trb, corev1.EventTypeWarning, greenhousemetav1alpha1.FailedDeleteEvent, "Failed to remove resources for %s from cluster %s", trb.GetName(), cluster.GetName())
			continue
		}
		trb.RemovePropagationStatus(cluster.GetName())
	}

	// all clusters have been processed, finalizer can be removed
	if len(trb.Status.PropagationStatus) == 0 {
		r.recorder.Eventf(trb, corev1.EventTypeNormal, greenhousemetav1alpha1.SuccessfulDeletedEvent, "Deleted TeamRoleBinding %s from all clusters", trb.GetName())
		return ctrl.Result{}, lifecycle.Success, nil
	}
	return ctrl.Result{}, lifecycle.Pending, nil
}

func (r *TeamRoleBindingReconciler) EnsureSuspended(_ context.Context, _ lifecycle.RuntimeObject) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

// doReconcile reconciles the TeamRoleBinding's rbacv1 resources on all relevant clusters
func (r *TeamRoleBindingReconciler) doReconcile(ctx context.Context, teamRole *greenhousev1alpha1.TeamRole, clusters *greenhousev1alpha1.ClusterList, trb *greenhousev1alpha2.TeamRoleBinding, team *greenhousev1alpha1.Team) error {
	failedClusters := []string{}
	cr := initRBACClusterRole(teamRole)

	for _, cluster := range clusters.Items {
		remoteRestClient, err := clientutil.NewK8sClientFromCluster(ctx, r.Client, &cluster)
		if err != nil {
			r.recorder.Eventf(trb, corev1.EventTypeWarning, greenhousemetav1alpha1.FailedEvent, "Error getting client for cluster %s to replicate %s", cluster.GetName(), trb.GetName())
			trb.SetPropagationStatus(cluster.GetName(), metav1.ConditionFalse, greenhousev1alpha2.ClusterConnectionFailed, err.Error())
			if !slices.Contains(failedClusters, cluster.GetName()) {
				failedClusters = append(failedClusters, cluster.GetName())
			}
			continue
		}

		if err := reconcileClusterRole(ctx, remoteRestClient, &cluster, cr); err != nil {
			r.recorder.Eventf(trb, corev1.EventTypeWarning, greenhousemetav1alpha1.FailedEvent, "Failed to reconcile ClusterRole %s in cluster %s", cr.GetName(), cluster.GetName())
			trb.SetPropagationStatus(cluster.GetName(), metav1.ConditionFalse, greenhousev1alpha2.ClusterRoleFailed, err.Error())
			if !slices.Contains(failedClusters, cluster.GetName()) {
				failedClusters = append(failedClusters, cluster.GetName())
			}
			continue
		}

		switch isClusterScoped(trb) {
		case true:
			crb := rbacClusterRoleBinding(trb, cr, team)
			if err := reconcileClusterRoleBinding(ctx, remoteRestClient, &cluster, crb); err != nil {
				r.recorder.Eventf(trb, corev1.EventTypeWarning, greenhousemetav1alpha1.FailedEvent, "Failed to reconcile ClusterRoleBinding %s in cluster %s", crb.GetName(), cluster.GetName())
				trb.SetPropagationStatus(cluster.GetName(), metav1.ConditionFalse, greenhousev1alpha2.RoleBindingFailed, err.Error())
				if !slices.Contains(failedClusters, cluster.GetName()) {
					failedClusters = append(failedClusters, cluster.GetName())
				}
				continue
			}
			trb.SetPropagationStatus(cluster.GetName(), metav1.ConditionTrue, greenhousev1alpha2.RBACReconciled, "")
		default:
			errorMesages := []string{}
			for _, namespace := range trb.Spec.Namespaces {
				rbacRoleBinding := rbacRoleBinding(trb, cr, team, namespace)

				if err := reconcileRoleBinding(ctx, remoteRestClient, &cluster, rbacRoleBinding, trb.Spec.CreateNamespaces); err != nil {
					r.recorder.Eventf(trb, corev1.EventTypeWarning, greenhousemetav1alpha1.FailedEvent, "Failed to reconcile RoleBinding %s in cluster/namespace %s/%s: ", rbacRoleBinding.GetName(), cluster.GetName(), namespace)
					if !slices.Contains(failedClusters, cluster.GetName()) {
						failedClusters = append(failedClusters, cluster.GetName())
					}
					errorMesages = append(errorMesages, err.Error())
				}
			}
			if len(errorMesages) > 0 {
				trb.SetPropagationStatus(cluster.GetName(), metav1.ConditionFalse, greenhousev1alpha2.RoleBindingFailed, "Failed to reconcile RoleBindings: "+strings.Join(errorMesages, ", "))
				continue
			}
		}

		trb.SetPropagationStatus(cluster.GetName(), metav1.ConditionTrue, greenhousev1alpha2.RBACReconciled, "")
	}

	if len(failedClusters) > 0 {
		message := "Error reconciling TeamRoleBinding for clusters: " + strings.Join(failedClusters, ", ")
		trb.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha2.RBACReady, greenhousev1alpha2.RBACReconcileFailed, message))
		return errors.New(message)
	}

	trb.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousev1alpha2.RBACReady, greenhousev1alpha2.RBACReconciled, ""))
	return nil
}

// combineClusterLists appends clusters from the TeamRoleBinding's status not contained in the given ClusterList. This is an in place operation.
func (r *TeamRoleBindingReconciler) combineClusterLists(ctx context.Context, namespace string, clusters *greenhousev1alpha1.ClusterList, trb *greenhousev1alpha2.TeamRoleBinding) error {
	for _, ps := range trb.Status.PropagationStatus {
		if slices.ContainsFunc(clusters.Items, func(c greenhousev1alpha1.Cluster) bool { return c.GetName() == ps.ClusterName }) {
			continue
		}

		cluster := &greenhousev1alpha1.Cluster{}
		err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: ps.ClusterName}, cluster)
		if apierrors.IsNotFound(err) {
			// cluster has been removed, nothing to be done
			trb.RemovePropagationStatus(cluster.GetName())
			continue
		}
		if err != nil {
			return err
		}
		clusters.Items = append(clusters.Items, *cluster)
	}
	return nil
}

// cleanupResources removes rbacv1 resources from all clusters that are no longer matching the TeamRoleBinding's clusterSelector/clusterName
// if the Cluster is not ready, the TeamRoleBinding's status will be updated accordingly but no resources will be removed
func (r *TeamRoleBindingReconciler) cleanupResources(ctx context.Context, trb *greenhousev1alpha2.TeamRoleBinding, clusters *greenhousev1alpha1.ClusterList) error {
	for _, s := range trb.Status.PropagationStatus {
		// remove rbac for all clusters no longer matching the clusterSelector
		if !slices.ContainsFunc(clusters.Items, func(c greenhousev1alpha1.Cluster) bool { return c.GetName() == s.ClusterName }) {
			cluster := &greenhousev1alpha1.Cluster{}
			err := r.Get(ctx, types.NamespacedName{Namespace: trb.Namespace, Name: s.ClusterName}, cluster)
			if apierrors.IsNotFound(err) {
				// cluster has been removed, nothing to be done
				trb.RemovePropagationStatus(s.ClusterName)
				continue
			}
			if err != nil {
				return err
			}
			if !cluster.Status.IsReadyTrue() {
				trb.SetPropagationStatus(s.ClusterName, metav1.ConditionFalse, greenhousev1alpha2.ClusterConnectionFailed, "Cluster is not ready")
				continue
			}
			if err = r.cleanupCluster(ctx, trb, cluster); err != nil {
				return err
			}
			trb.RemovePropagationStatus(s.ClusterName)
			continue
		}

		// Remove RoleBindings from all namespaces not matching the .spec.namespaces.
		cluster := &greenhousev1alpha1.Cluster{}
		err := r.Get(ctx, types.NamespacedName{Namespace: trb.Namespace, Name: s.ClusterName}, cluster)
		if err != nil {
			return err
		}
		if err = r.cleanupClusterNamespaces(ctx, trb, cluster); err != nil {
			return err
		}
	}
	return nil
}

// cleanupClusterNamespaces removes RoleBindings not matching the trb.spec.namespaces from the cluster.
func (r *TeamRoleBindingReconciler) cleanupClusterNamespaces(ctx context.Context, trb *greenhousev1alpha2.TeamRoleBinding, cluster *greenhousev1alpha1.Cluster) error {
	if isClusterScoped(trb) {
		return nil
	}

	cl, err := clientutil.NewK8sClientFromCluster(ctx, r.Client, cluster)
	if err != nil {
		log.FromContext(ctx).Error(err, "Error getting client for cluster", "cluster", cluster.GetName())
		return err
	}

	var roleBindings = new(rbacv1.RoleBindingList)
	err = cl.List(ctx, roleBindings, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("metadata.name", trb.GetRBACName()),
	})
	if apierrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}

	roleBindingsToDelete := slices.DeleteFunc(roleBindings.Items, func(roleBinding rbacv1.RoleBinding) bool {
		return slices.ContainsFunc(trb.Spec.Namespaces, func(namespace string) bool {
			return roleBinding.Namespace == namespace
		})
	})
	if len(roleBindingsToDelete) == 0 {
		return nil
	}

	for _, roleBinding := range roleBindingsToDelete {
		result, err := clientutil.Delete(ctx, cl, &roleBinding)
		switch {
		case err != nil:
			log.FromContext(ctx).Error(err, "error deleting RoleBinding", "roleBinding", trb.GetRBACName(), "cluster", cluster.GetName(), "namespace", roleBinding.Namespace)
			return err
		case result == clientutil.DeletionResultDeleted:
			log.FromContext(ctx).Info("deleted RoleBinding successfully", "roleBinding", trb.GetRBACName(), "cluster", cluster.GetName(), "namespace", roleBinding.Namespace)
		}
	}
	return nil
}

// cleanupCluster removes the TeamRoleBinding's rbacv1 resources from the cluster, returns an error if the cleanup fails.
// This will remove the rbacv1.ClusterRoleBinding if the TeamRoleBinding is not namespaced, otherwise it will remove all rbacv1.RoleBindings
// If the rbacv1.ClusterRole is no longer referenced, it will be removed as well
func (r *TeamRoleBindingReconciler) cleanupCluster(ctx context.Context, trb *greenhousev1alpha2.TeamRoleBinding, cluster *greenhousev1alpha1.Cluster) error {
	cl, err := clientutil.NewK8sClientFromCluster(ctx, r.Client, cluster)
	if err != nil {
		log.FromContext(ctx).Error(err, "Error getting client for cluster", "cluster", cluster.GetName())
		return err
	}

	switch isClusterScoped(trb) {
	case true:
		if err := r.deleteClusterRoleBinding(ctx, cl, trb, cluster); err != nil {
			return err
		}
	default:
		if err := r.deleteAllDeployedRoleBindings(ctx, cl, trb, cluster); err != nil {
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
	return r.deleteClusterRole(ctx, cl, trb, cluster)
}

// initRBACClusterRole returns a ClusterRole that matches the spec defined by the Greenhouse Role
func initRBACClusterRole(teamRole *greenhousev1alpha1.TeamRole) *rbacv1.ClusterRole {
	roleLabels := teamRole.Spec.Labels
	if len(roleLabels) == 0 {
		roleLabels = make(map[string]string, 1)
	}
	roleLabels[greenhouseapis.LabelKeyRole] = teamRole.GetName()

	clusterRole := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: rbacv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   teamRole.GetRBACName(),
			Labels: roleLabels,
		},
		Rules:           teamRole.DeepCopy().Spec.Rules,
		AggregationRule: teamRole.Spec.AggregationRule.DeepCopy(),
	}
	return clusterRole
}

// rbacRoleBinding creates a rbacv1.RoleBinding for a rbacv1.ClusterRole, Team and Namespace
func rbacRoleBinding(trb *greenhousev1alpha2.TeamRoleBinding, clusterRole *rbacv1.ClusterRole, team *greenhousev1alpha1.Team, namespace string) *rbacv1.RoleBinding {
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
		Subjects: generateSubjects(trb.Spec.Usernames, team.Spec.MappedIDPGroup),
	}
}

// rbacClusterRoleBinding creates a rbacv1.ClusterRoleBinding for a rbacv1.ClusterRole, Team and Namespace
func rbacClusterRoleBinding(trb *greenhousev1alpha2.TeamRoleBinding, clusterRole *rbacv1.ClusterRole, team *greenhousev1alpha1.Team) *rbacv1.ClusterRoleBinding {
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
		Subjects: generateSubjects(trb.Spec.Usernames, team.Spec.MappedIDPGroup),
	}
}

// generateSubjects returns a list of subjects with mappedIDPGroup as a rbacv1.GroupKind, and any usernames as rbacv1.UserKind
func generateSubjects(usernames []string, mappedIDPGroup string) []rbacv1.Subject {
	var subjects []rbacv1.Subject
	for _, username := range usernames {
		subjects = append(subjects, rbacv1.Subject{
			Kind:     rbacv1.UserKind,
			APIGroup: rbacv1.GroupName,
			Name:     username,
		})
	}

	return append(subjects, rbacv1.Subject{
		APIGroup: rbacv1.GroupName,
		Kind:     rbacv1.GroupKind,
		Name:     mappedIDPGroup,
	})
}

// getTeamRole retrieves the Role referenced by the given RoleBinding in the RoleBinding's Namespace
func getTeamRole(ctx context.Context, c client.Client, r record.EventRecorder, teamRoleBinding *greenhousev1alpha2.TeamRoleBinding) (*greenhousev1alpha1.TeamRole, error) {
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
func getTeam(ctx context.Context, c client.Client, teamRoleBinding *greenhousev1alpha2.TeamRoleBinding) (*greenhousev1alpha1.Team, error) {
	if teamRoleBinding.Spec.TeamRef == "" {
		return nil, errors.New("error missing team reference")
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
		remoteCR.AggregationRule = cr.AggregationRule
		return nil
	})

	if err != nil {
		return err
	}
	log.FromContext(ctx).Info(fmt.Sprintf("%s ClusterRoleBinding", result), "clusterRole", cr.GetName(), "cluster", c.GetName())
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
		log.FromContext(ctx).Info("noop ClusterRoleBinding", "clusterRoleBinding", crb.GetName(), "cluster", c.GetName())
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created ClusterRoleBinding", "clusterRoleBinding", crb.GetName(), "cluster", c.GetName())
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated ClusterRoleBinding", "clusterRoleBinding", crb.GetName(), "cluster", c.GetName())
	}
	return nil
}

// reconcileRoleBinding creates or updates a RoleBinding in the Cluster the given client.Client is created for
func reconcileRoleBinding(ctx context.Context, cl client.Client, c *greenhousev1alpha1.Cluster, rb *rbacv1.RoleBinding, createNamespaces bool) error {
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
		if createNamespaces && apierrors.IsNotFound(err) {
			err := cl.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: rb.Namespace}})
			if err != nil {
				return err
			}
			return errors.New("failed to create RoleBinding, created missing namespace")
		} else {
			return err
		}
	}

	switch result {
	case clientutil.OperationResultNone:
		log.FromContext(ctx).Info("noop RoleBinding", "roleBinding", rb.GetName(), "cluster", c.GetName(), "namespace", rb.GetNamespace())
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created RoleBinding", "roleBinding", rb.GetName(), "cluster", c.GetName(), "namespace", rb.GetNamespace())
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated RoleBinding", "roleBinding", rb.GetName(), "cluster", c.GetName(), "namespace", rb.GetNamespace())
	}
	return nil
}

// deleteAllDeployedRoleBindings deletes all RoleBindings deployed to a remote cluster.
// Deletes not only those specified in .spec.namespaces, but all by the trb.GetRBACName() name.
func (r TeamRoleBindingReconciler) deleteAllDeployedRoleBindings(ctx context.Context, cl client.Client, trb *greenhousev1alpha2.TeamRoleBinding, cluster *greenhousev1alpha1.Cluster) error {
	var roleBindingsToDelete = new(rbacv1.RoleBindingList)
	err := cl.List(ctx, roleBindingsToDelete, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("metadata.name", trb.GetRBACName()),
	})
	if apierrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}

	for _, roleBinding := range roleBindingsToDelete.Items {
		result, err := clientutil.Delete(ctx, cl, &roleBinding)

		switch {
		case err != nil:
			log.FromContext(ctx).Error(err, "error deleting RoleBinding", "roleBinding", trb.GetRBACName(), "cluster", cluster.GetName(), "namespace", roleBinding.Namespace)
			return err
		case result == clientutil.DeletionResultDeleted:
			log.FromContext(ctx).Info("deleted RoleBinding successfully", "roleBinding", trb.GetRBACName(), "cluster", cluster.GetName(), "namespace", roleBinding.Namespace)
		}
	}
	return nil
}

func (r TeamRoleBindingReconciler) deleteClusterRoleBinding(ctx context.Context, cl client.Client, trb *greenhousev1alpha2.TeamRoleBinding, cluster *greenhousev1alpha1.Cluster) error {
	remoteObject := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: trb.GetRBACName(),
		},
	}

	result, err := clientutil.Delete(ctx, cl, remoteObject)
	switch {
	case err != nil:
		log.FromContext(ctx).Error(err, "error deleting ClusterRoleBinding", "clusterRoleBinding", trb.GetRBACName(), "cluster", cluster.GetName())
		return err
	case result == clientutil.DeletionResultDeleted:
		log.FromContext(ctx).Info("deleted ClusterRoleBinding successfully", "clusterRoleBinding", trb.GetRBACName(), "cluster", cluster.GetName())
	}
	return nil
}

func (r TeamRoleBindingReconciler) deleteClusterRole(ctx context.Context, cl client.Client, trb *greenhousev1alpha2.TeamRoleBinding, cluster *greenhousev1alpha1.Cluster) error {
	remoteObject := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: greenhouseapis.RBACPrefix + trb.Spec.TeamRoleRef,
		},
	}
	result, err := clientutil.Delete(ctx, cl, remoteObject)
	switch {
	case err != nil:
		log.FromContext(ctx).Error(err, "error deleting ClusterRole", "clusterRole", trb.GetRBACName(), "cluster", cluster.GetName())
		return err
	case result == clientutil.DeletionResultDeleted:
		log.FromContext(ctx).Info("deleted ClusterRole successfully", "clusterRole", trb.GetRBACName(), "cluster", cluster.GetName())
	}
	return nil
}

// enqueueTeamRoleBindingsFor enqueues all TeamRoleBindings that are referenced by the given TeamRole or Team
func (r *TeamRoleBindingReconciler) enqueueTeamRoleBindingsFor(ctx context.Context, o client.Object) []ctrl.Request {
	fieldRef := ""
	// determine the field to select TeamRoleBindings by
	switch o.(type) {
	case *greenhousev1alpha1.TeamRole:
		fieldRef = greenhouseapis.RolebindingTeamRoleRefField
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
	teamRoleBindings := &greenhousev1alpha2.TeamRoleBindingList{}
	if err := r.List(ctx, teamRoleBindings, listOpts); err != nil {
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
	var allTeamRoleBindings = new(greenhousev1alpha2.TeamRoleBindingList)
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
func isClusterScoped(trb *greenhousev1alpha2.TeamRoleBinding) bool {
	return len(trb.Spec.Namespaces) == 0
}

// isRoleReferenced checks if the given TeamRoleBinding's TeamRole is still referenced by any Role or ClusterRole
func isRoleReferenced(ctx context.Context, c client.Client, teamRoleBinding *greenhousev1alpha2.TeamRoleBinding) (bool, error) {
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

// ListClusters returns the list of Clusters that match the ClusterSelector's Name or LabelSelector with applied ExcludeList.
// If the Name or LabelSelector does not return any cluster, an empty ClusterList is returned without error.
// If a cluster in the list is not ready, then it is removed from the list and the PropagationStatus updated.
func (r *TeamRoleBindingReconciler) listClusters(ctx context.Context, trb *greenhousev1alpha2.TeamRoleBinding) (*greenhousev1alpha1.ClusterList, error) {
	if trb.Spec.ClusterSelector.Name != "" {
		cluster := new(greenhousev1alpha1.Cluster)
		err := r.Get(ctx, types.NamespacedName{Name: trb.Spec.ClusterSelector.Name, Namespace: trb.GetNamespace()}, cluster)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return &greenhousev1alpha1.ClusterList{}, nil
			}
			return nil, err
		}
		return &greenhousev1alpha1.ClusterList{Items: []greenhousev1alpha1.Cluster{*cluster}}, nil
	}

	labelSelector, err := metav1.LabelSelectorAsSelector(&trb.Spec.ClusterSelector.LabelSelector)
	if err != nil {
		return nil, err
	}
	var clusters = new(greenhousev1alpha1.ClusterList)
	err = r.List(ctx, clusters, client.InNamespace(trb.GetNamespace()), client.MatchingLabelsSelector{Selector: labelSelector})
	if err != nil {
		return nil, err
	}
	// remove clusters which are not ready
	clusters.Items = slices.DeleteFunc(clusters.Items, func(c greenhousev1alpha1.Cluster) bool {
		if !c.Status.IsReadyTrue() {
			trb.SetPropagationStatus(c.GetName(), metav1.ConditionFalse, greenhousev1alpha2.ClusterConnectionFailed, "Cluster is not ready")
			return true
		}
		return false
	})
	return clusters, nil
}

// initTeamRoleBindingStatus ensures that all required conditions are present in the TeamRoleBinding's Status
func initTeamRoleBindingStatus(trb *greenhousev1alpha2.TeamRoleBinding) {
	for _, ct := range exposedConditions {
		if trb.Status.GetConditionByType(ct) == nil {
			trb.SetCondition(greenhousemetav1alpha1.UnknownCondition(ct, "", ""))
		}
	}
}

// computeReadyCondition computes the ReadyCondition based on the TeamRoleBinding's StatusConditions
func computeReadyCondition(status greenhousev1alpha2.TeamRoleBindingStatus) greenhousemetav1alpha1.Condition {
	readyCondition := *status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)

	for _, condition := range status.PropagationStatus {
		if condition.IsTrue() {
			continue // skip if the RBACReady condition is true for this cluster
		}
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Message = "Team RBAC reconciliation failed"
		return readyCondition
	}

	readyCondition.Status = metav1.ConditionTrue
	readyCondition.Message = "ready"
	return readyCondition
}
