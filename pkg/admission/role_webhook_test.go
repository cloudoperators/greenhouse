// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package admission

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	extensionsgreenhouse "github.com/cloudoperators/greenhouse/pkg/apis/extensions.greenhouse"
	extensionsgreenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/extensions.greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

var testRole = &extensionsgreenhousev1alpha1.Role{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: test.TestNamespace,
		Name:      "test-role",
	},
	Spec: extensionsgreenhousev1alpha1.RoleSpec{
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get"},
				APIGroups: []string{"*"},
				Resources: []string{"*"},
			},
		},
	},
}

var testRoleBinding = &extensionsgreenhousev1alpha1.RoleBinding{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: test.TestNamespace,
		Name:      "test-rolebinding",
	},
	Spec: extensionsgreenhousev1alpha1.RoleBindingSpec{
		ClusterSelector: metav1.LabelSelector{
			MatchLabels: map[string]string{
				"test.greenhouse.sap/cluster": "test-cluster",
			},
		},
		RoleRef: "test-role",
	},
}

var _ = Describe("Validate Role Deletion", Ordered, func() {
	It("should not allow deleting a role with references", func() {
		err := test.K8sClient.Create(test.Ctx, testRole)
		Expect(err).ToNot(HaveOccurred(), "there should be no error creating the role")
		err = test.K8sClient.Create(test.Ctx, testRoleBinding)
		Expect(err).ToNot(HaveOccurred(), "there should be no error creating the rolebinding")

		err = test.K8sClient.Delete(test.Ctx, testRole)
		Expect(err).To(HaveOccurred(), "there should be an error deleting the role with references")
	})
})

// setupRoleBindingWebhookForTest adds an indexField for '.spec.roleRef', additionally to setting up the webhook for the RoleBinding resource. It is used in the webhook tests.
// we can't add this to the webhook setup because it's already indexed by the controller and indexing the field twice is not possible.
// This is to have the webhook tests run independently of the controller.
func setupRoleBindingWebhookForTest(mgr manager.Manager) error {
	err := mgr.GetFieldIndexer().IndexField(context.Background(), &extensionsgreenhousev1alpha1.RoleBinding{}, extensionsgreenhouse.RolebindingRoleRefField, func(rawObj client.Object) []string {
		// Extract the Role name from the RoleBinding Spec, if one is provided
		roleBinding, ok := rawObj.(*extensionsgreenhousev1alpha1.RoleBinding)
		if roleBinding.Spec.RoleRef == "" || !ok {
			return nil
		}
		return []string{roleBinding.Spec.RoleRef}
	})
	Expect(err).ToNot(HaveOccurred(), "there should be no error indexing the rolebindings by roleRef")
	return SetupRoleBindingWebhookWithManager(mgr)
}
