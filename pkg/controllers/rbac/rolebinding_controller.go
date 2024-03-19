// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package rbac

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

	extensionsgreenhouse "github.com/cloudoperators/greenhouse/pkg/apis/extensions.greenhouse"
	extensionsgreenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/extensions.greenhouse/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
)

// RoleBindingReconciler reconciles a Role object
type RoleBindingReconciler struct {
	client.Client
	recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=extensions.greenhouse.sap,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=extensions.greenhouse.sap,resources=roles,verbs=get;list;watch;
//+kubebuilder:rbac:groups=extensions.greenhouse.sap,resources=rolebindings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=extensions.greenhouse.sap,resources=rolebindings/finalizers,verbs=update

// SetupWithManager sets up the controller with the Manager.
func (r *RoleBindingReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorderFor(name)

	// index RoleBindings by the RoleRef field for faster lookups
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &extensionsgreenhousev1alpha1.RoleBinding{}, extensionsgreenhouse.RolebindingRoleRefField, func(rawObj client.Object) []string {
		// Extract the Role name from the RoleBinding Spec, if one is provided
		roleBinding, ok := rawObj.(*extensionsgreenhousev1alpha1.RoleBinding)
		if roleBinding.Spec.RoleRef == "" || !ok {
			return nil
		}
		return []string{roleBinding.Spec.RoleRef}
	}); err != nil {
		return err
	}

	// index RoleBindings by the TeamRef field for faster lookups
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &extensionsgreenhousev1alpha1.RoleBinding{}, extensionsgreenhouse.RolebindingTeamRefField, func(rawObj client.Object) []string {
		// Extract the Role name from the RoleBinding Spec, if one is provided
		roleBinding, ok := rawObj.(*extensionsgreenhousev1alpha1.RoleBinding)
		if roleBinding.Spec.TeamRef == "" || !ok {
			return nil
		}
		return []string{roleBinding.Spec.TeamRef}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&extensionsgreenhousev1alpha1.RoleBinding{}).
		Watches(&extensionsgreenhousev1alpha1.Role{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueRoleBindingsFor)).
		Watches(&greenhousev1alpha1.Team{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueRoleBindingsFor)).
		Complete(r)
}

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *RoleBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	roleBinding := &extensionsgreenhousev1alpha1.RoleBinding{}
	if err := r.Get(ctx, req.NamespacedName, roleBinding); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	_ = log.FromContext(ctx)

	if roleBinding.DeletionTimestamp != nil && controllerutil.ContainsFinalizer(roleBinding, extensionsgreenhouse.FinalizerCleanupRoleBinding) {
		deletionSuccessful := true
		clusters, err := getClusters(ctx, r.Client, roleBinding)
		if err != nil {
			r.recorder.Eventf(roleBinding, corev1.EventTypeNormal, extensionsgreenhousev1alpha1.ClusterNotFoundReason, "Error listing clusters for RoleBinding %s", roleBinding.GetName)
			return ctrl.Result{}, err
		}

		for _, cluster := range clusters {
			cluster := cluster
			remoteRestClient, err := getK8sClient(ctx, r.Client, &cluster)
			if err != nil {
				log.FromContext(ctx).Error(err, "Error getting client for cluster %s to delete RoleBinding %s", cluster.GetName(), roleBinding.GetName())
			}

			switch len(roleBinding.Spec.Namespaces) > 0 {
			case true:
				for _, namespace := range roleBinding.Spec.Namespaces {
					remoteObjectKey := types.NamespacedName{Name: roleBinding.GetName(), Namespace: namespace}
					remoteObject := &rbacv1.RoleBinding{}
					err := remoteRestClient.Get(ctx, remoteObjectKey, remoteObject)
					switch {
					case apierrors.IsNotFound(err):
						continue
					case !apierrors.IsNotFound(err) && err != nil:
						r.recorder.Eventf(roleBinding, corev1.EventTypeWarning, extensionsgreenhousev1alpha1.FailedDeleteRoleBindingReason, "Error retrieving RoleBinding %s for cluster %s in namespace %s", roleBinding.GetName(), cluster.GetName(), namespace)
						// TODO(d059176): collect errors and return them
						deletionSuccessful = false
						continue
					}
					if err := remoteRestClient.Delete(ctx, remoteObject); err != nil {
						r.recorder.Eventf(roleBinding, corev1.EventTypeWarning, extensionsgreenhousev1alpha1.FailedDeleteRoleBindingReason, "Error deleting RoleBinding %s for cluster %s in namespace %s", roleBinding.GetName(), cluster.GetName(), namespace)
						// TODO(d059176): collect errors and return them
						deletionSuccessful = false
					}
				}
			case false:
				remoteObjectKey := types.NamespacedName{Name: roleBinding.GetName()}
				remoteObject := &rbacv1.ClusterRoleBinding{}
				err := remoteRestClient.Get(ctx, remoteObjectKey, remoteObject)
				switch {
				case apierrors.IsNotFound(err):
					continue
				case !apierrors.IsNotFound(err) && err != nil:
					r.recorder.Eventf(roleBinding, corev1.EventTypeWarning, extensionsgreenhousev1alpha1.FailedDeleteRoleBindingReason, "Error retrieving ClusterRoleBinding %s for cluster %s", roleBinding.GetName, cluster.GetName())
					// TODO(d059176): collect errors and return them
					deletionSuccessful = false
					continue
				}
				if err := remoteRestClient.Delete(ctx, remoteObject); err != nil {
					r.recorder.Eventf(roleBinding, corev1.EventTypeWarning, extensionsgreenhousev1alpha1.FailedDeleteClusterRoleBindingReason, "Error deleting ClusterRoleBinding %s for cluster %s", roleBinding.GetName, cluster.GetName())
					// TODO(d059176): collect errors and return them
					deletionSuccessful = false
				}
			}
		}
		if deletionSuccessful {
			if err := clientutil.RemoveFinalizer(ctx, r.Client, roleBinding, extensionsgreenhouse.FinalizerCleanupRoleBinding); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	if err := clientutil.EnsureFinalizer(ctx, r.Client, roleBinding, extensionsgreenhouse.FinalizerCleanupRoleBinding); err != nil {
		return ctrl.Result{}, err
	}

	role, err := getRole(ctx, r.Client, r.recorder, roleBinding)
	if err != nil {
		return ctrl.Result{}, err
	}

	clusters, err := getClusters(ctx, r.Client, roleBinding)
	if err != nil {
		r.recorder.Eventf(roleBinding, corev1.EventTypeNormal, extensionsgreenhousev1alpha1.ClusterNotFoundReason, "Error listing clusters for RoleBinding %s", roleBinding.GetName)
		return ctrl.Result{}, err
	}

	team, err := getTeam(ctx, r.Client, roleBinding)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.recorder.Eventf(roleBinding, corev1.EventTypeNormal, extensionsgreenhousev1alpha1.TeamNotFoundReason, "Team %s not found in Namespace %s for RoleBinding %s", roleBinding.Spec.TeamRef, roleBinding.GetNamespace(), roleBinding.GetName())
		}
		return ctrl.Result{}, err
	}

	cr := initRBACClusterRole(role)

	for _, cluster := range clusters {
		cluster := cluster
		remoteRestClient, err := getK8sClient(ctx, r.Client, &cluster)
		if err != nil {
			r.recorder.Eventf(roleBinding, corev1.EventTypeWarning, "ClusterClientError", "Error getting client for cluster %s to replicate %s", cluster.GetName(), roleBinding.GetName())
			continue
		}

		if err := reconcileClusterRole(ctx, remoteRestClient, &cluster, cr); err != nil {
			r.recorder.Eventf(roleBinding, corev1.EventTypeWarning, extensionsgreenhousev1alpha1.FailedReconcileClusterRoleReason, "Error reconciling ClusterRole %s in cluster %s", cr.GetName(), cluster.GetName())
			continue
		}

		switch len(roleBinding.Spec.Namespaces) == 0 {
		case true:
			crb := initRBACClusterRoleBinding(roleBinding, cr, team)
			if err := reconcileClusterRoleBinding(ctx, remoteRestClient, &cluster, crb); err != nil {
				r.recorder.Eventf(roleBinding, corev1.EventTypeWarning, extensionsgreenhousev1alpha1.FailedReconcileClusterRoleBindingReason, "Error reconciling ClusterRoleBinding %s in cluster %s: %s", crb.GetName(), cluster.GetName(), err.Error())
			}
		default:
			for _, namespace := range roleBinding.Spec.Namespaces {
				rbacRoleBinding := initRBACRoleBinding(roleBinding, cr, team, namespace)
				if err := reconcileRoleBinding(ctx, remoteRestClient, &cluster, rbacRoleBinding); err != nil {
					r.recorder.Eventf(roleBinding, corev1.EventTypeWarning, extensionsgreenhousev1alpha1.FailedReconcileRoleBindingReason, "Error reconciling RoleBinding %s in cluster %s: %s", rbacRoleBinding.GetName(), cluster.GetName(), err.Error())
				}
			}
		}
	}

	// TODO(d059176): add status update logic here
	// - [ ] Status of Role/RoleBinding w.r.t. Cluster??
	return ctrl.Result{}, nil
}

