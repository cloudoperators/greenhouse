// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package features

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/mocks"
)

// Test_DexFeatures -  test dex storage type feature gate
func Test_DexFeatures(t *testing.T) {
	type testCase struct {
		name          string
		configMapData map[string]string
		getError      error
		expectedValue *string
	}

	testCases := []testCase{
		{
			name:          "it should return kubernetes as storage type from feature-flags cm",
			configMapData: map[string]string{DexFeatureKey: "storage: kubernetes\n"},
			expectedValue: clientutil.Ptr("kubernetes"),
		},
		{
			name:          "it should return postgres as storage type from feature-flags cm",
			configMapData: map[string]string{DexFeatureKey: "storage: postgres\n"},
			expectedValue: clientutil.Ptr("postgres"),
		},
		{
			name:          "it should return nil when storage type is not found in feature-flags cm",
			configMapData: map[string]string{"someOtherKey": "value\n"},
			expectedValue: nil, // should return nil since `dex` is missing
		},
		{
			name:          "it should return nil when storage type is empty in feature-flags cm",
			configMapData: map[string]string{DexFeatureKey: "storage: "},
			expectedValue: nil,
		},
		{
			name:          "it should return a nil instance of features when feature-flags cm is not found",
			getError:      apierrors.NewNotFound(schema.GroupResource{}, "configmap not found"),
			expectedValue: nil, // should return nil since ConfigMap is not found
		},
		{
			name:          "it should return nil when flag is malformed in feature-flags cm",
			configMapData: map[string]string{DexFeatureKey: "storage:: invalid_yaml"},
			expectedValue: nil, // should return an empty string and log an error
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = log.IntoContext(ctx, log.Log)

			mockK8sClient := &mocks.MockClient{}
			configMap := &corev1.ConfigMap{}

			if tc.getError != nil {
				mockK8sClient.On("Get", ctx, types.NamespacedName{
					Name: clientutil.GetEnvOrDefault("FEATURE_FLAGS", "greenhouse-feature-flags"), Namespace: clientutil.GetEnvOrDefault("POD_NAMESPACE", "greenhouse"),
				}, mock.Anything).Return(tc.getError)
			} else {
				configMap.Data = tc.configMapData
				mockK8sClient.On("Get", ctx, types.NamespacedName{
					Name: clientutil.GetEnvOrDefault("FEATURE_FLAGS", "greenhouse-feature-flags"), Namespace: clientutil.GetEnvOrDefault("POD_NAMESPACE", "greenhouse"),
				}, mock.Anything).Run(func(args mock.Arguments) {
					arg := args.Get(2).(*corev1.ConfigMap) //nolint:errcheck
					*arg = *configMap
				}).Return(nil)
			}

			// Create Features instance
			featuresInstance, err := NewFeatures(ctx, mockK8sClient, clientutil.GetEnvOrDefault("FEATURE_FLAGS", "greenhouse-feature-flags"), clientutil.GetEnvOrDefault("POD_NAMESPACE", "greenhouse"))
			if tc.getError != nil && client.IgnoreNotFound(tc.getError) == nil {
				assert.NoError(t, client.IgnoreNotFound(err))
				assert.Nil(t, featuresInstance, "Expected nil when ConfigMap is missing")
				return
			}
			assert.NoError(t, err)

			// Get Dex storage type
			var value *string
			if featuresInstance != nil {
				value = featuresInstance.GetDexStorageType(ctx)
			}

			// Assert expected value
			assert.Equal(t, tc.expectedValue, value)
			mockK8sClient.AssertExpectations(t)
		})
	}
}

