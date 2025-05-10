// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/dexidp/dex/storage"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/common"
	"github.com/cloudoperators/greenhouse/internal/version"
)

const (
	serviceProxyName       = "service-proxy"
	oauthPreviewAnnotation = "greenhouse.sap/oauth-proxy-preview"
	cookieSecretKey        = "oauth2proxy-cookie-secret" //nolint:gosec
	// internalSuffix is used for the internal secret of the organization
	// this secret is used to store secrets that are not created by the user
	technicalSecretSuffix         = "-internal"
	dexOAuth2ProxyClientPrefix    = "oauth2-proxy-"
	dexOAuth2ProxyClientIDKey     = "oauth2proxy-clientID"
	dexOAuth2ProxyClientSecretKey = "oauth2proxy-clientSecret" //nolint:gosec
)

func (r *OrganizationReconciler) reconcileServiceProxy(ctx context.Context, org *greenhousev1alpha1.Organization) error {
	if err := r.reconcileOAuth2ProxySecret(ctx, org); err != nil {
		return err
	}
	if err := r.reconcileServiceProxyPlugin(ctx, org); err != nil {
		return err
	}
	return nil
}

// reconcileOAuth2ProxySecret - creates oauth2client redirect in dex for oauth2-proxy to authenticate
func (r *OrganizationReconciler) reconcileOAuth2ProxySecret(ctx context.Context, org *greenhousev1alpha1.Organization) error {
	if !isOauthProxyEnabled(org) {
		log.FromContext(ctx).Info("oauth-proxy feature is disabled for the organization")
		return nil
	}
	intSecret, err := r.getOrCreateOrgSecret(ctx, org)
	if err != nil {
		return err
	}
	if _, ok := intSecret.Data[cookieSecretKey]; !ok {
		cookieData, err := generateCookieSecret()
		if err != nil {
			log.FromContext(ctx).Info("failed to generate oauth2 proxy cookie secret", "name", org.Name, "error", err)
			return err
		}
		if intSecret.Data == nil {
			intSecret.Data = make(map[string][]byte)
		}
		intSecret.Data[cookieSecretKey] = []byte(cookieData)
	}
	oAuthProxyClientName := fmt.Sprintf("%s-%s", dexOAuth2ProxyClientPrefix, org.Name)
	intSecret.Data[dexOAuth2ProxyClientIDKey] = []byte(oAuthProxyClientName)
	oauthProxyClientSecret, err := generateOauth2ProxySecret()
	if err != nil {
		log.FromContext(ctx).Error(err, "failed to create oauth2 proxy client secret for org", "name", org.Name)
		return err
	}
	_, sOK := intSecret.Data[dexOAuth2ProxyClientSecretKey]
	if !sOK {
		intSecret.Data[dexOAuth2ProxyClientSecretKey] = []byte(oauthProxyClientSecret)
	}

	oAuthProxyClient, err := r.dex.GetClient(oAuthProxyClientName)
	oAuthProxyCallbackURL := fmt.Sprintf("https://auth-proxy.%s/oauth2/callback", getOauthProxyURL(org.Name))
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			if err = r.dex.CreateClient(ctx, storage.Client{
				Public:       false,
				ID:           oAuthProxyClientName,
				Name:         org.Name + " Service Proxy",
				RedirectURIs: []string{oAuthProxyCallbackURL}, // add service proxy redirect URI
				Secret:       oauthProxyClientSecret,
			}); err != nil {
				log.FromContext(ctx).Error(err, "failed to create oauth-proxy client credentials", "name", org.Name)
				return err
			}
			log.FromContext(ctx).Info("successfully created oauth-proxy client credentials", "name", org.Name)
			return nil
		}
		log.FromContext(ctx).Error(err, "failed to get oauth2client", "name", org.Name)
		return err
	}

	if err = r.dex.UpdateClient(oAuthProxyClient.ID, func(authClient storage.Client) (storage.Client, error) {
		authClient.Public = false
		authClient.Secret = string(intSecret.Data[dexOAuth2ProxyClientSecretKey])
		authClient.RedirectURIs = []string{oAuthProxyCallbackURL}
		authClient.Name = org.Name + " Service Proxy"
		return authClient, nil
	}); err != nil {
		log.FromContext(ctx).Error(err, "failed to update oauth-proxy client credentials", "name", org.Name)
		return err
	}
	log.FromContext(ctx).Info("successfully updated oauth-proxy client credentials", "name", org.Name)

	if err := r.Update(ctx, intSecret); err != nil {
		log.FromContext(ctx).Error(err, "failed to update oauth2-proxy secret", "name", org.Name)
		return err
	}
	return nil
}

