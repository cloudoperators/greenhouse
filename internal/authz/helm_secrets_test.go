// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package authz

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/mocks"
)

const (
	testNamespace   = "demo"
	testTeam        = "containers"
	testReleaseName = "sh.helm.release.v1.containers.v3"
)

func helmSecret(ownedBy string) *corev1.Secret {
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testReleaseName,
			Namespace: testNamespace,
		},
	}
	if ownedBy != "" {
		s.Labels = map[string]string{greenhouseapis.LabelKeyOwnedBy: ownedBy}
	}
	return s
}

func supportGroupTeam(name string) *greenhousev1alpha1.Team {
	return &greenhousev1alpha1.Team{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
			Labels:    map[string]string{greenhouseapis.LabelKeySupportGroup: "true"},
		},
	}
}

func expectGetSecret(mc *mocks.MockClient, want *corev1.Secret) {
	mc.EXPECT().Get(mock.Anything, mock.Anything, mock.AnythingOfType("*v1.Secret")).
		Run(func(_ context.Context, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) {
			*(obj.(*corev1.Secret)) = *want
		}).Return(nil).Maybe()
}

func expectGetSecretNotFound(mc *mocks.MockClient) {
	mc.EXPECT().Get(mock.Anything, mock.Anything, mock.AnythingOfType("*v1.Secret")).
		Return(apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, testReleaseName)).Maybe()
}

func expectGetTeam(mc *mocks.MockClient, want *greenhousev1alpha1.Team) {
	mc.EXPECT().Get(mock.Anything, mock.Anything, mock.AnythingOfType("*v1alpha1.Team")).
		Run(func(_ context.Context, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) {
			*(obj.(*greenhousev1alpha1.Team)) = *want
		}).Return(nil).Maybe()
}

func expectGetTeamNotFound(mc *mocks.MockClient) {
	mc.EXPECT().Get(mock.Anything, mock.Anything, mock.AnythingOfType("*v1alpha1.Team")).
		Return(apierrors.NewNotFound(schema.GroupResource{Resource: "teams"}, testTeam)).Maybe()
}

func ownerSelector() *authv1.LabelSelectorAttributes {
	return &authv1.LabelSelectorAttributes{
		Requirements: []metav1.LabelSelectorRequirement{
			{Key: "owner", Operator: metav1.LabelSelectorOpIn, Values: []string{"helm"}},
		},
	}
}

func doReview(t *testing.T, mc *mocks.MockClient, spec authv1.SubjectAccessReviewSpec) authv1.SubjectAccessReviewStatus {
	t.Helper()
	handler := NewAuthorizer(mc, logf.Log)

	body, err := json.Marshal(authv1.SubjectAccessReview{Spec: spec})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/authorize", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var out authv1.SubjectAccessReview
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	return out.Status
}

