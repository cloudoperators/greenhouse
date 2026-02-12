// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

type GenericPluginDefinition interface {
	lifecycle.RuntimeObject
	GetPluginDefinitionSpec() *greenhousev1alpha1.PluginDefinitionSpec
	FluxHelmChartResourceName() string
}

func GetPluginDefinitionFromPlugin(ctx context.Context, c client.Client, pluginDefRef greenhousev1alpha1.PluginDefinitionReference, resourceNamespace string) (GenericPluginDefinition, error) {
	switch pluginDefRef.Kind {
	case greenhousev1alpha1.PluginDefinitionKind:
		pluginDefinition := &greenhousev1alpha1.PluginDefinition{}
		err := c.Get(ctx, client.ObjectKey{Namespace: resourceNamespace, Name: pluginDefRef.Name}, pluginDefinition)
		switch {
		case apierrors.IsNotFound(err):
			return nil, fmt.Errorf("PluginDefinition %s/%s does not exist", resourceNamespace, pluginDefRef.Name)
		case err != nil:
			return nil, fmt.Errorf("failed to get PluginDefinition %s/%s: %s", resourceNamespace, pluginDefRef.Name, err.Error())
		default:
			err = isPluginDefinitionReady(pluginDefinition)
			if err != nil {
				return nil, err
			}
			return pluginDefinition, nil
		}
	case greenhousev1alpha1.ClusterPluginDefinitionKind:
		clusterPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{}
		err := c.Get(ctx, client.ObjectKey{Name: pluginDefRef.Name}, clusterPluginDefinition)
		switch {
		case apierrors.IsNotFound(err):
			return nil, fmt.Errorf("ClusterPluginDefinition %s/%s does not exist", resourceNamespace, pluginDefRef.Name)
		case err != nil:
			return nil, fmt.Errorf("failed to get ClusterPluginDefinition %s/%s: %s", resourceNamespace, pluginDefRef.Name, err.Error())
		default:
			err = isPluginDefinitionReady(clusterPluginDefinition)
			if err != nil {
				return nil, err
			}
			return clusterPluginDefinition, nil
		}
	case "":
		return nil, errors.New("PluginDefinitionRef.Kind has not been set")
	default:
		return nil, errors.New("unsupported PluginDefinition kind: " + pluginDefRef.Kind)
	}
}

func isPluginDefinitionReady(pluginDef GenericPluginDefinition) error {
	conditions := pluginDef.GetConditions()
	readyCond := conditions.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
	if readyCond == nil {
		return fmt.Errorf("PluginDefinition %s/%s is not Ready", pluginDef.GetNamespace(), pluginDef.GetName())
	}
	if readyCond.Status != metav1.ConditionTrue {
		return fmt.Errorf("PluginDefinition %s/%s is not Ready: %s", pluginDef.GetNamespace(), pluginDef.GetName(), readyCond.Message)
	}
	return nil
}

// GetPluginDefinitionSpec resolves the (Cluster-)PluginDefinition reference and returns shared PluginDefinitionSpec
func GetPluginDefinitionSpec(ctx context.Context, c client.Client, pluginDefinitionRef greenhousev1alpha1.PluginDefinitionReference, resourceNamespace string) (*greenhousev1alpha1.PluginDefinitionSpec, error) {
	var pluginDefinitionSpec greenhousev1alpha1.PluginDefinitionSpec

	switch pluginDefinitionRef.Kind {
	case greenhousev1alpha1.PluginDefinitionKind:
		pluginDefinition := &greenhousev1alpha1.PluginDefinition{}
		err := c.Get(ctx, client.ObjectKey{Namespace: resourceNamespace, Name: pluginDefinitionRef.Name}, pluginDefinition)
		switch {
		case apierrors.IsNotFound(err):
			return nil, fmt.Errorf("PluginDefinition %s does not exist in namespace %s", pluginDefinitionRef.Name, resourceNamespace)
		case err != nil:
			return nil, fmt.Errorf("failed to get PluginDefinition %s from namespace %s: %s", pluginDefinitionRef.Name, resourceNamespace, err.Error())
		default:
			pluginDefinitionSpec = pluginDefinition.Spec
		}
	case greenhousev1alpha1.ClusterPluginDefinitionKind:
		clusterPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{}
		err := c.Get(ctx, client.ObjectKey{Name: pluginDefinitionRef.Name}, clusterPluginDefinition)
		switch {
		case apierrors.IsNotFound(err):
			return nil, fmt.Errorf("ClusterPluginDefinition %s does not exist", pluginDefinitionRef.Name)
		case err != nil:
			return nil, fmt.Errorf("failed to get ClusterPluginDefinition %s: %s", pluginDefinitionRef.Name, err.Error())
		default:
			pluginDefinitionSpec = clusterPluginDefinition.Spec
		}
	case "":
		return nil, errors.New("PluginDefinitionRef.Kind has not been set")
	default:
		return nil, errors.New("unsupported PluginDefinition kind: " + pluginDefinitionRef.Kind)
	}

	return &pluginDefinitionSpec, nil
}