func (r *OrganizationReconciler) reconcileServiceProxyPlugin(ctx context.Context, org *greenhousev1alpha1.Organization) error {
	domain := getOauthProxyURL(org.Name)
	domainJSON, err := json.Marshal(domain)
	if err != nil {
		return fmt.Errorf("failed to marshal domain: %w", err)
	}
	versionJSON, err := json.Marshal(version.GitCommit)
	if err != nil {
		return fmt.Errorf("failed to marshal version.GitCommit: %w", err)
	}

	var pluginDefinition = new(greenhousev1alpha1.PluginDefinition)
	if err := r.Get(ctx, types.NamespacedName{Name: serviceProxyName, Namespace: ""}, pluginDefinition); err != nil {
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
		}
		/*
			if operating under default cluster dns domain, we may need to set provider specific ingress annotations
			example below for gardener
			gardenAnnotations := map[string]string{
						"cert.gardener.cloud/purpose": "managed",
						"dns.gardener.cloud/class":    "garden",
						"dns.gardener.cloud/dnsnames": "*",
						"dns.gardener.cloud/ttl":      "600",
					}
					gardenAnnBytes, err := json.Marshal(gardenAnnotations)
					if err != nil {
						return fmt.Errorf("failed to marshal garden annotations: %w", err)
					}
					ingressOptionValue := greenhousev1alpha1.PluginOptionValue{
						Name:  "ingress.annotations",
						Value: &apiextensionsv1.JSON{Raw: gardenAnnBytes},
					}
		*/
		if oauthProxyEnabled {
			oauth2ProxyInternalSecretName := getInternalSecretName(org.GetName())
			oauthProxyValues := []greenhousev1alpha1.PluginOptionValue{
				// ingressOptionValue,
				{
					Name:  "oauth2proxy.enabled",
					Value: &apiextensionsv1.JSON{Raw: []byte("\"true\"")},
				},
				{
					Name:  "oauth2proxy.issuerURL",
					Value: &apiextensionsv1.JSON{Raw: []byte(fmt.Sprintf("\"%s\"", "https://auth."+common.DNSDomain))},
				},
				{
					Name:  "oauth2proxy.clientIDRef.secret",
					Value: &apiextensionsv1.JSON{Raw: []byte(fmt.Sprintf("\"%s\"", oauth2ProxyInternalSecretName))},
				},
				{
					Name:  "oauth2proxy.clientIDRef.key",
					Value: &apiextensionsv1.JSON{Raw: []byte(fmt.Sprintf("\"%s\"", dexOAuth2ProxyClientIDKey))},
				},
				{
					Name:  "oauth2proxy.clientSecretRef.secret",
					Value: &apiextensionsv1.JSON{Raw: []byte(fmt.Sprintf("\"%s\"", oauth2ProxyInternalSecretName))},
				},
				{
					Name:  "oauth2proxy.clientSecretRef.key",
					Value: &apiextensionsv1.JSON{Raw: []byte(fmt.Sprintf("\"%s\"", dexOAuth2ProxyClientSecretKey))},
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

// getOrCreateOrgSecret creates the internal secret of an organization, used to store secrets that are not created by the user.
// The secret is created with the name <org.Name>-internal and the namespace <org.Namespace>.
func (r *OrganizationReconciler) getOrCreateOrgSecret(ctx context.Context, org *greenhousev1alpha1.Organization) (*corev1.Secret, error) {
	secret := new(corev1.Secret)
	secret.Name = org.Name + technicalSecretSuffix
	secret.Namespace = org.Name
	secret.Type = greenhouseapis.SecretTypeOrganization

	// check if the secret already exists
	err := r.Get(ctx, types.NamespacedName{Namespace: secret.Namespace, Name: secret.Name}, secret)
	if err == nil {
		return secret, nil
	}
	if !apierrors.IsNotFound(err) {
		return nil, err
	}

	result, err := clientutil.CreateOrPatch(ctx, r.Client, secret, func() error {
		return controllerutil.SetControllerReference(org, secret, r.Scheme())
	})
	if err != nil {
		return nil, err
	}
	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created secret", "name", secret.Name)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated secret", "name", secret.Name)
	}
	return secret, nil
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

func getInternalSecretName(orgName string) string {
	return orgName + technicalSecretSuffix
}

// TODO: remove this once the feature is considered stable.
// This allows to enable/disable the oauth-proxy feature for a specific organization
// isOauthProxyEnabled - checks if the oauth-proxy feature is enabled for the organization
func isOauthProxyEnabled(org *greenhousev1alpha1.Organization) bool {
	oauthProxyEnabled := false
	if val, ok := org.GetAnnotations()[oauthPreviewAnnotation]; ok {
		oauthProxyEnabled = val == "true"
	}
	return oauthProxyEnabled
}

func generateOauth2ProxySecret() (string, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("failed to generate oauth proxy client secret: %w", err)
	}
	return base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(secret), nil
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

func getOauthProxyURL(orgName string) string {
	return fmt.Sprintf("%s.%s", orgName, common.DNSDomain)
}
