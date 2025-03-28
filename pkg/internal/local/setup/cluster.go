// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package setup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudoperators/greenhouse/pkg/internal/local/klient"
	"github.com/cloudoperators/greenhouse/pkg/internal/local/utils"
)

type PostSetupAction struct {
	Command string            `yaml:"command" json:"command"`
	Vars    map[string]string `yaml:"vars" json:"vars"`
}

type Cluster struct {
	Name           string            `yaml:"name" json:"name"`
	Namespace      *string           `yaml:"namespace" json:"namespace"`
	Version        string            `yaml:"version" json:"version"`
	ConfigPath     string            `yaml:"configPath" json:"configPath"`
	PostSetup      []PostSetupAction `yaml:"postSetup" json:"postSetup"`
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
	return env.cluster.executePostSetup()
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

// executePostSetup - executes post setup actions on the cluster as defined in the configuration file
// piped shell commands are converted to ShellPipe type
// NOTE: output redirects to file are not supported
func (c *Cluster) executePostSetup() error {
	if len(c.PostSetup) == 0 {
		return nil
	}
	var err error
	for _, action := range c.PostSetup {
		action.Vars["kubeconfig"] = c.kubeConfigPath
		shells := make([]utils.Shell, 0)
		commands := strings.Split(action.Command, "|")
		if len(commands) > 1 {
			for _, cmd := range commands {
				shells = append(shells, utils.Shell{
					Cmd:  strings.TrimSpace(cmd),
					Vars: action.Vars,
				})
			}
			err = utils.ShellPipe{Shells: shells}.Exec()
		} else {
			err = utils.Shell{
				Cmd:  action.Command,
				Vars: action.Vars,
			}.Exec()
		}
		if err != nil {
			utils.Logf("post setup acction error: %s", err.Error())
			break
		}
	}
	return err
}
