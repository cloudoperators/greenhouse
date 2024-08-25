// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package setup

import (
	"errors"
	"fmt"
	"github.com/cloudoperators/greenhouse/pkg/internal/local/klient"
	"github.com/cloudoperators/greenhouse/pkg/internal/local/utils"
	"os"
	"path/filepath"
)

type Cluster struct {
	Name           string  `json:"name"`
	Namespace      *string `json:"namespace"`
	kubeConfigPath string
}

// clusterSetup - creates a kind Cluster with a given name and optionally creates a namespace if specified
func clusterSetup(env *ExecutionEnv) error {
	if env.cluster == nil {
		return errors.New("cluster configuration is missing")
	}
	err := klient.CreateCluster(env.cluster.Name)
	if err != nil {
		return err
	}
	if env.cluster.Namespace != nil {
		err = klient.CreateNamespace(*env.cluster.Namespace)
		if err != nil {
			return err
		}
	}
	err = env.cluster.saveConfig()
	if err != nil {
		return err
	}
	env.info = append(env.info, fmt.Sprintf("cluster %s - kubeconfig: %s", env.cluster.Name, env.cluster.kubeConfigPath))
	return nil
}

// clusterDelete - deletes a kind Cluster with a given name
func clusterDelete(env *ExecutionEnv) error {
	if env.cluster == nil {
		return errors.New("cluster configuration is missing")
	}
	return klient.DeleteCluster(env.cluster.Name)
}

func (c *Cluster) saveConfig() error {
	kubeConfig, err := klient.GetKubeCfg(c.Name, false)
	if err != nil {
		return err
	}
	dir := filepath.Join(os.TempDir(), "greenhouse")
	file := c.Name + ".kubeconfig"
	err = utils.WriteToPath(dir, file, kubeConfig)
	if err != nil {
		return err
	}
	c.kubeConfigPath = filepath.Join(dir, file)
	return nil
}