func TestAuthorizeHelmSecret(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(mc *mocks.MockClient)
		attrs       authv1.ResourceAttributes
		user        string
		groups      []string
		wantAllowed bool
	}{
		{
			name:        "it should allow list with owner=helm selector",
			attrs:       authv1.ResourceAttributes{Verb: "list", Resource: "secrets", Namespace: testNamespace, LabelSelector: ownerSelector()},
			groups:      []string{supportGroupClaimPrefix + testTeam},
			wantAllowed: true,
		},
		{
			name:        "it should deny list without owner selector",
			attrs:       authv1.ResourceAttributes{Verb: "list", Resource: "secrets", Namespace: testNamespace},
			groups:      []string{supportGroupClaimPrefix + testTeam},
			wantAllowed: false,
		},
		{
			name:        "it should allow create by a support-group member",
			attrs:       authv1.ResourceAttributes{Verb: "create", Resource: "secrets", Namespace: testNamespace},
			groups:      []string{supportGroupClaimPrefix + testTeam},
			wantAllowed: true,
		},
		{
			name:        "it should deny create without a support-group claim",
			attrs:       authv1.ResourceAttributes{Verb: "create", Resource: "secrets", Namespace: testNamespace},
			groups:      []string{"organization:demo"},
			wantAllowed: false,
		},
		{
			name: "it should allow get when owned-by matches the support-group",
			setup: func(mc *mocks.MockClient) {
				expectGetSecret(mc, helmSecret(testTeam))
				expectGetTeam(mc, supportGroupTeam(testTeam))
			},
			attrs:       authv1.ResourceAttributes{Verb: "get", Resource: "secrets", Namespace: testNamespace, Name: testReleaseName},
			groups:      []string{supportGroupClaimPrefix + testTeam},
			wantAllowed: true,
		},
		{
			name: "it should allow update when owned-by matches the support-group",
			setup: func(mc *mocks.MockClient) {
				expectGetSecret(mc, helmSecret(testTeam))
				expectGetTeam(mc, supportGroupTeam(testTeam))
			},
			attrs:       authv1.ResourceAttributes{Verb: "update", Resource: "secrets", Namespace: testNamespace, Name: testReleaseName},
			groups:      []string{supportGroupClaimPrefix + testTeam},
			wantAllowed: true,
		},
		{
			name: "it should deny get when owned-by does not match the support-group",
			setup: func(mc *mocks.MockClient) {
				expectGetSecret(mc, helmSecret("observability"))
			},
			attrs:       authv1.ResourceAttributes{Verb: "get", Resource: "secrets", Namespace: testNamespace, Name: testReleaseName},
			groups:      []string{supportGroupClaimPrefix + testTeam},
			wantAllowed: false,
		},
		{
			name:        "it should deny get on a non-helm secret name",
			attrs:       authv1.ResourceAttributes{Verb: "get", Resource: "secrets", Namespace: testNamespace, Name: "demo-oidc"},
			groups:      []string{supportGroupClaimPrefix + testTeam},
			wantAllowed: false,
		},
		{
			name: "it should deny get on a helm secret without an owned-by label",
			setup: func(mc *mocks.MockClient) {
				expectGetSecret(mc, helmSecret(""))
			},
			attrs:       authv1.ResourceAttributes{Verb: "get", Resource: "secrets", Namespace: testNamespace, Name: testReleaseName},
			groups:      []string{supportGroupClaimPrefix + testTeam},
			wantAllowed: false,
		},
		{
			name: "it should deny get when the secret fetch fails",
			setup: func(mc *mocks.MockClient) {
				expectGetSecretNotFound(mc)
			},
			attrs:       authv1.ResourceAttributes{Verb: "get", Resource: "secrets", Namespace: testNamespace, Name: testReleaseName},
			groups:      []string{supportGroupClaimPrefix + testTeam},
			wantAllowed: false,
		},
		{
			name: "it should deny get when the owner team is not a support-group",
			setup: func(mc *mocks.MockClient) {
				expectGetSecret(mc, helmSecret(testTeam))
				expectGetTeamNotFound(mc)
			},
			attrs:       authv1.ResourceAttributes{Verb: "get", Resource: "secrets", Namespace: testNamespace, Name: testReleaseName},
			groups:      []string{supportGroupClaimPrefix + testTeam},
			wantAllowed: false,
		},
		{
			name:        "it should deny the delete verb on secrets",
			attrs:       authv1.ResourceAttributes{Verb: "delete", Resource: "secrets", Namespace: testNamespace, Name: testReleaseName},
			groups:      []string{supportGroupClaimPrefix + testTeam},
			wantAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := mocks.NewMockClient(t)
			if tt.setup != nil {
				tt.setup(mc)
			}
			attrs := tt.attrs
			status := doReview(t, mc, authv1.SubjectAccessReviewSpec{
				User:               tt.user,
				Groups:             tt.groups,
				ResourceAttributes: new(attrs),
			})
			require.Equal(t, tt.wantAllowed, status.Allowed, status.Reason)
		})
	}
}
