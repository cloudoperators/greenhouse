// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"strings"

	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ManagerConfig is the operational configuration for the greenhouse controller manager,
// loaded from the greenhouse-manager-cm ConfigMap.
type ManagerConfig struct {
	Controller map[string]ConcurrencyConfig
}

// ConcurrencyConfig holds concurrency settings for a single controller.
type ConcurrencyConfig struct {
	MaxConcurrentReconciles int
}

// NewManagerConfig loads ManagerConfig from the named ConfigMap.
// Returns a zero-value config (all defaults apply) if the ConfigMap does not exist.
func NewManagerConfig(ctx context.Context, cl client.Reader, name, namespace string) (*ManagerConfig, error) {
	cm := &corev1.ConfigMap{}
	cm.SetName(name)
	cm.SetNamespace(namespace)
	if err := cl.Get(ctx, client.ObjectKeyFromObject(cm), cm); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	if cm.Data == nil {
		return nil, nil
	}
	v := viper.New()
	for k, val := range cm.Data {
		v.Set(strings.ToLower(k), val)
	}
	cfg := &ManagerConfig{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// GetMaxConcurrentReconciles returns the configured concurrency for a controller,
// falling back to defaultVal when not set or zero.
func (c *ManagerConfig) GetMaxConcurrentReconciles(controllerName string, defaultVal int) int {
	if c == nil {
		return defaultVal
	}
	if cfg, ok := c.Controller[strings.ToLower(controllerName)]; ok && cfg.MaxConcurrentReconciles > 0 {
		return cfg.MaxConcurrentReconciles
	}
	return defaultVal
}
