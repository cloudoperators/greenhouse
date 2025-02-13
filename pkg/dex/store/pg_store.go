// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/sql"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/common"
	"github.com/cloudoperators/greenhouse/pkg/util"
)

type pgDex struct {
	storage storage.Storage
	backend string
}

const (
	hostEnv   = "PG_HOST"
	portEnv   = "PG_PORT"
	userEnv   = "PG_USER"
	passEnv   = "PG_PASSWORD"
	dbNameEnv = "PG_DATABASE"
)

// newPostgresStore - creates a new postgres storage backend for dex
func newPostgresStore(logger *slog.Logger) (storage.Storage, error) {
	var host, user, pass, database string
	var port int
	var err error
	database = clientutil.GetEnvOrDefault(dbNameEnv, "postgres")
	user = clientutil.GetEnvOrDefault(userEnv, "postgres")
	port = clientutil.GetIntEnvWithDefault(portEnv, 5432)
	if host, err = clientutil.GetEnv(hostEnv); err != nil {
		return nil, err
	}
	if pass, err = clientutil.GetEnv(passEnv); err != nil {
		return nil, err
	}
	cfg := &sql.Postgres{
		SSL: sql.SSL{Mode: "disable"},
		NetworkDB: sql.NetworkDB{
			Host:     host,
			Port:     uint16(port),
			User:     user,
			Password: pass,
			Database: database,
		},
	}
	return cfg.Open(logger)
}

func (p *pgDex) GetBackend() string {
	return p.backend
}

// CreateUpdateConnector - creates or updates a dex connector in dex postgres storage backend
func (p *pgDex) CreateUpdateConnector(ctx context.Context, _ client.Client, org *greenhouseapisv1alpha1.Organization, configByte []byte) error {
	oidcConnector, err := p.storage.GetConnector(org.Name)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			err = p.storage.CreateConnector(ctx, storage.Connector{
				ID:     org.Name,
				Type:   dexConnectorTypeGreenhouse,
				Name:   cases.Title(language.English).String(org.Name),
				Config: configByte,
			})
			if err != nil {
				return err
			}
			log.FromContext(ctx).Info("created dex connector in SQL storage", "name", org.Name)
			return nil
		}
		log.FromContext(ctx).Error(err, "failed to get dex connector in SQL storage", "name", org.Name)
		return err
	}
	if err = p.storage.UpdateConnector(oidcConnector.ID, func(c storage.Connector) (storage.Connector, error) {
		c.ID = org.Name
		c.Type = dexConnectorTypeGreenhouse
		c.Name = cases.Title(language.English).String(org.Name)
		c.Config = configByte
		return c, nil
	}); err != nil {
		log.FromContext(ctx).Error(err, "failed to update dex connector in SQL storage", "name", org.Name)
		return err
	}
	log.FromContext(ctx).Info("updated dex connector in SQL storage", "name", org.Name)
	return nil
}

// CreateUpdateOauth2Client - creates or updates an oauth2 client in dex postgres storage backend
func (p *pgDex) CreateUpdateOauth2Client(ctx context.Context, _ client.Client, org *greenhouseapisv1alpha1.Organization) error {
	oAuthClient, err := p.storage.GetClient(org.Name)
	if err != nil && errors.Is(err, storage.ErrNotFound) {
		if err = p.storage.CreateClient(ctx, storage.Client{
			Public: true,
			ID:     org.Name,
			Name:   org.Name,
			RedirectURIs: []string{
				"http://localhost:8085",
				"https://dashboard." + common.DNSDomain,
				fmt.Sprintf("https://%s.dashboard.%s", org.Name, common.DNSDomain),
			},
		}); err != nil {
			return err
		}
		log.FromContext(ctx).Info("created oauth2client", "name", org.Name)
		return nil
	}
	if err != nil {
		log.FromContext(ctx).Error(err, "failed to get oauth2client", "name", org.Name)
	}

	err = p.storage.UpdateClient(oAuthClient.Name, func(authClient storage.Client) (storage.Client, error) {
		authClient.Public = true
		authClient.ID = org.Name
		authClient.Name = org.Name
		for _, requiredRedirectURL := range []string{
			"http://localhost:8085",
			"https://dashboard." + common.DNSDomain,
			fmt.Sprintf("https://%s.dashboard.%s", org.Name, common.DNSDomain),
		} {
			authClient.RedirectURIs = util.AppendStringToSliceIfNotContains(requiredRedirectURL, authClient.RedirectURIs)
		}
		return authClient, nil
	})
	if err != nil {
		return err
	}
	return nil
}

// GetStorage - returns the underlying dex storage interface
func (p *pgDex) GetStorage() storage.Storage {
	return p.storage
}

func (p *pgDex) Close() error {
	return p.storage.Close()
}
