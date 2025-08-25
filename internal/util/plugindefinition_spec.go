// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"context"
	"errors"
	"fmt"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetPluginDefinitionSpec resolves the (Cluster-)PluginDefinition reference and returns shared PluginDefinitionSpec
func GetPluginDefinitionSpec(ctx context.Context, c client.Client, pluginDefinitionName, pluginDefinitionKind, resourceNamespace string) (*greenhousev1alpha1.PluginDefinitionSpec, error) {
	var pluginDefinitionSpec greenhousev1alpha1.PluginDefinitionSpec

	switch pluginDefinitionKind {
	case greenhousev1alpha1.PluginDefinitionKind:
		pluginDefinition := &greenhousev1alpha1.PluginDefinition{}
		err := c.Get(ctx, client.ObjectKey{Namespace: resourceNamespace, Name: pluginDefinitionName}, pluginDefinition)
		switch {
		case apierrors.IsNotFound(err):
			return nil, fmt.Errorf("PluginDefinition %s does not exist in namespace %s", pluginDefinitionName, resourceNamespace)
		case err != nil:
			return nil, fmt.Errorf("failed to get PluginDefinition %s from namespace %s: %s", pluginDefinitionName, resourceNamespace, err.Error())
		default:
			pluginDefinitionSpec = pluginDefinition.Spec
		}
	case greenhousev1alpha1.ClusterPluginDefinitionKind:
		clusterPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{}
		err := c.Get(ctx, client.ObjectKey{Name: pluginDefinitionName}, clusterPluginDefinition)
		switch {
		case apierrors.IsNotFound(err):
			return nil, fmt.Errorf("ClusterPluginDefinition %s does not exist", pluginDefinitionName)
		case err != nil:
			return nil, fmt.Errorf("failed to get ClusterPluginDefinition %s: %s", pluginDefinitionName, err.Error())
		default:
			pluginDefinitionSpec = clusterPluginDefinition.Spec
		}
	case "": // existing Plugins/Presets created without PluginDefinitionKind
		clusterPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{}
		err := c.Get(ctx, client.ObjectKey{Name: pluginDefinitionName}, clusterPluginDefinition)
		switch {
		case apierrors.IsNotFound(err):
			return nil, fmt.Errorf("ClusterPluginDefinition %s does not exist", pluginDefinitionName)
		case err != nil:
			return nil, fmt.Errorf("failed to get ClusterPluginDefinition %s: %s", pluginDefinitionName, err.Error())
		default:
			pluginDefinitionSpec = clusterPluginDefinition.Spec
		}
	default:
		return nil, errors.New("unsupported PluginDefinitionKind: " + pluginDefinitionKind)
	}

	return &pluginDefinitionSpec, nil
}
