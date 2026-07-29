// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/common"
)

var retiredGreenhouseValues = []string{
	"global.greenhouse.clusterNames",
	"global.greenhouse.teamNames",
}

func GetPluginOptionValuesForPlugin(ctx context.Context, c client.Client, plugin *greenhousev1alpha1.Plugin) ([]greenhousev1alpha1.PluginOptionValue, error) {
	pluginDefinitionSpec, err := common.GetPluginDefinitionSpec(ctx, c, plugin.Spec.PluginDefinitionRef, plugin.GetNamespace())
	if err != nil {
		return nil, err
	}

	// Clone, as the merge writes through to its input when the PluginDefinition declares no options.
	values := MergePluginAndPluginOptionValueSlice(pluginDefinitionSpec.Options, slices.Clone(plugin.Spec.OptionValues))
	// Enrich with default greenhouse values.
	greenhouseValues, err := GetGreenhouseValues(ctx, c, *plugin)
	if err != nil {
		return nil, err
	}
	values = MergePluginOptionValues(values, greenhouseValues)
	return StripRetiredGreenhouseValues(values), nil
}

// StripRetiredGreenhouseValues drops the values Greenhouse no longer injects.
// It clones, as the input may alias a Plugin's OptionValues.
func StripRetiredGreenhouseValues(values []greenhousev1alpha1.PluginOptionValue) []greenhousev1alpha1.PluginOptionValue {
	return slices.DeleteFunc(slices.Clone(values), func(v greenhousev1alpha1.PluginOptionValue) bool {
		return slices.Contains(retiredGreenhouseValues, v.Name)
	})
}

// GetGreenhouseValues generate values for greenhouse core resources in the form:
//
//	global:
//	  greenhouse:
//	    organizationName: <name>
//	    clusterName: <name>
//	    baseDomain: <domain>
//	    ownedBy: <team>
//	    metadata:
//	      <key>: <value>
func GetGreenhouseValues(ctx context.Context, c client.Client, p greenhousev1alpha1.Plugin) ([]greenhousev1alpha1.PluginOptionValue, error) {
	greenhouseValues := make([]greenhousev1alpha1.PluginOptionValue, 0)

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

		// Append cluster metadata labels as global.greenhouse.metadata values.
		var cluster = new(greenhousev1alpha1.Cluster)
		if err := c.Get(ctx, types.NamespacedName{Namespace: p.GetNamespace(), Name: p.Spec.ClusterName}, cluster); err == nil {
			for key, value := range cluster.Labels {
				// Convert metadata.greenhouse.sap/* to global.greenhouse.metadata.*
				if metadataKey := extractMetadataKey(key); metadataKey != "" {
					metadataVal, err := json.Marshal(value)
					if err != nil {
						continue
					}

					greenhouseValues = append(greenhouseValues, greenhousev1alpha1.PluginOptionValue{
						Name:      "global.greenhouse.metadata." + metadataKey,
						Value:     &apiextensionsv1.JSON{Raw: metadataVal},
						ValueFrom: nil,
					})
				}
			}
		}
	}

	// append DNSDomain
	baseDomainVal, err := json.Marshal(common.DNSDomain)
	if err != nil {
		return nil, err
	}
	greenhouseValues = append(greenhouseValues, greenhousev1alpha1.PluginOptionValue{
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
		greenhouseValues = append(greenhouseValues, greenhousev1alpha1.PluginOptionValue{
			Name:      "global.greenhouse.ownedBy",
			Value:     &apiextensionsv1.JSON{Raw: owningTeamVal},
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

// extractMetadataKey extracts the metadata key from cluster labels with the pattern "metadata.greenhouse.sap/<key>".
// For example, "metadata.greenhouse.sap/region" becomes "region".
// Returns empty string if the label doesn't match the metadata pattern.
func extractMetadataKey(labelKey string) string {
	result, ok := strings.CutPrefix(labelKey, greenhouseapis.LabelKeyMetadataPrefix)
	if ok {
		return result
	}
	return ""
}
