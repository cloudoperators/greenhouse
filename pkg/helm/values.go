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

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

func GetPluginOptionValuesForPlugin(ctx context.Context, c client.Client, plugin *greenhousev1alpha1.Plugin) ([]greenhousev1alpha1.PluginOptionValue, error) {
	var pluginDefinition = new(greenhousev1alpha1.PluginDefinition)
	if err := c.Get(ctx, types.NamespacedName{Namespace: "", Name: plugin.Spec.PluginDefinition}, pluginDefinition); err != nil {
		return nil, err
	}
	values := mergePluginAndPluginOptionValueSlice(pluginDefinition.Spec.Options, plugin.Spec.OptionValues)
	// Enrich with default greenhouse values.
	greenhouseValues, err := getGreenhouseValues(ctx, c, *plugin)
	if err != nil {
		return nil, err
	}
	values = mergePluginOptionValues(values, greenhouseValues)
	return values, nil
}

func mergePluginAndPluginOptionValueSlice(pluginOptions []greenhousev1alpha1.PluginOption, pluginOptionValues []greenhousev1alpha1.PluginOptionValue) []greenhousev1alpha1.PluginOptionValue {
	// Make sure there's always a non-nil slice.
	out := make([]greenhousev1alpha1.PluginOptionValue, 0)
	defer func() {
		sort.Slice(out, func(i, j int) bool {
			return out[i].Name < out[j].Name
		})
	}()
	// If the PluginDefinition doesn't define values, we're done.
	if pluginOptions == nil {
		return pluginOptionValues
	}
	for _, option := range pluginOptions {
		if option.Default != nil {
			out = append(out, greenhousev1alpha1.PluginOptionValue{Name: option.Name, Value: option.Default})
		}
	}
	for _, pluginVal := range pluginOptionValues {
		out = setOrAppendNameValue(out, pluginVal)
	}
	return out
}

// mergePluginOptionValues merges the given src into the dst PluginOptionValue slice.
func mergePluginOptionValues(dst, src []greenhousev1alpha1.PluginOptionValue) []greenhousev1alpha1.PluginOptionValue {
	if dst == nil {
		dst = make([]greenhousev1alpha1.PluginOptionValue, 0)
	}
	for _, srcOptionValue := range src {
		dst = setOrAppendNameValue(dst, srcOptionValue)
	}
	return dst
}

// getGreenhouseValues generate values for greenhouse core resources in the form:
//
//	global:
//	  greenhouse:
//	    clusterNames:
//		  - <name>
//		teams:
//		  - <name>
func getGreenhouseValues(ctx context.Context, c client.Client, p greenhousev1alpha1.Plugin) ([]greenhousev1alpha1.PluginOptionValue, error) {
	greenhouseValues := make([]greenhousev1alpha1.PluginOptionValue, 0)
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

	greenhouseValues = append(greenhouseValues, greenhousev1alpha1.PluginOptionValue{
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

	greenhouseValues = append(greenhouseValues, greenhousev1alpha1.PluginOptionValue{
		Name:      "global.greenhouse.teamNames",
		Value:     teamNamesVal,
		ValueFrom: nil,
	})

	// append orgName
	orgNameVal, err := json.Marshal(p.GetNamespace())
	if err != nil {
		return nil, err
	}

	greenhouseValues = append(greenhouseValues, greenhousev1alpha1.PluginOptionValue{
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

		greenhouseValues = append(greenhouseValues, greenhousev1alpha1.PluginOptionValue{
			Name:      "global.greenhouse.clusterName",
			Value:     &apiextensionsv1.JSON{Raw: clusterNameVal},
			ValueFrom: nil,
		})
	}
	return greenhouseValues, nil
}

func setOrAppendNameValue(valueSlice []greenhousev1alpha1.PluginOptionValue, valueToSetOrAppend greenhousev1alpha1.PluginOptionValue) []greenhousev1alpha1.PluginOptionValue {
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
