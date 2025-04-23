// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/dexidp/dex/connector/oidc"
	"github.com/dexidp/dex/storage"
	"github.com/pkg/errors"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/common"
)

const dexConnectorTypeGreenhouse = "greenhouse-oidc"

func (r *OrganizationReconciler) discoverOIDCRedirectURL(ctx context.Context, org *greenhousev1alpha1.Organization) (string, error) {
	if r := org.Spec.Authentication.OIDCConfig.RedirectURI; r != "" {
		return r, nil
	}
	var ingressList = new(networkingv1.IngressList)
	if err := r.List(ctx, ingressList, client.InNamespace(r.Namespace), client.MatchingLabels{"app.kubernetes.io/name": "idproxy"}); err != nil {
		return "", err
	}
	for _, ing := range ingressList.Items {
		for _, rule := range ing.Spec.Rules {
			if rule.Host != "" {
				return ensureCallbackURL(rule.Host), nil
			}
		}
	}
	return "", errors.New("oidc redirect URL not provided and cannot be discovered")
}

// removeAuthRedirectFromDefaultConnector - removes oauth redirects of the org being deleted
// in the default connector's OAuth2Client
func (r *OrganizationReconciler) removeAuthRedirectFromDefaultConnector(ctx context.Context, org *greenhousev1alpha1.Organization) error {
	defaultClient, err := r.dex.GetClient(defaultGreenhouseConnectorID)
	if err != nil {
		log.FromContext(ctx).Error(err, "failed to get default oauth2client", "name", defaultGreenhouseConnectorID)
		return err
	}
	err = r.dex.UpdateClient(defaultClient.Name, func(authClient storage.Client) (storage.Client, error) {
		orgRedirect := getRedirectForOrg(org.Name)
		updatedRedirects := slices.DeleteFunc(authClient.RedirectURIs, func(s string) bool {
			return s == orgRedirect
		})
		authClient.RedirectURIs = updatedRedirects
		return authClient, nil
	})
	if err != nil {
		log.FromContext(ctx).Error(err, "failed to update default connector's oauth2client redirects", "ID", defaultGreenhouseConnectorID)
		return err
	}
	log.FromContext(ctx).Info("successfully removed redirects from default connector's oauth2client redirects", "ID", defaultGreenhouseConnectorID)
	return nil
}

func (r *OrganizationReconciler) deleteDexConnector(ctx context.Context, org *greenhousev1alpha1.Organization) error {
	if err := r.dex.DeleteConnector(org.Name); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			log.FromContext(ctx).Info("dex connector not found", "name", org.Name)
			return nil
		}
		log.FromContext(ctx).Error(err, "failed to delete dex connector", "name", org.Name)
		return err
	}
	log.FromContext(ctx).Info("successfully deleted dex connector", "name", org.Name)
	return nil
}

func (r *OrganizationReconciler) deleteOAuth2Client(ctx context.Context, org *greenhousev1alpha1.Organization) error {
	if err := r.dex.DeleteClient(org.Name); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			log.FromContext(ctx).Info("oauth2client not found", "name", org.Name)
			return nil
		}
		log.FromContext(ctx).Error(err, "failed to delete oauth2client", "name", org.Name)
		return err
	}
	log.FromContext(ctx).Info("successfully deleted oauth2client", "name", org.Name)
	return nil
}

