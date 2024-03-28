// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package teamrbac

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
)

// TeamRoleBindingReconciler reconciles a TeamRole object
type TeamRoleBindingReconciler struct {
	client.Client
	recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=teamrolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greenhouse.sap,resources=teamroles,verbs=get;list;watch;
//+kubebuilder:rbac:groups=greenhouse.sap,resources=teamrolebindings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=teamrolebindings/finalizers,verbs=update

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
		Complete(r)
}

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *TeamRoleBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	teamRoleBinding := &greenhousev1alpha1.TeamRoleBinding{}
	if err := r.Get(ctx, req.NamespacedName, teamRoleBinding); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	_ = log.FromContext(ctx)

	if teamRoleBinding.DeletionTimestamp != nil && controllerutil.ContainsFinalizer(teamRoleBinding, greenhouseapis.FinalizerCleanupTeamRoleBinding) {
		deletionSuccessful := false
		cluster, err := getCluster(ctx, r.Client, teamRoleBinding)
		if err != nil {
			r.recorder.Eventf(teamRoleBinding, corev1.EventTypeNormal, greenhousev1alpha1.ClusterNotFoundReason, "Error retrieving cluster for TeamRoleBinding %s", teamRoleBinding.GetName)
			return ctrl.Result{}, err
		}

		remoteRestClient, err := getK8sClient(ctx, r.Client, cluster)
		if err != nil {
			log.FromContext(ctx).Error(err, "Error getting client for cluster %s to delete TeamRoleBinding %s", cluster.GetName(), teamRoleBinding.GetName())
		}

		switch len(teamRoleBinding.Spec.Namespaces) > 0 {
		case true:
			for _, namespace := range teamRoleBinding.Spec.Namespaces {
				remoteObjectKey := types.NamespacedName{Name: teamRoleBinding.GetName(), Namespace: namespace}
				remoteObject := &rbacv1.RoleBinding{}
				err := remoteRestClient.Get(ctx, remoteObjectKey, remoteObject)
				switch {
				case apierrors.IsNotFound(err):
					continue
				case !apierrors.IsNotFound(err) && err != nil:
					r.recorder.Eventf(teamRoleBinding, corev1.EventTypeWarning, greenhousev1alpha1.FailedDeleteRoleBindingReason, "Error retrieving RoleBinding %s for cluster %s in namespace %s", teamRoleBinding.GetName(), cluster.GetName(), namespace)
					// TODO: collect errors and return them
					return ctrl.Result{}, err
				default:
					if err := remoteRestClient.Delete(ctx, remoteObject); err != nil {
						r.recorder.Eventf(teamRoleBinding, corev1.EventTypeWarning, greenhousev1alpha1.FailedDeleteRoleBindingReason, "Error deleting RoleBinding %s for cluster %s in namespace %s", teamRoleBinding.GetName(), cluster.GetName(), namespace)
						// TODO: collect errors and return them
						return ctrl.Result{}, err
					}
				}
			}
			deletionSuccessful = true
		case false:
			remoteObjectKey := types.NamespacedName{Name: teamRoleBinding.GetName()}
			remoteObject := &rbacv1.ClusterRoleBinding{}
			err := remoteRestClient.Get(ctx, remoteObjectKey, remoteObject)
			switch {
			case apierrors.IsNotFound(err):
				deletionSuccessful = true
			case err != nil:
				r.recorder.Eventf(teamRoleBinding, corev1.EventTypeWarning, greenhousev1alpha1.FailedDeleteRoleBindingReason, "Error retrieving ClusterRoleBinding %s for cluster %s", teamRoleBinding.GetName, cluster.GetName())
				// TODO: collect errors and return them
			default:
				if err := remoteRestClient.Delete(ctx, remoteObject); err != nil {
					r.recorder.Eventf(teamRoleBinding, corev1.EventTypeWarning, greenhousev1alpha1.FailedDeleteClusterRoleBindingReason, "Error deleting ClusterRoleBinding %s for cluster %s", teamRoleBinding.GetName, cluster.GetName())
					// TODO: collect errors and return them
					return ctrl.Result{}, err
				} else {
					deletionSuccessful = true
				}
			}
		}

		if deletionSuccessful {
			if err := clientutil.RemoveFinalizer(ctx, r.Client, teamRoleBinding, greenhouseapis.FinalizerCleanupTeamRoleBinding); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	if err := clientutil.EnsureFinalizer(ctx, r.Client, teamRoleBinding, greenhouseapis.FinalizerCleanupTeamRoleBinding); err != nil {
		return ctrl.Result{}, err
	}

	teamRole, err := getTeamRole(ctx, r.Client, r.recorder, teamRoleBinding)
	if err != nil {
		return ctrl.Result{}, err
	}

	cluster, err := getCluster(ctx, r.Client, teamRoleBinding)
	if err != nil {
		r.recorder.Eventf(teamRoleBinding, corev1.EventTypeNormal, greenhousev1alpha1.ClusterNotFoundReason, "Error retrieving cluster for TeamRoleBinding %s", teamRoleBinding.GetName)
		return ctrl.Result{}, err
	}

	team, err := getTeam(ctx, r.Client, teamRoleBinding)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.recorder.Eventf(teamRoleBinding, corev1.EventTypeNormal, greenhousev1alpha1.TeamNotFoundReason, "Team %s not found in Namespace %s for TeamRoleBinding %s", teamRoleBinding.Spec.TeamRef, teamRoleBinding.GetNamespace(), teamRoleBinding.GetName())
		}
		return ctrl.Result{}, err
	}

	cr := initRBACClusterRole(teamRole)

	remoteRestClient, err := getK8sClient(ctx, r.Client, cluster)
	if err != nil {
		r.recorder.Eventf(teamRoleBinding, corev1.EventTypeWarning, "ClusterClientError", "Error getting client for cluster %s to replicate %s", cluster.GetName(), teamRoleBinding.GetName())
	}

	if err := reconcileClusterRole(ctx, remoteRestClient, cluster, cr); err != nil {
		r.recorder.Eventf(teamRoleBinding, corev1.EventTypeWarning, greenhousev1alpha1.FailedReconcileClusterRoleReason, "Error reconciling ClusterRole %s in cluster %s", cr.GetName(), cluster.GetName())
		return ctrl.Result{}, err
	}

	switch len(teamRoleBinding.Spec.Namespaces) == 0 {
	case true:
		crb := rbacClusterRoleBinding(teamRoleBinding, cr, team)
		if err := reconcileClusterRoleBinding(ctx, remoteRestClient, cluster, crb); err != nil {
			r.recorder.Eventf(teamRoleBinding, corev1.EventTypeWarning, greenhousev1alpha1.FailedReconcileClusterRoleBindingReason, "Error reconciling ClusterRoleBinding %s in cluster %s: %s", crb.GetName(), cluster.GetName(), err.Error())
		}
	default:
		for _, namespace := range teamRoleBinding.Spec.Namespaces {
			rbacRoleBinding := rbacRoleBinding(teamRoleBinding, cr, team, namespace)
			if err := reconcileRoleBinding(ctx, remoteRestClient, cluster, rbacRoleBinding); err != nil {
				r.recorder.Eventf(teamRoleBinding, corev1.EventTypeWarning, greenhousev1alpha1.FailedReconcileRoleBindingReason, "Error reconciling RoleBinding %s in cluster %s: %s", rbacRoleBinding.GetName(), cluster.GetName(), err.Error())
			}
		}
	}

	// TODO: add status update logic here
	return ctrl.Result{}, nil
}

