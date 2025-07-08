// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package flux

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	sourcecontroller "github.com/fluxcd/source-controller/api/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/controller/cluster/utils"
)

const (
	pluginDefinitionURLField = "spec.url"
)

// PluginDefinitionReconciler reconciles plugindefinitions and translates them into Flux resources
type PluginDefinitionReconciler struct {
	client.Client
}

// Greenhouse related RBAC rules for the FluxReconciler
//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugindefinitions,verbs=get;list;watch

// Flux related RBAC rules for the FluxReconciler
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=helmrepositories,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=helmrepositories/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=helmrepositories/finalizers,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// SetupWithManager sets up the controller with the Manager.
func (r *PluginDefinitionReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()

	// index PluginDefinitions by the URL field for faster lookups
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &sourcecontroller.HelmRepository{}, pluginDefinitionURLField, func(rawObj client.Object) []string {
		helmRepository, ok := rawObj.(*sourcecontroller.HelmRepository)
		if helmRepository.Spec.URL == "" || !ok {
			return nil
		}
		return []string{helmRepository.Spec.URL}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousev1alpha1.PluginDefinition{}).
		WithEventFilter(predicate.Or(predicate.GenerationChangedPredicate{}, predicate.LabelChangedPredicate{}, predicate.AnnotationChangedPredicate{})).
		Complete(r)
}

// Reconcile reads the PluginDefinition object and makes creates a HelmRepository and a HelmChart flux object
func (r *PluginDefinitionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var pluginDefinition = new(greenhousev1alpha1.PluginDefinition)
	var namespace string

	if err := r.Get(ctx, req.NamespacedName, pluginDefinition); err != nil {
		log.FromContext(ctx).Error(err, "Failed to get pluginDefinition")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	switch {
	case req.Namespace != "":
		namespace = req.Namespace
	default:
		// if the namespace is empty, we use the default namespace
		// preparation for the future when we will use the namespaced plugin definitions
		namespace = defaultNameSpace
	}

	// reconcileHelmRepository is creating or updating the helmRepository out form PluginDefinition spec
	if err := r.reconcileHelmRepository(ctx, pluginDefinition, namespace); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: utils.DefaultRequeueInterval}, nil
}

func (r *PluginDefinitionReconciler) reconcileHelmRepository(ctx context.Context, pluginDef *greenhousev1alpha1.PluginDefinition, namespace string) error {
	var helmRepository = new(sourcecontroller.HelmRepository)

	result, err := clientutil.CreateOrPatch(ctx, r.Client, helmRepository, func() error {
		helmRepository.Name, helmRepository.Spec.Type = convertName(pluginDef.Spec.HelmChart.Repository)
		helmRepository.Namespace = namespace
		helmRepository.Spec.Interval = metav1.Duration{Duration: 5 * time.Minute}
		helmRepository.Spec.URL = pluginDef.Spec.HelmChart.Repository
		return controllerutil.SetOwnerReference(pluginDef, helmRepository, r.Scheme())
	})
	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("Created helmRepository", "name", helmRepository.Name)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("Updated helmRepository", "name", helmRepository.Name)
	case clientutil.OperationResultNone:
		log.FromContext(ctx).Info("No changes to helmRepository", "name", helmRepository.Name)
	}

	return nil
}
