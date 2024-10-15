// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cloudoperators/greenhouse/pkg/controllers/fixtures"
)

func TestReceiveObjectCopy(t *testing.T) {
	t.Parallel()
	testResource := &fixtures.Dummy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-resource",
			Namespace: "default",
			Labels: map[string]string{
				"key1": "value1",
			},
			Annotations: map[string]string{
				"annotation1": "value1",
			},
		},
	}

	ctx := createContextFromRuntimeObject(context.Background(), testResource)
	testResource.GetLabels()["key1"] = "value2"
	origResource, err := getOriginalResourceFromContext(ctx)

	require.NoError(t, err)
	require.Equal(t, "value1", origResource.GetLabels()["key1"])
	require.Equal(t, "value2", testResource.GetLabels()["key1"])
}