// initRBACClusterRole returns a ClusterRole that matches the spec defined by the Greenhouse Role
func initRBACClusterRole(teamRole *greenhousev1alpha1.TeamRole) *rbacv1.ClusterRole {
	clusterRole := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: rbacv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   greenhouseapis.RoleAndBindingNamePrefix + teamRole.GetName(),
			Labels: map[string]string{greenhouseapis.LabelKeyRole: teamRole.GetName()},
		},
		Rules: teamRole.DeepCopy().Spec.Rules,
	}
	return clusterRole
}

// rbacRoleBinding creates a rbacv1.RoleBinding for a rbacv1.ClusterRole, Team and Namespace
func rbacRoleBinding(teamRoleBinding *greenhousev1alpha1.TeamRoleBinding, clusterRole *rbacv1.ClusterRole, team *greenhousev1alpha1.Team, namespace string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      greenhouseapis.RoleAndBindingNamePrefix + teamRoleBinding.GetName(),
			Namespace: namespace,
			Labels:    map[string]string{greenhouseapis.LabelKeyRoleBinding: teamRoleBinding.GetName()},
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

// rbacClusterRoleBinding creates a rbacv1.RoleBinding for a rbacv1.ClusterRole, Team and Namespace
func rbacClusterRoleBinding(teamRoleBinding *greenhousev1alpha1.TeamRoleBinding, clusterRole *rbacv1.ClusterRole, team *greenhousev1alpha1.Team) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   greenhouseapis.RoleAndBindingNamePrefix + teamRoleBinding.GetName(),
			Labels: map[string]string{greenhouseapis.LabelKeyRoleBinding: teamRoleBinding.GetName()},
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

// getCluster returns the Cluster referenced by Name in the given TeamRoleBinding in the TeamRoleBinding's Namespace
func getCluster(ctx context.Context, c client.Client, teamRoleBinding *greenhousev1alpha1.TeamRoleBinding) (*greenhousev1alpha1.Cluster, error) {
	cluster := &greenhousev1alpha1.Cluster{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: teamRoleBinding.Namespace, Name: teamRoleBinding.Spec.ClusterName}, cluster); err != nil {
		return nil, fmt.Errorf("error finding cluster: %w", err)
	}
	return cluster, nil
}

// getK8sClient returns a client.Client for the given Cluster
func getK8sClient(ctx context.Context, c client.Client, cluster *greenhousev1alpha1.Cluster) (client.Client, error) {
	secret := new(corev1.Secret)
	if err := c.Get(ctx, types.NamespacedName{Name: cluster.GetSecretName(), Namespace: cluster.GetNamespace()}, secret); err != nil {
		return nil, err
	}

	restClientGetter, err := clientutil.NewRestClientGetterFromSecret(secret, cluster.GetNamespace(), clientutil.WithPersistentConfig())
	if err != nil {
		return nil, err
	}

	remoteRestClient, err := clientutil.NewK8sClientFromRestClientGetter(restClientGetter)
	if err != nil {
		return nil, err
	}
	return remoteRestClient, nil
}

// reconcileClusterRole creates or updates a ClusterRole in the Cluster the given client.Client is created for
func reconcileClusterRole(ctx context.Context, cl client.Client, c *greenhousev1alpha1.Cluster, cr *rbacv1.ClusterRole) error {
	remoteCR := &rbacv1.ClusterRole{}
	remoteCR.Name = cr.Name
	result, err := clientutil.CreateOrPatch(ctx, cl, remoteCR, func() error {
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
	remoteCRB := &rbacv1.ClusterRoleBinding{}
	remoteCRB.Name = crb.Name
	result, err := clientutil.CreateOrPatch(ctx, cl, remoteCRB, func() error {
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
	remoteRB := &rbacv1.RoleBinding{}
	remoteRB.Name = rb.Name
	remoteRB.Namespace = rb.Namespace
	result, err := clientutil.CreateOrPatch(ctx, cl, remoteRB, func() error {
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
