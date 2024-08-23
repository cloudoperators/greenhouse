// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package setup

import "github.com/cloudoperators/greenhouse/pkg/internal/local/utils"

type Cluster struct {
	Name       string  `json:"name"`
	Namespace  *string `json:"namespace"`
	skipCreate bool
}

type IClusterSetup interface {
	Setup() error
	Delete() error
}

func NewLocalCmdCluster(name, namespaceName string) IClusterSetup {
	return &Cluster{
		Name:      name,
		Namespace: utils.StringP(namespaceName),
	}
}

func Configure(config *Cluster, skipCreate bool) *Cluster {
	config.skipCreate = skipCreate
	return config
}

// Setup - creates a kind cluster with a given name and optionally creates a namespace if specified
func (c *Cluster) Setup() error {
	if c.skipCreate {
		return nil
	}
	err := createKindCluster(c.Name)
	if err != nil {
		return err
	}
	if c.Namespace == nil {
		return nil
	}
	return createNamespace(*c.Namespace)
}

// Delete - deletes a kind cluster with a given name
func (c *Cluster) Delete() error {
	return deleteKindCluster(c.Name)
}
