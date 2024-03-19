// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhousesapv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/common"
	"github.com/cloudoperators/greenhouse/pkg/version"
)

// ServiceProxyReconciler reconciles a ServiceProxy PluginConfig for a Organization object
type ServiceProxyReconciler struct {
	client.Client
	recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=organizations,verbs=get;list;watch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=pluginconfigs,verbs=get;list;watch;create;update;patch;delete

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceProxyReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorderFor(name)
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousesapv1alpha1.Organization{}).
		Owns(&greenhousesapv1alpha1.PluginConfig{}).
		Complete(r)
}

func (r *ServiceProxyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx = clientutil.LogIntoContextFromRequest(ctx, req)

	var org = new(greenhousesapv1alpha1.Organization)
	if err := r.Get(ctx, req.NamespacedName, org); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.reconcileServiceProxy(ctx, org); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
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

	pluginConfig := &greenhousesapv1alpha1.PluginConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "service-proxy",
			Namespace: org.Name,
		},
		Spec: greenhousesapv1alpha1.PluginConfigSpec{
			Plugin: "service-proxy",
		},
	}

	result, err := clientutil.CreateOrPatch(ctx, r.Client, pluginConfig, func() error {
		pluginConfig.Spec.DisplayName = "Remote service proxy"
		pluginConfig.Spec.OptionValues = []greenhousesapv1alpha1.PluginOptionValue{
			{
				Name:  "domain",
				Value: &apiextensionsv1.JSON{Raw: domainJSON},
			},
			{
				Name:  "image.tag",
				Value: &apiextensionsv1.JSON{Raw: versionJSON},
			},
		}
		return controllerutil.SetControllerReference(org, pluginConfig, r.Scheme())
	})
	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created service-proxy PluginConfig", "name", pluginConfig.Name)
		r.recorder.Eventf(org, corev1.EventTypeNormal, "CreatedPluginConfig", "Created PluginConfig %s", pluginConfig.Name)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated service-proxy PluginConfig", "name", pluginConfig.Name)
		r.recorder.Eventf(org, corev1.EventTypeNormal, "UpdatedPluginConfig", "Updated PluginConfig %s", pluginConfig.Name)
	}
	return nil
}
