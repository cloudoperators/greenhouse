// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization

import (
	"context"
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

const (
	dexConnectorTypeGreenhouse = "greenhouse-oidc"
)

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
	defaultClient, err := r.dex.GetClient(ctx, defaultGreenhouseConnectorID)
	if err != nil {
		log.FromContext(ctx).Error(err, "failed to get default oauth2client", "name", defaultGreenhouseConnectorID)
		return err
	}
	err = r.dex.UpdateClient(ctx, defaultClient.Name, func(authClient storage.Client) (storage.Client, error) {
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
	if err := r.dex.DeleteConnector(ctx, org.Name); err != nil {
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
	if err := r.dex.DeleteClient(ctx, org.Name); err != nil {
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
	var userNameKey = "login_name"
	var skipEmailVerified = false
	if org.Spec.Authentication.OIDCConfig.ExtraConfig != nil {
		if org.Spec.Authentication.OIDCConfig.ExtraConfig.UserIDClaim == "" {
			userNameKey = org.Spec.Authentication.OIDCConfig.ExtraConfig.UserIDClaim
		}
		if org.Spec.Authentication.OIDCConfig.ExtraConfig != nil {
			skipEmailVerified = org.Spec.Authentication.OIDCConfig.ExtraConfig.InsecureSkipEmailVerified
		}
	}
	oidcConfig := &oidc.Config{
		Issuer:                    org.Spec.Authentication.OIDCConfig.Issuer,
		ClientID:                  clientID,
		ClientSecret:              clientSecret,
		RedirectURI:               redirectURL,
		UserNameKey:               userNameKey,
		UserIDKey:                 userNameKey,
		InsecureSkipEmailVerified: skipEmailVerified,
		InsecureSkipVerify:        true,
		InsecureEnableGroups:      true,
	}
	configByte, err := json.Marshal(oidcConfig)
	if err != nil {
		return err
	}
	oidcConnector, err := r.dex.GetConnector(ctx, org.Name)
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
	if err = r.dex.UpdateConnector(ctx, oidcConnector.ID, func(c storage.Connector) (storage.Connector, error) {
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
	oAuthClient, err := r.dex.GetClient(ctx, org.Name)
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
	if err = r.dex.UpdateClient(ctx, oAuthClient.ID, func(authClient storage.Client) (storage.Client, error) {
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
	defaultOAuthClient, err := r.dex.GetClient(ctx, defaultGreenhouseConnectorID)
	if err != nil {
		log.FromContext(ctx).Error(err, "failed to get default connector's oauth2client", "ID", defaultGreenhouseConnectorID)
		return err
	}
	orgRedirect := getRedirectForOrg(orgName)
	err = r.dex.UpdateClient(ctx, defaultOAuthClient.Name, func(authClient storage.Client) (storage.Client, error) {
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
