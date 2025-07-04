// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

//go:build organizationE2E

package organization

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/e2e/shared"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
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
	env = env.WithOrganization(ctx, adminClient, "./testdata/organization.yaml")
	testStartTime = time.Now().UTC()
})

var _ = AfterSuite(func() {
	org := &greenhousev1alpha1.Organization{ObjectMeta: metav1.ObjectMeta{Name: env.TestNamespace}}
	if err := adminClient.Delete(ctx, org); err != nil && !apierrors.IsNotFound(err) {
		Fail(fmt.Sprintf("deleting Organization %s should not error: %v", env.TestNamespace, err))
	}

	Eventually(func(g Gomega) {
		err := adminClient.Get(ctx, client.ObjectKey{Name: env.TestNamespace}, org)
		g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
	}).Should(Succeed(), fmt.Sprintf("Organization %s should be fully deleted", env.TestNamespace))

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: env.TestNamespace}}
	if err := adminClient.Delete(ctx, ns); err != nil && !apierrors.IsNotFound(err) {
		Fail(fmt.Sprintf("deleting Namespace %s should not error: %v", env.TestNamespace, err))
	}

	Eventually(func(g Gomega) {
		err := adminClient.Get(ctx, client.ObjectKey{Name: env.TestNamespace}, ns)
		g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
	}).Should(Succeed(), fmt.Sprintf("Namespace %s should be fully deleted", env.TestNamespace))

	env.GenerateControllerLogs(ctx, testStartTime)
})

var _ = Describe("Organization E2E", Ordered, func() {
	DescribeTable("onboarding Organization should create default resource",
		func(objKey client.ObjectKey, obj client.Object) {
			entryLabel := CurrentSpecReport().LeafNodeText

			Eventually(func(g Gomega) {
				err := adminClient.Get(ctx, objKey, obj)
				g.Expect(err).ToNot(HaveOccurred())
			}).Should(Succeed(), "expected %s to be created", entryLabel)
		},
		// TeamRoles.
		Entry("TeamRole cluster-admin", client.ObjectKey{Name: "cluster-admin", Namespace: "organization-e2e"}, &greenhousev1alpha1.TeamRole{}),
		Entry("TeamRole cluster-viewer", client.ObjectKey{Name: "cluster-viewer", Namespace: "organization-e2e"}, &greenhousev1alpha1.TeamRole{}),
		Entry("TeamRole cluster-developer", client.ObjectKey{Name: "cluster-developer", Namespace: "organization-e2e"}, &greenhousev1alpha1.TeamRole{}),
		Entry("TeamRole application-developer", client.ObjectKey{Name: "application-developer", Namespace: "organization-e2e"}, &greenhousev1alpha1.TeamRole{}),
		Entry("TeamRole node-maintainer", client.ObjectKey{Name: "node-maintainer", Namespace: "organization-e2e"}, &greenhousev1alpha1.TeamRole{}),
		Entry("TeamRole namespace-creator", client.ObjectKey{Name: "namespace-creator", Namespace: "organization-e2e"}, &greenhousev1alpha1.TeamRole{}),
		// Teams.
		Entry("Team admin", client.ObjectKey{Name: "organization-e2e-admin", Namespace: "organization-e2e"}, &greenhousev1alpha1.Team{}),
		// ClusterRoles.
		Entry("ClusterRole admin", client.ObjectKey{Name: "role:organization-e2e:admin", Namespace: ""}, &rbacv1.ClusterRole{}),
		Entry("ClusterRole member", client.ObjectKey{Name: "organization:organization-e2e", Namespace: ""}, &rbacv1.ClusterRole{}),
		// Roles.
		Entry("Role admin", client.ObjectKey{Name: "role:organization-e2e:admin", Namespace: "organization-e2e"}, &rbacv1.Role{}),
		Entry("Role member", client.ObjectKey{Name: "organization:organization-e2e", Namespace: "organization-e2e"}, &rbacv1.Role{}),
		Entry("Role clusterAdmin", client.ObjectKey{Name: "role:organization-e2e:cluster-admin", Namespace: "organization-e2e"}, &rbacv1.Role{}),
		Entry("Role pluginAdmin", client.ObjectKey{Name: "role:organization-e2e:plugin-admin", Namespace: "organization-e2e"}, &rbacv1.Role{}),
		// ClusterRoleBindings.
		Entry("ClusterRoleBinding admin", client.ObjectKey{Name: "role:organization-e2e:admin", Namespace: ""}, &rbacv1.ClusterRoleBinding{}),
		Entry("ClusterRoleBinding member", client.ObjectKey{Name: "organization:organization-e2e", Namespace: ""}, &rbacv1.ClusterRoleBinding{}),
		// RoleBindings.
		Entry("RoleBinding admin", client.ObjectKey{Name: "role:organization-e2e:admin", Namespace: "organization-e2e"}, &rbacv1.RoleBinding{}),
		Entry("RoleBinding member", client.ObjectKey{Name: "organization:organization-e2e", Namespace: "organization-e2e"}, &rbacv1.RoleBinding{}),
	)
})
