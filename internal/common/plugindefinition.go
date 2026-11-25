// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

var PromotionService string

type GenericPluginDefinition interface {
	lifecycle.RuntimeObject
	GetPluginDefinitionSpec() *greenhousev1alpha1.PluginDefinitionSpec
	FluxHelmChartResourceName() string
	GetLastSyncedArtifact() *greenhousev1alpha1.LastSyncedArtifact
	SetLastSyncedArtifact(artifact *greenhousev1alpha1.LastSyncedArtifact)
}

func getPluginDefinitionFromPromotionService(ctx context.Context, name, namespace string) (GenericPluginDefinition, error) {
	source := fmt.Sprintf("http://%s/apis/promotion.cloudoperators.dev/v1alpha1/namespaces/%s/revisions/%s/resolve", PromotionService, namespace, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, http.NoBody)
	if err != nil {
		return nil, err
	}
	c := &http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("failed to resolve PluginDefinition due to time out: %w", ctx.Err())
		}
		return nil, fmt.Errorf("failed to resolve PluginDefinition due to HTTP error: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get PluginDefinition from %s/%s - response status code: %d", namespace, name, resp.StatusCode)
	}

	pluginDefinition := greenhousev1alpha1.PluginDefinition{}
	if err := json.NewDecoder(resp.Body).Decode(&pluginDefinition); err != nil {
		return nil, fmt.Errorf("failed to decode PluginDefinition: %w", err)
	}

	return &pluginDefinition, nil
}

func GetPluginDefinitionFromPlugin(ctx context.Context, c client.Client, plugin *greenhousev1alpha1.Plugin) (GenericPluginDefinition, error) {
	if revision, ok := plugin.Annotations[api.AnnotationKeyPromotion]; ok {
		return getPluginDefinitionFromPromotionService(ctx, revision, plugin.Namespace)
	}

	pluginDefRef := plugin.Spec.PluginDefinitionRef
	switch plugin.Spec.PluginDefinitionRef.Kind {
	case greenhousev1alpha1.PluginDefinitionKind:
		pluginDefinition := &greenhousev1alpha1.PluginDefinition{}
		err := c.Get(ctx, client.ObjectKey{Namespace: plugin.Namespace, Name: pluginDefRef.Name}, pluginDefinition)
		switch {
		case apierrors.IsNotFound(err):
			return nil, fmt.Errorf("PluginDefinition %s/%s does not exist", plugin.Namespace, pluginDefRef.Name)
		case err != nil:
			return nil, fmt.Errorf("failed to get PluginDefinition %s/%s: %s", plugin.Namespace, pluginDefRef.Name, err.Error())
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
			return nil, fmt.Errorf("ClusterPluginDefinition %s/%s does not exist", plugin.Namespace, pluginDefRef.Name)
		case err != nil:
			return nil, fmt.Errorf("failed to get ClusterPluginDefinition %s/%s: %s", plugin.Namespace, pluginDefRef.Name, err.Error())
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
