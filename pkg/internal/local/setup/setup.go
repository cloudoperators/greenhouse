// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package setup

import (
	"context"
	"strings"

	"github.com/cloudoperators/greenhouse/pkg/internal/local/utils"
)

type ExecutionEnv struct {
	cluster *Cluster
	steps   []Step
	info    []string // info messages to be displayed at the end of run
}

type Step func(builder *ExecutionEnv) error

func NewExecutionEnv() *ExecutionEnv {
	return &ExecutionEnv{
		steps: make([]Step, 0),
	}
}

func (env *ExecutionEnv) WithClusterSetup(name, namespace, version string) *ExecutionEnv {
	env.cluster = &Cluster{
		Name:      name,
		Namespace: nil,
		Version:   version,
	}
	if strings.TrimSpace(namespace) != "" {
		env.cluster.Namespace = &namespace
	}
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

func (env *ExecutionEnv) WithAllManifests(ctx context.Context, manifest *Manifest) *ExecutionEnv {
	env.steps = append(env.steps, allManifestSetup(ctx, manifest))
	return env
}

func (env *ExecutionEnv) WithLimitedManifests(ctx context.Context, manifest *Manifest) *ExecutionEnv {
	env.steps = append(env.steps, limitedManifestSetup(ctx, manifest))
	return env
}

func (env *ExecutionEnv) WithWebhookDevelopment(ctx context.Context, manifest *Manifest) *ExecutionEnv {
	env.steps = append(env.steps, webhookManifestSetup(ctx, manifest))
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
