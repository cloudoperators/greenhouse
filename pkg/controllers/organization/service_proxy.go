// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhousesapv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/common"
	"github.com/cloudoperators/greenhouse/pkg/version"
)

const serviceProxyName = "service-proxy"

func (r *OrganizationReconciler) reconcileServiceProxy(ctx context.Context, org *greenhousesapv1alpha1.Organization) error {
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

func (r *OrganizationReconciler) enqueueAllOrganizationsForServiceProxyPluginDefinition(ctx context.Context, o client.Object) []ctrl.Request {
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