// reconcileDexConnector - creates or updates dex connector
func (r *OrganizationReconciler) reconcileDexConnector(ctx context.Context, org *greenhousev1alpha1.Organization) error {
	clientID, err := clientutil.GetSecretKeyFromSecretKeyReference(ctx, r.Client, org.Name, org.Spec.Authentication.OIDCConfig.ClientIDReference)
	if err != nil {
		return err
	}
	clientSecret, err := clientutil.GetSecretKeyFromSecretKeyReference(ctx, r.Client, org.Name, org.Spec.Authentication.OIDCConfig.ClientSecretReference)
	if err != nil {
		return err
	}
	redirectURL, err := r.discoverOIDCRedirectURL(ctx, org)
	if err != nil {
		return err
	}
	oidcConfig := &oidc.Config{
		Issuer:               org.Spec.Authentication.OIDCConfig.Issuer,
		ClientID:             clientID,
		ClientSecret:         clientSecret,
		RedirectURI:          redirectURL,
		UserNameKey:          "login_name",
		UserIDKey:            "login_name",
		InsecureSkipVerify:   true,
		InsecureEnableGroups: true,
	}
	configByte, err := json.Marshal(oidcConfig)
	if err != nil {
		return err
	}
	oidcConnector, err := r.dex.GetConnector(org.Name)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			if err = r.dex.CreateConnector(ctx, storage.Connector{
				ID:     org.Name,
				Type:   dexConnectorTypeGreenhouse,
				Name:   cases.Title(language.English).String(org.Name),
				Config: configByte,
			}); err != nil {
				log.FromContext(ctx).Error(err, "failed to create dex connector", "name", org.Name)
				return err
			}
			log.FromContext(ctx).Info("successfully created dex connector", "name", org.Name)
			return nil
		}
		log.FromContext(ctx).Error(err, "failed to get dex connector", "name", org.Name)
		return err
	}
	if err = r.dex.UpdateConnector(oidcConnector.ID, func(c storage.Connector) (storage.Connector, error) {
		c.ID = org.Name
		c.Type = dexConnectorTypeGreenhouse
		c.Name = cases.Title(language.English).String(org.Name)
		c.Config = configByte
		return c, nil
	}); err != nil {
		log.FromContext(ctx).Error(err, "failed to update dex connector", "name", org.Name)
		return err
	}
	log.FromContext(ctx).Info("successfully updated dex connector", "name", org.Name)
	return nil
}

// reconcileOAuth2Client - creates or updates oauth2client
func (r *OrganizationReconciler) reconcileOAuth2Client(ctx context.Context, org *greenhousev1alpha1.Organization) error {
	oAuthClient, err := r.dex.GetClient(org.Name)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			if err = r.dex.CreateClient(ctx, storage.Client{
				Public:       true,
				ID:           org.Name,
				Name:         org.Name,
				RedirectURIs: getRedirects(org.Name, org.Spec.Authentication.OIDCConfig.OAuth2ClientRedirectURIs),
			}); err != nil {
				log.FromContext(ctx).Error(err, "failed to create oauth2client", "name", org.Name)
				return err
			}
			log.FromContext(ctx).Info("successfully created oauth2client", "name", org.Name)
			return nil
		}
		log.FromContext(ctx).Error(err, "failed to get oauth2client", "name", org.Name)
		return err
	}
	if err = r.dex.UpdateClient(oAuthClient.ID, func(authClient storage.Client) (storage.Client, error) {
		authClient.Public = true
		authClient.ID = org.Name
		authClient.Name = org.Name
		redirects := getRedirects(org.Name, org.Spec.Authentication.OIDCConfig.OAuth2ClientRedirectURIs)
		// this ensures that reconciling the default connector does not remove the org specific redirect URIs
		if authClient.ID == defaultGreenhouseConnectorID {
			redirects = appendRedirects(redirects, authClient.RedirectURIs...)
		}
		authClient.RedirectURIs = redirects
		return authClient, nil
	}); err != nil {
		log.FromContext(ctx).Error(err, "failed to update oauth2client", "name", org.Name)
		return err
	}
	log.FromContext(ctx).Info("successfully updated oauth2client", "name", org.Name)
	return nil
}

