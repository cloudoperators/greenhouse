// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package authz

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	authv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	"github.com/cloudoperators/greenhouse/pkg/mocks"
)

var greenhouseKinds = []string{"Catalog", "Plugin", "PluginPreset"}

func greenhouseRESTMapper() meta.RESTMapper {
	gv := schema.GroupVersion{Group: greenhouseapis.GroupName, Version: "v1alpha1"}
	m := meta.NewDefaultRESTMapper([]schema.GroupVersion{gv})
	for _, kind := range greenhouseKinds {
		m.Add(gv.WithKind(kind), meta.RESTScopeNamespace)
	}
	return m
}

func greenhouseResource(resource, ownedBy string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{Group: greenhouseapis.GroupName, Version: "v1alpha1", Kind: resource})
	obj.SetName("some-" + resource)
	obj.SetNamespace(testNamespace)
	if ownedBy != "" {
		obj.SetLabels(map[string]string{greenhouseapis.LabelKeyOwnedBy: ownedBy})
	}
	return obj
}

func expectGetUnstructured(mc *mocks.MockClient, want *unstructured.Unstructured) {
	mc.EXPECT().Get(mock.Anything, mock.Anything, mock.AnythingOfType("*unstructured.Unstructured")).
		Run(func(_ context.Context, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) {
			u := obj.(*unstructured.Unstructured)
			u.Object = want.DeepCopy().Object
		}).Return(nil).Maybe()
}

func expectGetUnstructuredNotFound(mc *mocks.MockClient) {
	mc.EXPECT().Get(mock.Anything, mock.Anything, mock.AnythingOfType("*unstructured.Unstructured")).
		Return(apierrors.NewNotFound(schema.GroupResource{Group: greenhouseapis.GroupName, Resource: "catalogs"}, "some-Catalog")).Maybe()
}

func TestAuthorizeGreenhouseResource(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(mc *mocks.MockClient)
		attrs       authv1.ResourceAttributes
		groups      []string
		wantAllowed bool
	}{
		{
			name: "it should allow patch on a catalog owned by the support-group",
			setup: func(mc *mocks.MockClient) {
				mc.EXPECT().RESTMapper().Return(greenhouseRESTMapper()).Maybe()
				expectGetUnstructured(mc, greenhouseResource("Catalog", testTeam))
				expectGetTeam(mc, supportGroupTeam(testTeam))
			},
			attrs:       authv1.ResourceAttributes{Group: greenhouseapis.GroupName, Verb: "patch", Resource: "catalogs", Namespace: testNamespace, Name: "some-Catalog"},
			groups:      []string{supportGroupClaimPrefix + testTeam},
			wantAllowed: true,
		},
		{
			name: "it should allow delete on a plugin owned by the support-group",
			setup: func(mc *mocks.MockClient) {
				mc.EXPECT().RESTMapper().Return(greenhouseRESTMapper()).Maybe()
				expectGetUnstructured(mc, greenhouseResource("Plugin", testTeam))
				expectGetTeam(mc, supportGroupTeam(testTeam))
			},
			attrs:       authv1.ResourceAttributes{Group: greenhouseapis.GroupName, Verb: "delete", Resource: "plugins", Namespace: testNamespace, Name: "some-Plugin"},
			groups:      []string{supportGroupClaimPrefix + testTeam},
			wantAllowed: true,
		},
		{
			name: "it should deny update on a pluginpreset owned by another support-group",
			setup: func(mc *mocks.MockClient) {
				mc.EXPECT().RESTMapper().Return(greenhouseRESTMapper()).Maybe()
				expectGetUnstructured(mc, greenhouseResource("PluginPreset", "observability"))
				expectGetTeam(mc, supportGroupTeam("observability"))
			},
			attrs:       authv1.ResourceAttributes{Group: greenhouseapis.GroupName, Verb: "update", Resource: "pluginpresets", Namespace: testNamespace, Name: "some-PluginPreset"},
			groups:      []string{supportGroupClaimPrefix + testTeam},
			wantAllowed: false,
		},
		{
			name: "it should deny access when the resource has no owned-by label",
			setup: func(mc *mocks.MockClient) {
				mc.EXPECT().RESTMapper().Return(greenhouseRESTMapper()).Maybe()
				expectGetUnstructured(mc, greenhouseResource("Catalog", ""))
			},
			attrs:       authv1.ResourceAttributes{Group: greenhouseapis.GroupName, Verb: "patch", Resource: "catalogs", Namespace: testNamespace, Name: "some-Catalog"},
			groups:      []string{supportGroupClaimPrefix + testTeam},
			wantAllowed: false,
		},
		{
			name: "it should deny access when the resource fetch fails",
			setup: func(mc *mocks.MockClient) {
				mc.EXPECT().RESTMapper().Return(greenhouseRESTMapper()).Maybe()
				expectGetUnstructuredNotFound(mc)
			},
			attrs:       authv1.ResourceAttributes{Group: greenhouseapis.GroupName, Verb: "patch", Resource: "catalogs", Namespace: testNamespace, Name: "some-Catalog"},
			groups:      []string{supportGroupClaimPrefix + testTeam},
			wantAllowed: false,
		},
		{
			name: "it should deny access when the owner team is not a support-group",
			setup: func(mc *mocks.MockClient) {
				mc.EXPECT().RESTMapper().Return(greenhouseRESTMapper()).Maybe()
				expectGetUnstructured(mc, greenhouseResource("Catalog", testTeam))
				expectGetTeamNotFound(mc)
			},
			attrs:       authv1.ResourceAttributes{Group: greenhouseapis.GroupName, Verb: "patch", Resource: "catalogs", Namespace: testNamespace, Name: "some-Catalog"},
			groups:      []string{supportGroupClaimPrefix + testTeam},
			wantAllowed: false,
		},
		{
			name:        "it should deny access without a resource name",
			attrs:       authv1.ResourceAttributes{Group: greenhouseapis.GroupName, Verb: "patch", Resource: "catalogs", Namespace: testNamespace},
			groups:      []string{supportGroupClaimPrefix + testTeam},
			wantAllowed: false,
		},
		{
			name:        "it should deny access without a support-group claim",
			attrs:       authv1.ResourceAttributes{Group: greenhouseapis.GroupName, Verb: "patch", Resource: "catalogs", Namespace: testNamespace, Name: "some-Catalog"},
			groups:      []string{"organization:demo"},
			wantAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := mocks.NewMockClient(t)
			if tt.setup != nil {
				tt.setup(mc)
			}
			status := doReview(t, mc, authv1.SubjectAccessReviewSpec{
				Groups:             tt.groups,
				ResourceAttributes: new(tt.attrs),
			})
			require.Equal(t, tt.wantAllowed, status.Allowed, status.Reason)
		})
	}
}
