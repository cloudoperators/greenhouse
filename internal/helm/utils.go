// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"sort"
	"time"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
)

// Default Greenhouse helm timeout duration in seconds for install, upgrade and rollback actions.
const helmReleaseTimeoutSeconds int = 300

// getHelmTimeout gets a timeout duration for helm release install, upgrade and rollback actions.
// Tries to get the value from HELM_RELEASE_TIMEOUT evironment variable, otherwise gets the default value.
// Mainly used for E2E tests, because in deployment mode this should always be set to the default 5 minutes.
func getHelmTimeout() time.Duration {
	val := clientutil.GetIntEnvWithDefault("HELM_RELEASE_TIMEOUT", helmReleaseTimeoutSeconds)
	return time.Duration(val) * time.Second
}

func MergePluginAndPluginOptionValueSlice(pluginOptions []greenhousev1alpha1.PluginOption, pluginOptionValues []greenhousev1alpha1.PluginOptionValue) []greenhousev1alpha1.PluginOptionValue {
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

// MergePluginOptionValues merges the given src into the dst PluginOptionValue slice.
func MergePluginOptionValues(dst, src []greenhousev1alpha1.PluginOptionValue) []greenhousev1alpha1.PluginOptionValue {
	if dst == nil {
		dst = make([]greenhousev1alpha1.PluginOptionValue, 0)
	}
	for _, srcOptionValue := range src {
		dst = setOrAppendNameValue(dst, srcOptionValue)
	}
	return dst
}
