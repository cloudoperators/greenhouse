// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package scenarios

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/test"
)

// ITRBScenario defines all executable TeamRoleBinding E2E scenarios.
type ITRBScenario interface {
	// ExecuteSingleTeamRefScenario verifies RBAC for a single teamRefs entry (baseline).
	ExecuteSingleTeamRefScenario(ctx context.Context)
	// ExecuteMultipleTeamRefsScenario verifies RBAC for two teams in teamRefs.
	ExecuteMultipleTeamRefsScenario(ctx context.Context)
	// ExecuteDeprecatedTeamRefMigrationScenario verifies the webhook migrates teamRef → teamRefs.
	ExecuteDeprecatedTeamRefMigrationScenario(ctx context.Context)
	// ExecuteTeamRefsMutationScenario verifies subjects update when teamRefs are modified.
	ExecuteTeamRefsMutationScenario(ctx context.Context)
	// ExecutePartialFailureScenario verifies RBAC resilience when some teams are missing.
	ExecutePartialFailureScenario(ctx context.Context)
	// ExecuteNamespaceCreationScenario verifies createNamespaces=true with multiple teams.
	ExecuteNamespaceCreationScenario(ctx context.Context)
	// ExecuteClusterSelectorScenario verifies RBAC is applied only to clusters matching a selector.
	ExecuteClusterSelectorScenario(ctx context.Context)
}

// scenario holds the shared state for all TeamRoleBinding E2E scenarios.
type scenario struct {
	adminClient  client.Client
	remoteClient client.Client
	namespace    string
	clusterName  string
	teamAlpha    *greenhousev1alpha1.Team
	teamBeta     *greenhousev1alpha1.Team
	teamRole     *greenhousev1alpha1.TeamRole
}

// NewScenario constructs an ITRBScenario from the given test context.
func NewScenario(
	adminClient, remoteClient client.Client,
	namespace, clusterName string,
	teamAlpha, teamBeta *greenhousev1alpha1.Team,
	teamRole *greenhousev1alpha1.TeamRole,
) ITRBScenario {

	GinkgoHelper()
	return &scenario{
		adminClient:  adminClient,
		remoteClient: remoteClient,
		namespace:    namespace,
		clusterName:  clusterName,
		teamAlpha:    teamAlpha,
		teamBeta:     teamBeta,
		teamRole:     teamRole,
	}
}

// createTRB is a helper that creates or patches a TeamRoleBinding and returns it.
func (s *scenario) createTRB(ctx context.Context, name string, opts ...func(*greenhousev1alpha2.TeamRoleBinding)) *greenhousev1alpha2.TeamRoleBinding {
	GinkgoHelper()
	trb := test.NewTeamRoleBinding(ctx, name, s.namespace)
	_, err := clientutil.CreateOrPatch(ctx, s.adminClient, trb, func() error {
		for _, opt := range opts {
			opt(trb)
		}
		return nil
	})
	Expect(err).ToNot(HaveOccurred(), "there should be no error creating or patching %T %s", trb, trb.GetName())
	return trb
}

// cleanup deletes a TeamRoleBinding if it is non-nil, waiting for it to disappear.
func (s *scenario) cleanup(ctx context.Context, trb *greenhousev1alpha2.TeamRoleBinding) {
	GinkgoHelper()
	test.EventuallyDeleted(ctx, s.adminClient, trb)
}
