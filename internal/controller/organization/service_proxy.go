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

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
)

const (
	serviceProxyName       = "service-proxy"
	cookieSecretKey        = "oauth2proxy-cookie-secret" //nolint:gosec
	oauthPreviewAnnotation = "greenhouse.sap/oauth-proxy-preview"
)

func (r *OrganizationReconciler) reconcileServiceProxy(ctx context.Context, org *greenhousev1alpha1.Organization) error {
	domain := getOauthProxyURL(org.Name)
	domainJSON, err := json.Marshal(domain)
	if err != nil {
		return fmt.Errorf("failed to marshal domain: %w", err)
	}
	versionJSON, err := json.Marshal("7295bfa")
	if err != nil {
		return fmt.Errorf("failed to marshal version.GitCommit: %w", err)
	}
	replicaCount, err := json.Marshal(1)
	if err != nil {
		return fmt.Errorf("failed to marshal replica count: %w", err)
	}

	var pluginDefinition = new(greenhousev1alpha1.PluginDefinition)
	if err := r.Client.Get(ctx, types.NamespacedName{Name: serviceProxyName, Namespace: ""}, pluginDefinition); err != nil {
		if apierrors.IsNotFound(err) {
			log.FromContext(ctx).Info("plugin definition for service-proxy not found")
			return nil
		}
		log.FromContext(ctx).Info("failed to get plugin definition for service-proxy", "error", err)
		return nil
	}

	oauthProxyEnabled := isOauthProxyEnabled(org)

	if oauthProxyEnabled {
		// oauth2-proxy requires OIDC Client config
		if org.Spec.Authentication == nil || org.Spec.Authentication.OIDCConfig == nil {
			log.FromContext(ctx).Info("skipping service-proxy Plugin reconciliation, Organization has no OIDCConfig")
			return nil
		}

		// oauth2-proxy requires a cookie secret, which needs to be provided from a secret
		secret, err := r.getOrCreateOrgSecret(ctx, org)
		if err != nil {
			log.FromContext(ctx).Info("failed to get or create internal Organization Secret", "error", err)
			return err
		}
		if _, ok := secret.Data[cookieSecretKey]; !ok {
			cookieData, err := generateCookieSecret()
			if err != nil {
				log.FromContext(ctx).Info("failed to generate cookie secret", "error", err)
				return err
			}
			if secret.Data == nil {
				secret.Data = make(map[string][]byte)
			}
			secret.Data[cookieSecretKey] = []byte(cookieData)
			if err := r.Client.Update(ctx, secret); err != nil {
				log.FromContext(ctx).Info("failed to update Organization OIDC Secret with cookie secret", "error", err)
				return fmt.Errorf("failed to update Organization OIDC Secret with cookie secret: %w", err)
			}
		}
	}

	plugin := &greenhousev1alpha1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceProxyName,
			Namespace: org.Name,
		},
		Spec: greenhousev1alpha1.PluginSpec{
			PluginDefinition: serviceProxyName,
		},
	}

	result, err := clientutil.CreateOrPatch(ctx, r.Client, plugin, func() error {
		plugin.Spec.DisplayName = "Remote service proxy"
		plugin.Spec.OptionValues = []greenhousev1alpha1.PluginOptionValue{
			{
				Name:  "domain",
				Value: &apiextensionsv1.JSON{Raw: domainJSON},
			},
			{
				Name:  "image.tag",
				Value: &apiextensionsv1.JSON{Raw: versionJSON},
			},
			{
				Name:  "replicaCount",
				Value: &apiextensionsv1.JSON{Raw: replicaCount},
			},
		}
		if oauthProxyEnabled {
			oauthProxyValues := []greenhousev1alpha1.PluginOptionValue{
				{
					Name:  "oauth2proxy.enabled",
					Value: &apiextensionsv1.JSON{Raw: []byte("\"true\"")},
				},
				{
					Name:  "oauth2proxy.issuerURL",
					Value: &apiextensionsv1.JSON{Raw: []byte(fmt.Sprintf("\"%s\"", org.Spec.Authentication.OIDCConfig.Issuer))},
				},
				{
					Name:  "oauth2proxy.clientIDRef.secret",
					Value: &apiextensionsv1.JSON{Raw: []byte(fmt.Sprintf("\"%s\"", org.Spec.Authentication.OIDCConfig.ClientIDReference.Name))},
				},
				{
					Name:  "oauth2proxy.clientIDRef.key",
					Value: &apiextensionsv1.JSON{Raw: []byte(fmt.Sprintf("\"%s\"", org.Spec.Authentication.OIDCConfig.ClientIDReference.Key))},
				},
				{
					Name:  "oauth2proxy.clientSecretRef.secret",
					Value: &apiextensionsv1.JSON{Raw: []byte(fmt.Sprintf("\"%s\"", org.Spec.Authentication.OIDCConfig.ClientSecretReference.Name))},
				},
				{
					Name:  "oauth2proxy.clientSecretRef.key",
					Value: &apiextensionsv1.JSON{Raw: []byte(fmt.Sprintf("\"%s\"", org.Spec.Authentication.OIDCConfig.ClientSecretReference.Key))},
				},
				{
					Name:  "oauth2proxy.cookieSecretRef.secret",
					Value: &apiextensionsv1.JSON{Raw: []byte(fmt.Sprintf("\"%s\"", org.Name+technicalSecretSuffix))},
				},
				{
					Name:  "oauth2proxy.cookieSecretRef.key",
					Value: &apiextensionsv1.JSON{Raw: []byte(fmt.Sprintf("\"%s\"", cookieSecretKey))},
				},
			}
			plugin.Spec.OptionValues = append(plugin.Spec.OptionValues, oauthProxyValues...)
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

func (r *OrganizationReconciler) enqueueAllOrganizationsForServiceProxyPluginDefinition(ctx context.Context, _ client.Object) []ctrl.Request {
	return listOrganizationsAsReconcileRequests(ctx, r.Client)
}

func listOrganizationsAsReconcileRequests(ctx context.Context, c client.Client, listOpts ...client.ListOption) []ctrl.Request {
	var organizationList = new(greenhousev1alpha1.OrganizationList)
	if err := c.List(ctx, organizationList, listOpts...); err != nil {
		return nil
	}
	res := make([]ctrl.Request, len(organizationList.Items))
	for idx, organization := range organizationList.Items {
		res[idx] = ctrl.Request{NamespacedName: types.NamespacedName{Name: organization.Name, Namespace: organization.Namespace}}
	}
	return res
}
