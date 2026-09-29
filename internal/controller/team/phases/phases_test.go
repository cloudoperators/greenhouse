// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/scim"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
)

// noopRecorder satisfies events.EventRecorder without emitting anything.
type noopRecorder struct{}

func (noopRecorder) Eventf(_, _ runtime.Object, _, _, _, _ string, _ ...any) {}

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, greenhousev1alpha1.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))
	require.NoError(t, rbacv1.AddToScheme(s))
	return s
}

func testTeam(ns string, supportGroup bool) *greenhousev1alpha1.Team {
	t := &greenhousev1alpha1.Team{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "myteam",
			Namespace: ns,
		},
	}
	if supportGroup {
		t.Labels = map[string]string{greenhouseapis.LabelKeySupportGroup: "true"}
	}
	return t
}

func testOrg(name string, scimReady bool, scimConfig *greenhousev1alpha1.SCIMConfig) *greenhousev1alpha1.Organization {
	org := &greenhousev1alpha1.Organization{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	if scimReady {
		org.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.SCIMAPIAvailableCondition, "", ""))
	} else {
		org.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.SCIMAPIAvailableCondition, "", ""))
	}
	if scimConfig != nil {
		org.Spec.Authentication = &greenhousev1alpha1.Authentication{
			SCIMConfig: scimConfig,
		}
	}
	return org
}

// stubSCIMClient is a minimal ISCIMClient for tests.
type stubSCIMClient struct {
	users []scim.Resource
	err   error
}

func (s *stubSCIMClient) GetUsers(_ context.Context, _ *scim.QueryOptions) ([]scim.Resource, error) {
	return s.users, s.err
}

func (s *stubSCIMClient) GetGroups(_ context.Context, _ *scim.QueryOptions) ([]scim.Resource, error) {
	return nil, nil
}

func makeResource(username, first, last, email string, active bool) scim.Resource {
	return scim.Resource{
		UserName: username,
		Name: scim.UserNameField{
			GivenName:  first,
			FamilyName: last,
		},
		Emails: []scim.UserEmails{
			{Value: email, Primary: true},
		},
		Active: active,
	}
}

func TestEnsureCleanupIfNotSupportGroup_LabelAbsent_DeletesResources(t *testing.T) {
	team := testTeam("myorg", false)
	sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "myteam-sa", Namespace: "myorg"}}
	role := &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: "myteam-sa-token-request", Namespace: "myorg"}}
	rb := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "myteam-sa-token-request", Namespace: "myorg"}}

	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(team, sa, role, rb).Build()
	p := &Phase{Client: c, Recorder: noopRecorder{}}

	result, err := p.ensureCleanupIfNotSupportGroup(team)(context.Background())
	require.NoError(t, err)
	require.False(t, result.Break)

	require.True(t, apierrors.IsNotFound(c.Get(context.Background(), client.ObjectKeyFromObject(sa), &corev1.ServiceAccount{})))
	require.True(t, apierrors.IsNotFound(c.Get(context.Background(), client.ObjectKeyFromObject(role), &rbacv1.Role{})))
	require.True(t, apierrors.IsNotFound(c.Get(context.Background(), client.ObjectKeyFromObject(rb), &rbacv1.RoleBinding{})))
}

func TestEnsureCleanupIfNotSupportGroup_LabelPresent_NoOp(t *testing.T) {
	team := testTeam("myorg", true)
	sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "myteam-sa", Namespace: "myorg"}}

	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(team, sa).Build()
	p := &Phase{Client: c, Recorder: noopRecorder{}}

	result, err := p.ensureCleanupIfNotSupportGroup(team)(context.Background())
	require.NoError(t, err)
	require.False(t, result.Break)

	require.NoError(t, c.Get(context.Background(), client.ObjectKeyFromObject(sa), &corev1.ServiceAccount{}))
}

func TestEnsureSCIMMembers_OrgNotFound_Break(t *testing.T) {
	team := testTeam("missing-org", false)
	team.Spec.MappedIDPGroup = "mygroup"

	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(team).Build()
	p := &Phase{Client: c, Recorder: noopRecorder{}}

	result, err := p.ensureSCIMMembers(team)(context.Background())
	require.NoError(t, err) // IgnoreNotFound
	require.True(t, result.Break)
}

func TestEnsureSCIMMembers_NoMappedIDPGroup_Break(t *testing.T) {
	org := testOrg("myorg", true, nil)
	team := testTeam("myorg", false) // MappedIDPGroup is empty

	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(team, org).Build()
	p := &Phase{Client: c, Recorder: noopRecorder{}}

	result, err := p.ensureSCIMMembers(team)(context.Background())
	require.NoError(t, err)
	require.True(t, result.Break)
}

func TestEnsureSCIMMembers_SCIMAPIUnavailable_SetsConditionAndBreaks(t *testing.T) {
	org := testOrg("myorg", false, nil)
	team := testTeam("myorg", false)
	team.Spec.MappedIDPGroup = "mygroup"

	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(team, org).Build()
	p := &Phase{Client: c, Recorder: noopRecorder{}}

	result, err := p.ensureSCIMMembers(team)(context.Background())
	require.NoError(t, err)
	require.True(t, result.Break)

	cond := team.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.SCIMAccessReadyCondition)
	require.NotNil(t, cond)
	require.Equal(t, metav1.ConditionFalse, cond.Status)
	require.Equal(t, greenhousev1alpha1.SCIMAPIUnavailableReason, cond.Reason)
}

