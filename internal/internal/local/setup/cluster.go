// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package setup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudoperators/greenhouse/internal/internal/local/klient"
	"github.com/cloudoperators/greenhouse/internal/internal/local/utils"
)

type Cluster struct {
	Name           string  `yaml:"name" json:"name"`
	Namespace      *string `yaml:"namespace" json:"namespace"`
	Version        string  `yaml:"version" json:"version"`
	ConfigPath     string  `yaml:"configPath" json:"configPath"`
	kubeConfigPath string
}

// clusterSetup - creates a kind Cluster with a given name and optionally creates a namespace if specified
// also accepts a KinD configuration file to create the cluster
func clusterSetup(env *ExecutionEnv) error {
	if env.cluster == nil {
		return errors.New("cluster configuration is missing")
	}
	err := klient.CreateCluster(env.cluster.Name, env.cluster.Version, env.cluster.ConfigPath)
	if err != nil {
		return err
	}
	err = env.cluster.saveConfig() // save kubeconfig after cluster creation
	if err != nil {
		return err
	}
	err = env.cluster.createNamespace() // create namespace if specified using kubeconfig
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

func (c *Cluster) createNamespace() error {
	if c.Namespace == nil {
		return nil
	}
	return klient.CreateNamespace(*c.Namespace, c.kubeConfigPath)
}