// initRBACClusterRole returns a ClusterRole that matches the spec defined by the Greenhouse Role
func initRBACClusterRole(role *extensionsgreenhousev1alpha1.Role) *rbacv1.ClusterRole {
	clusterRole := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: rbacv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   role.GetName(),
			Labels: map[string]string{extensionsgreenhouse.LabelKeyRole: role.GetName()},
		},
		Rules: role.DeepCopy().Spec.Rules,
	}
	return clusterRole
}

// initRBACRoleBinding creates a rbacv1.RoleBinding for a rbacv1.ClusterRole, Team and Namespace
func initRBACRoleBinding(roleBinding *extensionsgreenhousev1alpha1.RoleBinding, clusterRole *rbacv1.ClusterRole, team *greenhousev1alpha1.Team, namespace string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleBinding.GetName(),
			Namespace: namespace,
			Labels:    map[string]string{extensionsgreenhouse.LabelKeyRoleBinding: roleBinding.GetName()},
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

// initRBACClusterRoleBinding creates a rbacv1.RoleBinding for a rbacv1.ClusterRole, Team and Namespace
func initRBACClusterRoleBinding(roleBinding *extensionsgreenhousev1alpha1.RoleBinding, clusterRole *rbacv1.ClusterRole, team *greenhousev1alpha1.Team) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: roleBinding.GetName(),
			// TODO(d059176): add label to indicate this ClusterRoleBinding is managed by Greenhouse
			// Labels: []string{"extensions.greenhouse.sap.com/clusterrolebinding": roleBinding.GetName()
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

// getRole retrieves the Role referenced by the given RoleBinding in the RoleBinding's Namespace
func getRole(ctx context.Context, c client.Client, r record.EventRecorder, roleBinding *extensionsgreenhousev1alpha1.RoleBinding) (*extensionsgreenhousev1alpha1.Role, error) {
	if roleBinding.Spec.RoleRef == "" {
		r.Eventf(roleBinding, corev1.EventTypeNormal, "RoleReferenceMissing", "RoleBinding %s does not reference a Role", roleBinding.GetName())
		return nil, fmt.Errorf("error missing role reference for role %s", roleBinding.GetName())
	}

	role := &extensionsgreenhousev1alpha1.Role{}
	if err := c.Get(ctx, types.NamespacedName{Name: roleBinding.Spec.RoleRef, Namespace: roleBinding.GetNamespace()}, role); err != nil {
		return nil, fmt.Errorf("error getting role: %w", err)
	}
	return role, nil
}

// getTeam retrieves the Team referenced by the given RoleBinding in the RoleBinding's Namespace
func getTeam(ctx context.Context, c client.Client, roleBinding *extensionsgreenhousev1alpha1.RoleBinding) (*greenhousev1alpha1.Team, error) {
	if roleBinding.Spec.TeamRef == "" {
		return nil, fmt.Errorf("error missing team reference")
	}

	team := &greenhousev1alpha1.Team{}
	if err := c.Get(ctx, types.NamespacedName{Name: roleBinding.Spec.TeamRef, Namespace: roleBinding.GetNamespace()}, team); err != nil {
		return nil, fmt.Errorf("error getting team: %w", err)
	}
	return team, nil
}

// getClusters returns a List of Clusters that match the ClusterSelector of the given RoleBinding in the RoleBinding's Namespace
func getClusters(ctx context.Context, c client.Client, roleBinding *extensionsgreenhousev1alpha1.RoleBinding) ([]greenhousev1alpha1.Cluster, error) {
	selector, err := metav1.LabelSelectorAsSelector(&roleBinding.Spec.ClusterSelector)
	if err != nil {
		return nil, fmt.Errorf("error converting ClusterSelector: %w", err)
	}

	listOpts := &client.ListOptions{LabelSelector: selector, Namespace: roleBinding.GetNamespace()}

	clusterList := &greenhousev1alpha1.ClusterList{}
	if err := c.List(ctx, clusterList, listOpts); err != nil {
		return nil, fmt.Errorf("error listing clusters: %w", err)
	}
	return clusterList.Items, nil
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

// enqueueRoleBindingsFor enqueues all RoleBindings that are referenced by the given Role or Team
func (r *RoleBindingReconciler) enqueueRoleBindingsFor(ctx context.Context, o client.Object) []ctrl.Request {
	fieldRef := ""
	// determine the field to select RoleBindings by
	switch o.(type) {
	case *extensionsgreenhousev1alpha1.Role:
		fieldRef = extensionsgreenhouse.RolebindingRoleRefField
	case *greenhousev1alpha1.Team:
		fieldRef = extensionsgreenhouse.RolebindingTeamRefField
	default:
		return []ctrl.Request{}
	}
	listOpts := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(fieldRef, o.GetName()),
		Namespace:     o.GetNamespace(),
	}
	// list all referenced RoleBindings
	roleBindings := &extensionsgreenhousev1alpha1.RoleBindingList{}
	if err := r.Client.List(ctx, roleBindings, listOpts); err != nil {
		return []ctrl.Request{}
	}

	// return a list of reconcile.Requests for the list of referenced RoleBindings
	requests := make([]ctrl.Request, len(roleBindings.Items))
	for i, roleBinding := range roleBindings.Items {
		requests[i] = ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      roleBinding.GetName(),
				Namespace: roleBinding.GetNamespace(),
			},
		}
	}
	return requests
}
