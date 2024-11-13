// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package idproxy

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/connector/oidc"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousesapv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/rbac"
)

var (
	_ connector.CallbackConnector = (*oidcConnector)(nil)
	_ connector.RefreshConnector  = (*oidcConnector)(nil)
)

type OIDCConfig struct {
	oidc.Config          `json:",inline"`
	KeepUpstreamGroups   bool `json:"keepUpstreamGroups,omitempty"`
	client               client.Client
	redirectURIOverwrite string
}

const greenhouseLabelKeyPrefix = greenhouseapis.GroupName + "/"

func (c *OIDCConfig) AddClient(k8sClient client.Client) {
	c.client = k8sClient
}

func (c *OIDCConfig) AddRedirectURI(redirectURI string) {
	c.redirectURIOverwrite = redirectURI
}

func (c *OIDCConfig) Open(id string, logger *slog.Logger) (connector.Connector, error) {
	// overwrite redirectURI for (e.g. local) dex server talking to deployed connector running with differing config
	if c.redirectURIOverwrite != "" {
		c.RedirectURI = c.redirectURIOverwrite
	}
	conn, err := c.Config.Open(id, logger)
	if err != nil {
		return nil, err
	}

	return &oidcConnector{
		conn:               conn,
		logger:             logger,
		client:             c.client,
		id:                 id,
		keepUpstreamGroups: c.KeepUpstreamGroups,
	}, nil
}

type oidcConnector struct {
	conn               connector.Connector
	logger             *slog.Logger
	client             client.Client
	id                 string
	keepUpstreamGroups bool
}

func (c *oidcConnector) LoginURL(s connector.Scopes, callbackURL, state string) (string, error) {
	return c.conn.(connector.CallbackConnector).LoginURL(s, callbackURL, state) //nolint:errcheck
}

func (c *oidcConnector) HandleCallback(s connector.Scopes, r *http.Request) (connector.Identity, error) {
	identity, err := c.conn.(connector.CallbackConnector).HandleCallback(s, r) //nolint:errcheck
	if err != nil {
		return identity, err
	}

	groups, groupsErr := c.getGroups(c.id, identity.Groups, r.Context())
	if groupsErr != nil {
		c.logger.Info("failed getting group", "groupID", c.id, "error", groupsErr)

		if !c.keepUpstreamGroups {
			identity.Groups = []string{}
		}
	} else {
		identity.Groups = groups
	}

	c.logger.Info("created identity", slog.Group("user",
		"username", identity.Username,
		"preferredUsername", identity.PreferredUsername,
		"userID", identity.UserID,
		"email", identity.Email,
		"emailVerified", identity.EmailVerified,
		"groups", strings.Join(identity.Groups, ","),
	),
	)
	return identity, err
}

func (c *oidcConnector) Refresh(ctx context.Context, s connector.Scopes, identity connector.Identity) (connector.Identity, error) {
	identity, err := c.conn.(connector.RefreshConnector).Refresh(ctx, s, identity) //nolint:errcheck
	if err != nil {
		return identity, err
	}

	groups, groupsErr := c.getGroups(c.id, identity.Groups, ctx)
	if groupsErr != nil {
		c.logger.Info("failed getting groups", "connectorID", c.id, "error", groupsErr)
		identity.Groups = []string{}
	} else {
		identity.Groups = groups
	}

	c.logger.Info("refreshed identity", slog.Group("user",
		"username", identity.Username,
		"preferredUsername", identity.PreferredUsername,
		"userID", identity.UserID,
		"email", identity.Email,
		"emailVerified", identity.EmailVerified,
		"groups", strings.Join(identity.Groups, ","),
	))
	return identity, err
}

func (c *oidcConnector) getGroups(organization string, upstreamGroups []string, ctx context.Context) ([]string, error) {
	var groups []string
	groups = append(groups, rbac.OrganizationRoleName(c.id))

	teamNamesByIDPGroups := make(map[string][]string)
	roleNamesByIDPGroups := make(map[string]string)

	teamList := greenhousesapv1alpha1.TeamList{}

	// add team mappings
	err := c.client.List(ctx, &teamList, &client.ListOptions{Namespace: organization})
	if err != nil {
		return nil, err
	}
	for _, team := range teamList.Items {
		teamNamesByIDPGroups[team.Spec.MappedIDPGroup] = append(teamNamesByIDPGroups[team.Spec.MappedIDPGroup], "team:"+team.Name)
		for labelKey := range team.Labels {
			if strings.HasPrefix(labelKey, greenhouseLabelKeyPrefix) {
				teamCategoryName := strings.TrimPrefix(labelKey, greenhouseLabelKeyPrefix)
				teamNamesByIDPGroups[team.Spec.MappedIDPGroup] = append(teamNamesByIDPGroups[team.Spec.MappedIDPGroup], fmt.Sprintf("%s:%s", teamCategoryName, team.Name))
			}
		}
	}

	// add org admin role mapping
	org := new(greenhousesapv1alpha1.Organization)
	err = c.client.Get(ctx, types.NamespacedName{Namespace: "", Name: organization}, org)
	if err != nil {
		return nil, err
	}
	roleNamesByIDPGroups[org.Spec.MappedOrgAdminIDPGroup] = rbac.OrganizationAdminRoleName(organization)

	for _, group := range upstreamGroups {
		teamNameGroup, ok := teamNamesByIDPGroups[group]
		if ok {
			groups = append(groups, teamNameGroup...)
		}
		roleName, ok := roleNamesByIDPGroups[group]
		if ok {
			groups = append(groups, roleName)
		}
	}

	if c.keepUpstreamGroups {
		return append(upstreamGroups, groups...), nil
	}

	return groups, nil
}
