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

func (c *Cluster) Setup() error {
	if c.skipCreate {
		return nil
	}
	err := CreateKindCluster(c.Name)
	if err != nil {
		return err
	}
	if c.Namespace == nil {
		return nil
	}
	return CreateNamespace(*c.Namespace)
}

func (c *Cluster) Delete() error {
	return DeleteCluster(c.Name)
}
