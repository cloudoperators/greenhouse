// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package idproxy

import (
	"context"
	"log/slog"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/dexidp/dex/connector"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousesapv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/test"
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

	err := test.K8sClient.Create(context.TODO(), &greenhousesapv1alpha1.Organization{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousesapv1alpha1.GroupVersion.Group,
			Kind:       "Organization",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name: "test-org",
		},
		Spec: greenhousesapv1alpha1.OrganizationSpec{
			Description:            "Test Organization",
			MappedOrgAdminIDPGroup: "IDP_GROUP_NAME_MATCHING_TEAM_1",
		},
	}, &client.CreateOptions{})
	Expect(err).ToNot(HaveOccurred(), "There should be no error when creating an organization")

	err = test.K8sClient.Create(context.TODO(), &greenhousesapv1alpha1.Team{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousesapv1alpha1.GroupVersion.Group,
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
		Spec: greenhousesapv1alpha1.TeamSpec{
			Description:    "Test Team 1",
			MappedIDPGroup: "IDP_GROUP_NAME_MATCHING_TEAM_1",
		},
	}, &client.CreateOptions{})
	Expect(err).ToNot(HaveOccurred(), "There should be no error when creating a team")

	err = test.K8sClient.Create(context.TODO(), &greenhousesapv1alpha1.Team{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousesapv1alpha1.GroupVersion.Group,
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
		Spec: greenhousesapv1alpha1.TeamSpec{
			Description:    "Test Team 2",
			MappedIDPGroup: "IDP_GROUP_NAME_MATCHING_TEAM_2",
		},
	}, &client.CreateOptions{})
	Expect(err).ToNot(HaveOccurred(), "There should be no error when creating a team")

	err = test.K8sClient.Create(context.TODO(), &greenhousesapv1alpha1.Team{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousesapv1alpha1.GroupVersion.Group,
			Kind:       "Team",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-team-3",
			Namespace: test.TestNamespace,
		},
		Spec: greenhousesapv1alpha1.TeamSpec{
			Description:    "Test Team 3",
			MappedIDPGroup: "IDP_GROUP_NAME_MATCHING_TEAM_3",
		},
	}, &client.CreateOptions{})
	Expect(err).ToNot(HaveOccurred(), "There should be no error when creating a team")

	err = test.K8sClient.Create(context.TODO(), &greenhousesapv1alpha1.Team{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousesapv1alpha1.GroupVersion.Group,
			Kind:       "Team",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-team-4",
			Namespace: test.TestNamespace,
		},
		Spec: greenhousesapv1alpha1.TeamSpec{
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
