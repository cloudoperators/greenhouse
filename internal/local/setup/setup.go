// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package setup

import (
	"context"

	"github.com/cloudoperators/greenhouse/internal/local/utils"
)

type ExecutionEnv struct {
	cluster *Cluster
	steps   []Step
	info    []string // info messages to be displayed at the end of run
	devMode bool
}

type Step func(builder *ExecutionEnv) error

func NewExecutionEnv(devMode bool) *ExecutionEnv {
	return &ExecutionEnv{
		steps:   make([]Step, 0),
		devMode: devMode,
	}
}

func (env *ExecutionEnv) WithClusterSetup(cluster *Cluster) *ExecutionEnv {
	env.cluster = cluster
	env.steps = append(env.steps, clusterSetup)
	return env
}

func (env *ExecutionEnv) WithClusterDelete(name string) *ExecutionEnv {
	env.cluster = &Cluster{
		Name: name,
	}
	env.steps = append(env.steps, clusterDelete)
	return env
}

func (env *ExecutionEnv) WithLocalPluginDev(manifest *Manifest) *ExecutionEnv {
	manifest.enableLocalPluginDev = true
	return env
}

func (env *ExecutionEnv) WithLimitedManifests(ctx context.Context, manifest *Manifest) *ExecutionEnv {
	env.steps = append(env.steps, limitedManifestSetup(ctx, manifest))
	return env
}

func (env *ExecutionEnv) WithGreenhouseDevelopment(ctx context.Context, manifest *Manifest) *ExecutionEnv {
	env.steps = append(env.steps, greenhouseManifestSetup(ctx, manifest))
	return env
}

func (env *ExecutionEnv) WithDashboardSetup(ctx context.Context, manifest *Manifest) *ExecutionEnv {
	env.steps = append(env.steps, dashboardSetup(ctx, manifest))
	return env
}

func (env *ExecutionEnv) Run() error {
	for _, step := range env.steps {
		err := step(env)
		if err != nil {
			return err
		}
	}
	for _, i := range env.info {
		utils.Log(i)
	}
	return nil
}
