// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package dex

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/dexidp/dex/connector"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var groupMock = []string{"IDP_GROUP_NAME_MATCHING_TEAM_1",
	"IDP_GROUP_NAME_MATCHING_TEAM_2",
	"ARBITRARY_IDP_GROUP_NAME_1",
	"IDP_GROUP_NAME_MATCHING_TEAM_3",
	"IDP_GROUP_NAME_MATCHING_TEAM_4",
	"ARBITRARY_IDP_GROUP_NAME_2"}

func TestGroups(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Connector Suite")
}

var _ = BeforeSuite(func() {

	test.TestBeforeSuite()

	err := test.K8sClient.Create(context.TODO(), &greenhousev1alpha1.Organization{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousev1alpha1.GroupVersion.Group,
			Kind:       "Organization",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name: "test-org",
		},
		Spec: greenhousev1alpha1.OrganizationSpec{
			Description:            "Test Organization",
			MappedOrgAdminIDPGroup: "IDP_GROUP_NAME_MATCHING_TEAM_1",
		},
	}, &client.CreateOptions{})
	Expect(err).ToNot(HaveOccurred(), "There should be no error when creating an organization")

	err = test.K8sClient.Create(context.TODO(), &greenhousev1alpha1.Team{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousev1alpha1.GroupVersion.Group,
			Kind:       "Team",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-team-1",
			Namespace: test.TestNamespace,
			Labels: map[string]string{
				"greenhouse.sap/test-team-category-1": "true",
				"some-key":                            "some-value",
			},
		},
		Spec: greenhousev1alpha1.TeamSpec{
			Description:    "Test Team 1",
			MappedIDPGroup: "IDP_GROUP_NAME_MATCHING_TEAM_1",
		},
	}, &client.CreateOptions{})
	Expect(err).ToNot(HaveOccurred(), "There should be no error when creating a team")

	err = test.K8sClient.Create(context.TODO(), &greenhousev1alpha1.Team{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousev1alpha1.GroupVersion.Group,
			Kind:       "Team",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-team-2",
			Namespace: test.TestNamespace,
			Labels: map[string]string{
				"greenhouse.sap/test-team-category-2": "true",
				"some-other-key":                      "some-other-value",
			},
		},
		Spec: greenhousev1alpha1.TeamSpec{
			Description:    "Test Team 2",
			MappedIDPGroup: "IDP_GROUP_NAME_MATCHING_TEAM_2",
		},
	}, &client.CreateOptions{})
	Expect(err).ToNot(HaveOccurred(), "There should be no error when creating a team")

	err = test.K8sClient.Create(context.TODO(), &greenhousev1alpha1.Team{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousev1alpha1.GroupVersion.Group,
			Kind:       "Team",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-team-3",
			Namespace: test.TestNamespace,
		},
		Spec: greenhousev1alpha1.TeamSpec{
			Description:    "Test Team 3",
			MappedIDPGroup: "IDP_GROUP_NAME_MATCHING_TEAM_3",
		},
	}, &client.CreateOptions{})
	Expect(err).ToNot(HaveOccurred(), "There should be no error when creating a team")

	err = test.K8sClient.Create(context.TODO(), &greenhousev1alpha1.Team{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousev1alpha1.GroupVersion.Group,
			Kind:       "Team",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-team-4",
			Namespace: test.TestNamespace,
		},
		Spec: greenhousev1alpha1.TeamSpec{
			Description:    "Test Team 4",
			MappedIDPGroup: "IDP_GROUP_NAME_MATCHING_TEAM_4",
		},
	}, &client.CreateOptions{})
	Expect(err).ToNot(HaveOccurred(), "There should be no error when creating a team")
})

var _ = AfterSuite(func() {
	test.TestAfterSuite()
})

var _ = Describe("Getting groups for token", func() {
	It("Should return the correct groups", func() {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		connectorOIDC := oidcConnector{conn: new(connector.Connector), logger: logger, client: test.K8sClient, id: test.TestNamespace}
		groups, err := connectorOIDC.getGroups(test.TestNamespace, groupMock, context.TODO())
		Expect(err).ToNot(HaveOccurred(), "There should be no error when getting groups")
		Expect(groups).To(Equal([]string{"organization:" + test.TestNamespace, "team:test-team-1", "test-team-category-1:test-team-1", "role:" + test.TestNamespace + ":admin", "team:test-team-2", "test-team-category-2:test-team-2", "team:test-team-3", "team:test-team-4"}), "The groups should be correct")
	})
})

// fakeUpstreamConnector is a minimal connector.Connector used to drive
// oidcConnector.HandleCallback and Refresh without a real upstream IdP.
type fakeUpstreamConnector struct {
	identity connector.Identity
}

var (
	_ connector.CallbackConnector = (*fakeUpstreamConnector)(nil)
	_ connector.RefreshConnector  = (*fakeUpstreamConnector)(nil)
)

func (f *fakeUpstreamConnector) LoginURL(connector.Scopes, string, string) (loginURL string, connData []byte, err error) {
	return "", nil, nil
}

func (f *fakeUpstreamConnector) HandleCallback(connector.Scopes, []byte, *http.Request) (connector.Identity, error) {
	return f.identity, nil
}

func (f *fakeUpstreamConnector) Refresh(context.Context, connector.Scopes, connector.Identity) (connector.Identity, error) {
	return f.identity, nil
}

var _ = Describe("Overriding email_verified", func() {
	It("Should force EmailVerified to true on HandleCallback when overrideEmailVerified is set", func() {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		upstream := &fakeUpstreamConnector{identity: connector.Identity{Email: "user@example.com", EmailVerified: false}}
		c := oidcConnector{conn: upstream, logger: logger, client: test.K8sClient, id: test.TestNamespace, overrideEmailVerified: true}
		identity, err := c.HandleCallback(connector.Scopes{}, nil, &http.Request{})
		Expect(err).ToNot(HaveOccurred(), "There should be no error handling the callback")
		Expect(identity.EmailVerified).To(BeTrue(), "EmailVerified should be forced to true even though the upstream reported false")
	})

	It("Should keep the upstream EmailVerified on HandleCallback when overrideEmailVerified is not set", func() {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		upstream := &fakeUpstreamConnector{identity: connector.Identity{Email: "user@example.com", EmailVerified: false}}
		c := oidcConnector{conn: upstream, logger: logger, client: test.K8sClient, id: test.TestNamespace, overrideEmailVerified: false}
		identity, err := c.HandleCallback(connector.Scopes{}, nil, &http.Request{})
		Expect(err).ToNot(HaveOccurred(), "There should be no error handling the callback")
		Expect(identity.EmailVerified).To(BeFalse(), "EmailVerified should reflect the upstream value")
	})

	It("Should force EmailVerified to true on Refresh when overrideEmailVerified is set", func() {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		upstream := &fakeUpstreamConnector{identity: connector.Identity{Email: "user@example.com", EmailVerified: false}}
		c := oidcConnector{conn: upstream, logger: logger, client: test.K8sClient, id: test.TestNamespace, overrideEmailVerified: true}
		identity, err := c.Refresh(context.TODO(), connector.Scopes{}, connector.Identity{})
		Expect(err).ToNot(HaveOccurred(), "There should be no error refreshing the identity")
		Expect(identity.EmailVerified).To(BeTrue(), "EmailVerified should be forced to true even though the upstream reported false")
	})
})