// Test_PluginFeatures - test plugin expression evaluation feature gate
func Test_PluginFeatures(t *testing.T) {
	type testCase struct {
		name                         string
		configMapData                map[string]string
		getError                     error
		expectedExpressionEvaluation bool
		expectedIntegrationEnabled   bool
	}

	testCases := []testCase{
		{
			name:                         "it should return true when plugin expression evaluation is enabled",
			configMapData:                map[string]string{PluginFeatureKey: "expressionEvaluationEnabled: true\n"},
			expectedExpressionEvaluation: true,
			expectedIntegrationEnabled:   false,
		},
		{
			name:                         "it should return false when plugin expression evaluation is disabled",
			configMapData:                map[string]string{PluginFeatureKey: "expressionEvaluationEnabled: false\n"},
			expectedExpressionEvaluation: false,
			expectedIntegrationEnabled:   false,
		},
		{
			name:                         "it should return true when plugin integration is enabled",
			configMapData:                map[string]string{PluginFeatureKey: "integrationEnabled: true\n"},
			expectedExpressionEvaluation: false,
			expectedIntegrationEnabled:   true,
		},
		{
			name:                         "it should return both values when both are set",
			configMapData:                map[string]string{PluginFeatureKey: "expressionEvaluationEnabled: true\nintegrationEnabled: true\n"},
			expectedExpressionEvaluation: true,
			expectedIntegrationEnabled:   true,
		},
		{
			name:                         "it should return false when plugin key is not found in feature-flags cm",
			configMapData:                map[string]string{"someOtherKey": "value\n"},
			expectedExpressionEvaluation: false,
			expectedIntegrationEnabled:   false,
		},
		{
			name:                         "it should return false when feature-flags cm is not found",
			getError:                     apierrors.NewNotFound(schema.GroupResource{}, "configmap not found"),
			expectedExpressionEvaluation: false,
			expectedIntegrationEnabled:   false,
		},
		{
			name:                         "it should return false when flag is malformed in feature-flags cm",
			configMapData:                map[string]string{PluginFeatureKey: "expressionEvaluationEnabled:: invalid_yaml"},
			expectedExpressionEvaluation: false,
			expectedIntegrationEnabled:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = log.IntoContext(ctx, log.Log)

			mockK8sClient := &mocks.MockClient{}
			configMap := &corev1.ConfigMap{}

			if tc.getError != nil {
				mockK8sClient.On("Get", ctx, types.NamespacedName{
					Name: clientutil.GetEnvOrDefault("FEATURE_FLAGS", "greenhouse-feature-flags"), Namespace: clientutil.GetEnvOrDefault("POD_NAMESPACE", "greenhouse"),
				}, mock.Anything).Return(tc.getError)
			} else {
				configMap.Data = tc.configMapData
				mockK8sClient.On("Get", ctx, types.NamespacedName{
					Name: clientutil.GetEnvOrDefault("FEATURE_FLAGS", "greenhouse-feature-flags"), Namespace: clientutil.GetEnvOrDefault("POD_NAMESPACE", "greenhouse"),
				}, mock.Anything).Run(func(args mock.Arguments) {
					arg := args.Get(2).(*corev1.ConfigMap) //nolint:errcheck
					*arg = *configMap
				}).Return(nil)
			}

			// Create Features instance
			featuresInstance, err := NewFeatures(ctx, mockK8sClient, clientutil.GetEnvOrDefault("FEATURE_FLAGS", "greenhouse-feature-flags"), clientutil.GetEnvOrDefault("POD_NAMESPACE", "greenhouse"))
			if tc.getError != nil && client.IgnoreNotFound(tc.getError) == nil {
				assert.NoError(t, client.IgnoreNotFound(err))
				assert.Nil(t, featuresInstance, "Expected nil when ConfigMap is missing")

				expressionEvaluationValue := featuresInstance.IsExpressionEvaluationEnabled()
				integrationEnabledValue := featuresInstance.IsIntegrationEnabled()

				assert.Equal(t, tc.expectedExpressionEvaluation, expressionEvaluationValue)
				assert.Equal(t, tc.expectedIntegrationEnabled, integrationEnabledValue)
				mockK8sClient.AssertExpectations(t)
				return
			}
			assert.NoError(t, err)

			expressionEvaluationValue := featuresInstance.IsExpressionEvaluationEnabled()
			integrationEnabledValue := featuresInstance.IsIntegrationEnabled()

			// Assert expected values
			assert.Equal(t, tc.expectedExpressionEvaluation, expressionEvaluationValue)
			assert.Equal(t, tc.expectedIntegrationEnabled, integrationEnabledValue)
			mockK8sClient.AssertExpectations(t)
		})
	}
}
