// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package dex

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/kubernetes"
	"github.com/dexidp/dex/storage/sql"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/cloudoperators/greenhouse/pkg/clientutil"
)

const (
	Postgres = "postgres"
	K8s      = "kubernetes"
)

const (
	hostEnv   = "PG_HOST"
	portEnv   = "PG_PORT"
	userEnv   = "PG_USER"
	passEnv   = "PG_PASSWORD"
	dbNameEnv = "PG_DATABASE"
)

// newKubernetesStore - creates a new kubernetes storage backend for dex
func newKubernetesStore(logger *slog.Logger) (storage.Storage, error) {
	cfg := kubernetes.Config{InCluster: true}
	cfgPath := determineKubeMode()
	if strings.TrimSpace(cfgPath) != "" {
		cfg.InCluster = false
		cfg.KubeConfigFile = cfgPath
	}
	dexStorage, err := cfg.Open(logger)
	if err != nil {
		return nil, err
	}
	return dexStorage, nil
}

func determineKubeMode() string {
	cfgPath := os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
	if strings.TrimSpace(cfgPath) != "" {
		return cfgPath
	}
	_, err := rest.InClusterConfig()
	if err == nil {
		return ""
	}
	return filepath.Join(homedir.HomeDir(), ".kube", "config")
}

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
			Port:     uint16(port), //nolint:gosec
			User:     user,
			Password: pass,
			Database: database,
		},
	}
	return cfg.Open(logger)
}

// NewDexStorage - create a new dex storage adapter depending on the backend
func NewDexStorage(logger *slog.Logger, backend string) (storage.Storage, error) {
	switch backend {
	case Postgres:
		return newPostgresStore(logger)
	case K8s:
		return newKubernetesStore(logger)
	default:
		return nil, fmt.Errorf("unknown dexStorage backend: %s", backend)
	}
}
