// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

//go:build organizationE2E

package organization

import (
	"context"
	_ "embed"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/e2e/shared"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	testutil "github.com/cloudoperators/greenhouse/internal/test"
)

var (
	env           *shared.TestEnv
	ctx           context.Context
	adminClient   client.Client
	testStartTime time.Time
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Organization E2E Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx = context.Background()
	env = shared.NewExecutionEnv()

	var err error
	adminClient, err = clientutil.NewK8sClientFromRestClientGetter(env.AdminRestClientGetter)
	Expect(err).ToNot(HaveOccurred(), "no error creating the admin client")

	testStartTime = time.Now().UTC()
})

var _ = AfterSuite(func() {
	env.GenerateControllerLogs(ctx, testStartTime)
})

var _ = Describe("Organization E2E", Ordered, func() {
	Context("Happy path - creating and deleting organization", func() {
		org := testutil.NewOrganization(ctx, "organization-1-e2e")

		It("should onboard organization", func() {
			err := adminClient.Create(ctx, org)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should have Organization CR created", func() {
			Eventually(func(g Gomega) {
				err := adminClient.Get(ctx, client.ObjectKeyFromObject(org), org)
				g.Expect(err).ToNot(HaveOccurred())
			}).Should(Succeed(), "organization should be created")
		})

		It("should have namespace created", func() {
			Eventually(func(g Gomega) {
				err := adminClient.Get(ctx, client.ObjectKey{Name: org.Name}, &corev1.Namespace{})
				g.Expect(err).ToNot(HaveOccurred())
			}).Should(Succeed(), "namespace should be created")
		})

		DescribeTable("should have default resources created",
			func(objKey client.ObjectKey, obj client.Object) {
				entryLabel := CurrentSpecReport().LeafNodeText

				Eventually(func(g Gomega) {
					err := adminClient.Get(ctx, objKey, obj)
					g.Expect(err).ToNot(HaveOccurred())
				}).Should(Succeed(), "resource %s expected to be created", entryLabel)
			},
			// TeamRoles.
			Entry("TeamRole cluster-admin", client.ObjectKey{Name: "cluster-admin", Namespace: "organization-1-e2e"}, &greenhousev1alpha1.TeamRole{}),
			Entry("TeamRole cluster-viewer", client.ObjectKey{Name: "cluster-viewer", Namespace: "organization-1-e2e"}, &greenhousev1alpha1.TeamRole{}),
			Entry("TeamRole cluster-developer", client.ObjectKey{Name: "cluster-developer", Namespace: "organization-1-e2e"}, &greenhousev1alpha1.TeamRole{}),
			Entry("TeamRole application-developer", client.ObjectKey{Name: "application-developer", Namespace: "organization-1-e2e"}, &greenhousev1alpha1.TeamRole{}),
			Entry("TeamRole node-maintainer", client.ObjectKey{Name: "node-maintainer", Namespace: "organization-1-e2e"}, &greenhousev1alpha1.TeamRole{}),
			Entry("TeamRole namespace-creator", client.ObjectKey{Name: "namespace-creator", Namespace: "organization-1-e2e"}, &greenhousev1alpha1.TeamRole{}),
			// Teams.
			Entry("Team admin", client.ObjectKey{Name: "organization-1-e2e-admin", Namespace: "organization-1-e2e"}, &greenhousev1alpha1.Team{}),
			// Roles.
			Entry("Role admin", client.ObjectKey{Name: "role:organization-1-e2e:admin", Namespace: "organization-1-e2e"}, &rbacv1.Role{}),
			Entry("Role member", client.ObjectKey{Name: "organization:organization-1-e2e", Namespace: "organization-1-e2e"}, &rbacv1.Role{}),
			Entry("Role clusterAdmin", client.ObjectKey{Name: "role:organization-1-e2e:cluster-admin", Namespace: "organization-1-e2e"}, &rbacv1.Role{}),
			Entry("Role pluginAdmin", client.ObjectKey{Name: "role:organization-1-e2e:plugin-admin", Namespace: "organization-1-e2e"}, &rbacv1.Role{}),
			// RoleBindings.
			Entry("RoleBinding admin", client.ObjectKey{Name: "role:organization-1-e2e:admin", Namespace: "organization-1-e2e"}, &rbacv1.RoleBinding{}),
			Entry("RoleBinding member", client.ObjectKey{Name: "organization:organization-1-e2e", Namespace: "organization-1-e2e"}, &rbacv1.RoleBinding{}),
			// ClusterRoleBindings.
			Entry("ClusterRoleBinding admin", client.ObjectKey{Name: "role:organization-1-e2e:admin", Namespace: ""}, &rbacv1.ClusterRoleBinding{}),
			Entry("ClusterRoleBinding member", client.ObjectKey{Name: "organization:organization-1-e2e", Namespace: ""}, &rbacv1.ClusterRoleBinding{}),
			// ClusterRoles.
			Entry("ClusterRole admin", client.ObjectKey{Name: "role:organization-1-e2e:admin", Namespace: ""}, &rbacv1.ClusterRole{}),
			Entry("ClusterRole member", client.ObjectKey{Name: "organization:organization-1-e2e", Namespace: ""}, &rbacv1.ClusterRole{}),
		)

		It("should mark Organization CR as Ready", func() {
			Eventually(func(g Gomega) {
				err := adminClient.Get(ctx, client.ObjectKeyFromObject(org), org)
				Expect(err).ToNot(HaveOccurred())

				g.Expect(org.Status.Conditions).Should(ContainElement(
					MatchFields(IgnoreExtras, Fields{
						"Type":   Equal(greenhousemetav1alpha1.ReadyCondition),
						"Status": Equal(metav1.ConditionTrue),
					})))
			}).Should(Succeed(), "organization should have Ready condition")
		})

		It("should offboard organization", func() {
			if err := adminClient.Delete(ctx, org); err != nil && !apierrors.IsNotFound(err) {
				Fail(fmt.Sprintf("deleting Organization should not error: %v", err))
			}
		})

		It("should have Organization CR deleted", func() {
			Eventually(func(g Gomega) {
				err := adminClient.Get(ctx, client.ObjectKeyFromObject(org), org)
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
			}).Should(Succeed(), "organization should be fully deleted")
		})

		It("should have namespace deleted", func() {
			Eventually(func(g Gomega) {
				err := adminClient.Get(ctx, client.ObjectKey{Name: org.Name}, &corev1.Namespace{})
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
			}).Should(Succeed(), "namespace should be fully deleted")
		})

		DescribeTable("should have default resources removed",
			func(objKey client.ObjectKey, obj client.Object) {
				entryLabel := CurrentSpecReport().LeafNodeText

				Eventually(func(g Gomega) {
					err := adminClient.Get(ctx, objKey, obj)
					g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
				}).Should(Succeed(), "resource %s should be fully deleted", entryLabel)
			},
			// TeamRoles.
			Entry("TeamRole cluster-admin", client.ObjectKey{Name: "cluster-admin", Namespace: "organization-1-e2e"}, &greenhousev1alpha1.TeamRole{}),
			Entry("TeamRole cluster-viewer", client.ObjectKey{Name: "cluster-viewer", Namespace: "organization-1-e2e"}, &greenhousev1alpha1.TeamRole{}),
			Entry("TeamRole cluster-developer", client.ObjectKey{Name: "cluster-developer", Namespace: "organization-1-e2e"}, &greenhousev1alpha1.TeamRole{}),
			Entry("TeamRole application-developer", client.ObjectKey{Name: "application-developer", Namespace: "organization-1-e2e"}, &greenhousev1alpha1.TeamRole{}),
			Entry("TeamRole node-maintainer", client.ObjectKey{Name: "node-maintainer", Namespace: "organization-1-e2e"}, &greenhousev1alpha1.TeamRole{}),
			Entry("TeamRole namespace-creator", client.ObjectKey{Name: "namespace-creator", Namespace: "organization-1-e2e"}, &greenhousev1alpha1.TeamRole{}),
			// Teams.
			Entry("Team admin", client.ObjectKey{Name: "organization-1-e2e-admin", Namespace: "organization-1-e2e"}, &greenhousev1alpha1.Team{}),
			// Roles.
			Entry("Role admin", client.ObjectKey{Name: "role:organization-1-e2e:admin", Namespace: "organization-1-e2e"}, &rbacv1.Role{}),
			Entry("Role member", client.ObjectKey{Name: "organization:organization-1-e2e", Namespace: "organization-1-e2e"}, &rbacv1.Role{}),
			Entry("Role clusterAdmin", client.ObjectKey{Name: "role:organization-1-e2e:cluster-admin", Namespace: "organization-1-e2e"}, &rbacv1.Role{}),
			Entry("Role pluginAdmin", client.ObjectKey{Name: "role:organization-1-e2e:plugin-admin", Namespace: "organization-1-e2e"}, &rbacv1.Role{}),
			// RoleBindings.
			Entry("RoleBinding admin", client.ObjectKey{Name: "role:organization-1-e2e:admin", Namespace: "organization-1-e2e"}, &rbacv1.RoleBinding{}),
			Entry("RoleBinding member", client.ObjectKey{Name: "organization:organization-1-e2e", Namespace: "organization-1-e2e"}, &rbacv1.RoleBinding{}),
			// ClusterRoleBindings.
			Entry("ClusterRoleBinding admin", client.ObjectKey{Name: "role:organization-1-e2e:admin", Namespace: ""}, &rbacv1.ClusterRoleBinding{}),
			Entry("ClusterRoleBinding member", client.ObjectKey{Name: "organization:organization-1-e2e", Namespace: ""}, &rbacv1.ClusterRoleBinding{}),
			// ClusterRoles.
			Entry("ClusterRole admin", client.ObjectKey{Name: "role:organization-1-e2e:admin", Namespace: ""}, &rbacv1.ClusterRole{}),
			Entry("ClusterRole member", client.ObjectKey{Name: "organization:organization-1-e2e", Namespace: ""}, &rbacv1.ClusterRole{}),
		)
	})
})
