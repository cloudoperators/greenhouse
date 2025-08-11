// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"
	"encoding/json"
	"sort"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/common"
)

func GetPluginOptionValuesForPlugin(ctx context.Context, c client.Client, plugin *greenhousev1alpha1.Plugin) ([]greenhousemetav1alpha1.PluginOptionValue, error) {
	var pluginDefinition = new(greenhousev1alpha1.ClusterPluginDefinition)
	if err := c.Get(ctx, types.NamespacedName{Namespace: "", Name: plugin.Spec.PluginDefinition}, pluginDefinition); err != nil {
		return nil, err
	}
	values := MergePluginAndPluginOptionValueSlice(pluginDefinition.Spec.Options, plugin.Spec.OptionValues)
	// Enrich with default greenhouse values.
	greenhouseValues, err := GetGreenhouseValues(ctx, c, *plugin)
	if err != nil {
		return nil, err
	}
	values = MergePluginOptionValues(values, greenhouseValues)
	return values, nil
}

// GetGreenhouseValues generate values for greenhouse core resources in the form:
//
//	global:
//	  greenhouse:
//	    clusterNames:
//		  - <name>
//		teams:
//		  - <name>
func GetGreenhouseValues(ctx context.Context, c client.Client, p greenhousev1alpha1.Plugin) ([]greenhousemetav1alpha1.PluginOptionValue, error) {
	greenhouseValues := make([]greenhousemetav1alpha1.PluginOptionValue, 0)
	var clusterList = new(greenhousev1alpha1.ClusterList)
	if err := c.List(ctx, clusterList, &client.ListOptions{Namespace: p.GetNamespace()}); err != nil {
		return nil, err
	}
	clusterNames := make([]string, len(clusterList.Items))
	for idx, cluster := range clusterList.Items {
		clusterNames[idx] = cluster.Name
	}

	clusterNamesVal, err := stringSliceToHelmValue(clusterNames)
	if err != nil {
		return nil, err
	}

	greenhouseValues = append(greenhouseValues, greenhousemetav1alpha1.PluginOptionValue{
		Name:      "global.greenhouse.clusterNames",
		Value:     clusterNamesVal,
		ValueFrom: nil,
	})

	// Teams within the organization.
	var teamList = new(greenhousev1alpha1.TeamList)
	if err := c.List(ctx, teamList, client.InNamespace(p.GetNamespace())); err != nil {
		return nil, err
	}
	teamNames := make([]string, len(teamList.Items))
	for idx, team := range teamList.Items {
		teamNames[idx] = team.Name
	}

	teamNamesVal, err := stringSliceToHelmValue(teamNames)
	if err != nil {
		return nil, err
	}

	greenhouseValues = append(greenhouseValues, greenhousemetav1alpha1.PluginOptionValue{
		Name:      "global.greenhouse.teamNames",
		Value:     teamNamesVal,
		ValueFrom: nil,
	})

	// append orgName
	orgNameVal, err := json.Marshal(p.GetNamespace())
	if err != nil {
		return nil, err
	}

	greenhouseValues = append(greenhouseValues, greenhousemetav1alpha1.PluginOptionValue{
		Name:      "global.greenhouse.organizationName",
		Value:     &apiextensionsv1.JSON{Raw: orgNameVal},
		ValueFrom: nil,
	})

	// append clusterName if set
	if p.Spec.ClusterName != "" {
		clusterNameVal, err := json.Marshal(p.Spec.ClusterName)
		if err != nil {
			return nil, err
		}

		greenhouseValues = append(greenhouseValues, greenhousemetav1alpha1.PluginOptionValue{
			Name:      "global.greenhouse.clusterName",
			Value:     &apiextensionsv1.JSON{Raw: clusterNameVal},
			ValueFrom: nil,
		})
	}

	// append DNSDomain
	baseDomainVal, err := json.Marshal(common.DNSDomain)
	if err != nil {
		return nil, err
	}
	greenhouseValues = append(greenhouseValues, greenhousemetav1alpha1.PluginOptionValue{
		Name:      "global.greenhouse.baseDomain",
		Value:     &apiextensionsv1.JSON{Raw: baseDomainVal},
		ValueFrom: nil,
	})

	// append owning team if set
	if p.Labels[string(greenhouseapis.LabelKeyOwnedBy)] != "" {
		owningTeamVal, err := json.Marshal(p.Labels[greenhouseapis.LabelKeyOwnedBy])
		if err != nil {
			return nil, err
		}
		greenhouseValues = append(greenhouseValues, greenhousemetav1alpha1.PluginOptionValue{
			Name:      "global.greenhouse.ownedBy",
			Value:     &apiextensionsv1.JSON{Raw: owningTeamVal},
			ValueFrom: nil,
		})
	}
	return greenhouseValues, nil
}

func setOrAppendNameValue(valueSlice []greenhousemetav1alpha1.PluginOptionValue, valueToSetOrAppend greenhousemetav1alpha1.PluginOptionValue) []greenhousemetav1alpha1.PluginOptionValue {
	for idx, val := range valueSlice {
		if val.Name == valueToSetOrAppend.Name {
			valueSlice[idx].Value = valueToSetOrAppend.Value
			return valueSlice
		}
	}
	return append(valueSlice, valueToSetOrAppend)
}

// stringSliceToHelmValue sorts theSlice, marshals it to JSON and returns an apiextensionsv1.JSON object.
func stringSliceToHelmValue(theSlice []string) (*apiextensionsv1.JSON, error) {
	sort.Strings(theSlice)

	raw, err := json.Marshal(theSlice)
	if err != nil {
		return nil, err
	}
	return &apiextensionsv1.JSON{Raw: raw}, nil
}
