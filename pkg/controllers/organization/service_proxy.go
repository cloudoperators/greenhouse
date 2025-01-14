// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization

import (
	"context"
	"crypto/rand"
	"encoding/base64"
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

const (
	serviceProxyName       = "service-proxy"
	cookieSecretKey        = "oauth2proxy-cookie-secret" //nolint:gosec
	oauthPreviewAnnotation = "greenhouse.sap/oauth-proxy-preview"
)

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

	// TODO: remove this once the feature is considered stable.
	// This allows to enable/disable the oauth-proxy feature for a specific organization
	oauthProxyEnabled := false
	if val, ok := org.GetAnnotations()[oauthPreviewAnnotation]; ok {
		oauthProxyEnabled = val == "true"
	}

	if oauthProxyEnabled {
		// oauth2-proxy requires OIDC Client config
		if org.Spec.Authentication == nil || org.Spec.Authentication.OIDCConfig == nil {
			log.FromContext(ctx).Info("skipping service-proxy Plugin reconciliation, Organization has no OIDCConfig")
			return nil
		}

		// oauth2-proxy requires a cookie secret, which needs to be provided from a secret
		secret := &corev1.Secret{}
		if err := r.Client.Get(ctx, types.NamespacedName{Name: org.Spec.Authentication.OIDCConfig.ClientSecretReference.Name, Namespace: org.Name}, secret); err != nil {
			log.FromContext(ctx).Info("failed to get Organization OIDC Secret", "error", err)
			return nil
		}
		if _, ok := secret.Data[cookieSecretKey]; !ok {
			cookieData, err := generateCookieSecret()
			if err != nil {
				log.FromContext(ctx).Info("failed to generate cookie secret", "error", err)
				return err
			}
			secret.Data[cookieSecretKey] = []byte(cookieData)
			if err := r.Client.Update(ctx, secret); err != nil {
				log.FromContext(ctx).Info("failed to update Organization OIDC Secret with cookie secret", "error", err)
				return fmt.Errorf("failed to update Organization OIDC Secret with cookie secret: %w", err)
			}
		}
	}

	plugin := &greenhousesapv1alpha1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceProxyName,
			Namespace: org.Name,
			Annotations: map[string]string{
				greenhousesapv1alpha1.AllowCreateAnnotation: "true",
			},
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
		if oauthProxyEnabled {
			oauthProxyValues := []greenhousesapv1alpha1.PluginOptionValue{
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
					Value: &apiextensionsv1.JSON{Raw: []byte(fmt.Sprintf("\"%s\"", org.Spec.Authentication.OIDCConfig.ClientSecretReference.Name))},
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

// generateCookieSecret generates a random cookie secret
func generateCookieSecret() (string, error) {
	// Generate 16 random bytes
	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	// Base64 encode the token twice
	encodedToken := base64.StdEncoding.EncodeToString(token)
	return base64.StdEncoding.EncodeToString([]byte(encodedToken)), nil
}
