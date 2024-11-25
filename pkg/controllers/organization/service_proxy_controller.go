// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	greenhousesapv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/common"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
	"github.com/cloudoperators/greenhouse/pkg/version"
)

const serviceProxyName = "service-proxy"

// ServiceProxyReconciler reconciles a ServiceProxy Plugin for a Organization object
type ServiceProxyReconciler struct {
	client.Client
	recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=organizations,verbs=get;list;watch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugins,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugindefinitions,verbs=get;list;watch

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceProxyReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorderFor(name)
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousesapv1alpha1.Organization{}).
		Owns(&greenhousesapv1alpha1.Plugin{}).
		// If the service-proxy PluginDefinition was changed, reconcile all Organizations.
		Watches(&greenhousesapv1alpha1.PluginDefinition{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueAllOrganizationsForServiceProxyPluginDefinition),
			builder.WithPredicates(predicate.And(
				clientutil.PredicateByName(serviceProxyName),
				predicate.GenerationChangedPredicate{},
			))).
		Complete(r)
}

func (r *ServiceProxyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return lifecycle.Reconcile(ctx, r.Client, req.NamespacedName, &greenhousesapv1alpha1.Organization{}, r, noStatus())
}

func (r *ServiceProxyReconciler) EnsureDeleted(_ context.Context, _ lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	return ctrl.Result{}, lifecycle.Success, nil // nothing to do in that case
}

func (r *ServiceProxyReconciler) EnsureCreated(ctx context.Context, object lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	org, ok := object.(*greenhousesapv1alpha1.Organization)
	if !ok {
		return ctrl.Result{}, lifecycle.Failed, errors.Errorf("RuntimeObject has incompatible type.")
	}

	if err := r.reconcileServiceProxy(ctx, org); err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}

	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *ServiceProxyReconciler) reconcileServiceProxy(ctx context.Context, org *greenhousesapv1alpha1.Organization) error {
	domain := fmt.Sprintf("%s.%s", org.Name, common.DNSDomain)
	domainJSON, err := json.Marshal(domain)
	if err != nil {
		return fmt.Errorf("failed to marshal domain: %w", err)
	}
	versionJSON, err := json.Marshal(version.GitCommit)
	if err != nil {
		return fmt.Errorf("failed to marshal version.GitCommit: %w", err)
	}

	var pluginDefinition = new(greenhousesapv1alpha1.PluginDefinition)
	if err := r.Client.Get(ctx, types.NamespacedName{Name: serviceProxyName, Namespace: ""}, pluginDefinition); err != nil {
		if apierrors.IsNotFound(err) {
			log.FromContext(ctx).Info("plugin definition for service-proxy not found")
			return nil
		}
		log.FromContext(ctx).Info("failed to get plugin definition for service-proxy", "error", err)
		return nil
	}

	plugin := &greenhousesapv1alpha1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceProxyName,
			Namespace: org.Name,
		},
		Spec: greenhousesapv1alpha1.PluginSpec{
			PluginDefinition: serviceProxyName,
		},
	}

	result, err := clientutil.CreateOrPatch(ctx, r.Client, plugin, func() error {
		plugin.Spec.DisplayName = "Remote service proxy"
		plugin.Spec.OptionValues = []greenhousesapv1alpha1.PluginOptionValue{
			{
				Name:  "domain",
				Value: &apiextensionsv1.JSON{Raw: domainJSON},
			},
			{
				Name:  "image.tag",
				Value: &apiextensionsv1.JSON{Raw: versionJSON},
			},
		}
		return controllerutil.SetControllerReference(org, plugin, r.Scheme())
	})
	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created service-proxy Plugin", "name", plugin.Name)
		r.recorder.Eventf(org, corev1.EventTypeNormal, "CreatedPlugin", "Created Plugin %s", plugin.Name)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated service-proxy Plugin", "name", plugin.Name)
		r.recorder.Eventf(org, corev1.EventTypeNormal, "UpdatedPlugin", "Updated Plugin %s", plugin.Name)
	}
	return nil
}

func (r *ServiceProxyReconciler) enqueueAllOrganizationsForServiceProxyPluginDefinition(ctx context.Context, o client.Object) []ctrl.Request {
	return listOrganizationsAsReconcileRequests(ctx, r.Client)
}

func listOrganizationsAsReconcileRequests(ctx context.Context, c client.Client, listOpts ...client.ListOption) []ctrl.Request {
	var organizationList = new(greenhousesapv1alpha1.OrganizationList)
	if err := c.List(ctx, organizationList, listOpts...); err != nil {
		return nil
	}
	res := make([]ctrl.Request, len(organizationList.Items))
	for idx, organization := range organizationList.Items {
		res[idx] = ctrl.Request{NamespacedName: types.NamespacedName{Name: organization.Name, Namespace: organization.Namespace}}
	}
	return res
}