func TestEnsureSCIMMembers_SCIMConfigMissing_SetsConditionAndBreaks(t *testing.T) {
	org := testOrg("myorg", true, nil) // SCIM available but no config
	team := testTeam("myorg", false)
	team.Spec.MappedIDPGroup = "mygroup"

	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(team, org).Build()
	p := &Phase{Client: c, Recorder: noopRecorder{}}

	result, err := p.ensureSCIMMembers(team)(context.Background())
	require.NoError(t, err)
	require.True(t, result.Break)

	cond := team.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.SCIMAccessReadyCondition)
	require.NotNil(t, cond)
	require.Equal(t, metav1.ConditionFalse, cond.Status)
	require.Equal(t, greenhousev1alpha1.SecretNotFoundReason, cond.Reason)
}

func TestGetUsersFromSCIM(t *testing.T) {
	tests := []struct {
		name       string
		users      []scim.Resource
		err        error
		expectLen  int
		expectCond metav1.ConditionStatus
		expectErr  bool
	}{
		{
			name:       "valid users sets allMembersValid true",
			users:      []scim.Resource{makeResource("user1", "First", "Last", "first@example.com", true)},
			expectLen:  1,
			expectCond: metav1.ConditionTrue,
		},
		{
			name:       "inactive user sets allMembersValid false",
			users:      []scim.Resource{makeResource("user1", "First", "Last", "first@example.com", false)},
			expectLen:  0,
			expectCond: metav1.ConditionFalse,
		},
		{
			name:       "malformed user (empty username) sets allMembersValid false",
			users:      []scim.Resource{makeResource("", "First", "Last", "first@example.com", true)},
			expectLen:  0,
			expectCond: metav1.ConditionFalse,
		},
		{
			name:      "SCIM error returns error",
			err:       errors.New("scim down"),
			expectErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &stubSCIMClient{users: tt.users, err: tt.err}
			users, cond, err := getUsersFromSCIM(context.Background(), stub, "mygroup")
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, users, tt.expectLen)
			require.Equal(t, tt.expectCond, cond.Status)
		})
	}
}

func TestEnsureSupportGroupResources_LabelAbsent_NoOp(t *testing.T) {
	team := testTeam("myorg", false)
	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(team).Build()
	p := &Phase{Client: c, Recorder: noopRecorder{}}

	result, err := p.ensureSupportGroupResources(team)(context.Background())
	require.NoError(t, err)
	require.False(t, result.Break)

	require.True(t, apierrors.IsNotFound(c.Get(context.Background(), client.ObjectKey{Name: "myteam-sa", Namespace: "myorg"}, &corev1.ServiceAccount{})))
}

func TestEnsureSupportGroupResources_LabelPresent_CreatesResources(t *testing.T) {
	team := testTeam("myorg", true)
	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(team).Build()
	p := &Phase{Client: c, Recorder: noopRecorder{}}

	result, err := p.ensureSupportGroupResources(team)(context.Background())
	require.NoError(t, err)
	require.False(t, result.Break)

	require.NoError(t, c.Get(context.Background(), client.ObjectKey{Name: "myteam-sa", Namespace: "myorg"}, &corev1.ServiceAccount{}))
	require.NoError(t, c.Get(context.Background(), client.ObjectKey{Name: "myteam-sa-token-request", Namespace: "myorg"}, &rbacv1.Role{}))
	require.NoError(t, c.Get(context.Background(), client.ObjectKey{Name: "myteam-sa-token-request", Namespace: "myorg"}, &rbacv1.RoleBinding{}))
}

func TestEnsureSupportGroupResources_Idempotent(t *testing.T) {
	team := testTeam("myorg", true)
	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(team).Build()
	p := &Phase{Client: c, Recorder: noopRecorder{}}

	for i := range 2 {
		result, err := p.ensureSupportGroupResources(team)(context.Background())
		require.NoError(t, err, "call %d", i+1)
		require.False(t, result.Break, "call %d", i+1)
	}
}

func TestEnsureSupportGroupResourcesDeleted_ResourcesExist_DeletesThem(t *testing.T) {
	team := testTeam("myorg", true)
	sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "myteam-sa", Namespace: "myorg"}}
	role := &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: "myteam-sa-token-request", Namespace: "myorg"}}
	rb := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "myteam-sa-token-request", Namespace: "myorg"}}

	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(team, sa, role, rb).Build()
	p := &Phase{Client: c, Recorder: noopRecorder{}}

	result, err := p.ensureSupportGroupResourcesDeleted(team)(context.Background())
	require.NoError(t, err)
	require.False(t, result.Break)

	require.True(t, apierrors.IsNotFound(c.Get(context.Background(), client.ObjectKeyFromObject(sa), &corev1.ServiceAccount{})))
	require.True(t, apierrors.IsNotFound(c.Get(context.Background(), client.ObjectKeyFromObject(role), &rbacv1.Role{})))
	require.True(t, apierrors.IsNotFound(c.Get(context.Background(), client.ObjectKeyFromObject(rb), &rbacv1.RoleBinding{})))
}

func TestEnsureSupportGroupResourcesDeleted_ResourcesAbsent_NoError(t *testing.T) {
	team := testTeam("myorg", true)
	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(team).Build()
	p := &Phase{Client: c, Recorder: noopRecorder{}}

	result, err := p.ensureSupportGroupResourcesDeleted(team)(context.Background())
	require.NoError(t, err)
	require.False(t, result.Break)
}