// appendRedirectsToDefaultConnector - appends new organization's OAuth2Client redirect URIs into the default OAuth2Client redirect URIs
// NOTE: this has to be separate and should not be used with in any dex.UpdateClient transaction as it does not support concurrent updates
// It is also not safe when using MaxConcurrentReconciles > 1 as the default connector's redirect URIs can be updated concurrently and
// the last update will win
func (r *OrganizationReconciler) appendRedirectsToDefaultConnector(ctx context.Context, orgName string) error {
	defaultOAuthClient, err := r.dex.GetClient(defaultGreenhouseConnectorID)
	if err != nil {
		log.FromContext(ctx).Error(err, "failed to get default connector's oauth2client", "ID", defaultGreenhouseConnectorID)
		return err
	}
	orgRedirect := getRedirectForOrg(orgName)
	err = r.dex.UpdateClient(defaultOAuthClient.Name, func(authClient storage.Client) (storage.Client, error) {
		appendedRedirects := appendRedirects(authClient.RedirectURIs, orgRedirect)
		authClient.RedirectURIs = appendedRedirects
		return authClient, nil
	})
	if err != nil {
		log.FromContext(ctx).Error(err, "failed to update default connector's oauth2client redirects", "ID", defaultGreenhouseConnectorID)
		return err
	}
	log.FromContext(ctx).Info("successfully updated default connector's oauth2client redirects", "ID", defaultGreenhouseConnectorID)
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
	oAuthProxyClientName := fmt.Sprintf("oauth2-proxy-%s", org.Name)
	intSecret.Data["oauth2proxy-clientID"] = []byte(oAuthProxyClientName)
	oauthProxyClientSecret, err := generateOauth2ProxySecret()
	if err != nil {
		log.FromContext(ctx).Error(err, "failed to create oauth2 proxy client secret for org", "name", org.Name)
		return err
	}
	_, sOK := intSecret.Data["oauth2proxy-clientSecret"]
	if !sOK {
		intSecret.Data["oauth2proxy-clientSecret"] = []byte(oauthProxyClientSecret)
	}

	oAuthProxyClient, err := r.dex.GetClient(oAuthProxyClientName)
	oAuthProxyCallbackURL := fmt.Sprintf("https://%s/oauth2/callback", getOauthProxyURL(org.Name))
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			if err = r.dex.CreateClient(ctx, storage.Client{
				Public:       true,
				ID:           oAuthProxyClientName,
				Name:         fmt.Sprintf("%s Service Proxy", org.Name),
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
		authClient.Public = true
		authClient.Secret = string(intSecret.Data["oauth2proxy-clientSecret"])
		authClient.RedirectURIs = []string{oAuthProxyCallbackURL}
		return authClient, nil
	}); err != nil {
		log.FromContext(ctx).Error(err, "failed to update oauth-proxy client credentials", "name", org.Name)
		return err
	}
	log.FromContext(ctx).Info("successfully updated oauth-proxy client credentials", "name", org.Name)

	if err := r.Client.Update(ctx, intSecret); err != nil {
		log.FromContext(ctx).Error(err, "failed to update oauth2-proxy secret", "name", org.Name)
		return err
	}
	return nil
}

func ensureCallbackURL(url string) string {
	prefix := "https://"
	if !strings.HasPrefix(url, prefix) {
		url = prefix + url
	}
	suffix := "/callback"
	if !strings.HasSuffix(url, suffix) {
		url += suffix
	}
	return url
}

// getRedirects - returns the list of default redirect URIs for the reconciling OAuth2Client
// and merges with the provided redirect URIs
// this is needed when the default connector is being reconciled as it should not overwrite
// any appended redirect URIs from other organizations
func getRedirects(orgName string, redirectURIs []string) []string {
	defaultRedirects := []string{
		"http://localhost:8085", // allowing local development of idproxy url
		"https://dashboard." + common.DNSDomain,
		getRedirectForOrg(orgName),
	}
	return appendRedirects(defaultRedirects, redirectURIs...)
}

func getRedirectForOrg(orgName string) string {
	return fmt.Sprintf("https://%s.dashboard.%s", orgName, common.DNSDomain)
}

// appendRedirects - appends newRedirects to the redirects slice if it does not exist
func appendRedirects(redirects []string, newRedirects ...string) []string {
	for _, r := range newRedirects {
		if !slices.Contains(redirects, r) {
			redirects = append(redirects, r)
		}
	}
	return redirects
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
	return fmt.Sprintf("auth-proxy.%s.%s", orgName, common.DNSDomain)
}
