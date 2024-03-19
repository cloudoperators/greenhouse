// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	extentionsgreenhouse "github.com/cloudoperators/greenhouse/pkg/apis/extensions.greenhouse"
	extensionsgreenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/extensions.greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

var testRole = &extensionsgreenhousev1alpha1.Role{
	TypeMeta: metav1.TypeMeta{
		Kind:       "Role",
		APIVersion: extensionsgreenhousev1alpha1.GroupVersion.String(),
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-role",
		Namespace: test.TestNamespace,
	},
	Spec: extensionsgreenhousev1alpha1.RoleSpec{
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
		},
	},
}

var roleUT *extensionsgreenhousev1alpha1.Role

var testRoleBinding = &extensionsgreenhousev1alpha1.RoleBinding{
	TypeMeta: metav1.TypeMeta{
		Kind:       "RoleBinding",
		APIVersion: extensionsgreenhousev1alpha1.GroupVersion.String(),
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-rolebinding",
		Namespace: test.TestNamespace,
	},
	Spec: extensionsgreenhousev1alpha1.RoleBindingSpec{
		RoleRef:    "test-role",
		TeamRef:    testTeamName,
		Namespaces: []string{test.TestNamespace},
	},
}

var testClusterRoleBinding = &extensionsgreenhousev1alpha1.RoleBinding{
	TypeMeta: metav1.TypeMeta{
		Kind:       "RoleBinding",
		APIVersion: extensionsgreenhousev1alpha1.GroupVersion.String(),
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-clusterrolebinding",
		Namespace: test.TestNamespace,
	},
	Spec: extensionsgreenhousev1alpha1.RoleBindingSpec{
		RoleRef: "test-role",
		TeamRef: testTeamName,
	},
}

var _ = Describe("Validate ClusterRole & RoleBinding on Remote Cluster", Ordered, func() {
	BeforeAll(func() {
		By("creating a cluster")
		Expect(test.K8sClient.Create(test.Ctx, testCluster)).Should(Succeed(), "there should be no error creating the cluster resource")

		// kubeConfigController ensures the namespace within the remote cluster -- we have to create it
		By("creating the namespace on the cluster")
		err := remoteK8sClient.Create(test.Ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: test.TestNamespace}})
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating the namespace")

		By("creating a secret with a valid kubeconfig for a remote cluster")
		testClusterK8sSecret.Data = map[string][]byte{
			greenhouseapis.KubeConfigKey: remoteKubeConfig,
		}
		Expect(test.K8sClient.Create(test.Ctx, testClusterK8sSecret)).Should(Succeed())

		By("creating a team")
		Expect(test.K8sClient.Create(test.Ctx, testTeam)).Should(Succeed())
	})

	BeforeEach(func() {
		// create updates the object with the latest version. deepcopy to avoid failures due to existing resource version
		roleUT = testRole.DeepCopy()
		// create Role on the central cluster"
		err := k8sClient.Create(test.Ctx, roleUT)
		Expect(err).NotTo(HaveOccurred())
	})

	// Delete all RoleBindings and Roles after each test, ensure that Remote Cluster is cleaned up
	// This ensures the deletion of the Remote Resources is working correctly.
	AfterEach(func() {
		// get and delete all RoleBindings on the central cluster
		rbList := &extensionsgreenhousev1alpha1.RoleBindingList{}
		err := k8sClient.List(test.Ctx, rbList)
		Expect(err).NotTo(HaveOccurred())
		for _, rb := range rbList.Items {
			rb := rb
			err := k8sClient.Delete(test.Ctx, &rb)
			Expect(err).NotTo(HaveOccurred())
		}
		Eventually(func() bool {
			err := k8sClient.List(test.Ctx, rbList)
			if err != nil || len(rbList.Items) > 0 {
				return false
			}
			return true
		}).Should(BeTrue(), "there should be no RoleBindings left to list on the central cluster")

		// check that all ClusterRoleBindings are eventually deleted on the remote cluster
		remoteCRBList := &rbacv1.ClusterRoleBindingList{}
		listOpts := []client.ListOption{
			client.HasLabels{extentionsgreenhouse.LabelKeyRoleBinding},
		}
		Eventually(func() bool {
			err := remoteK8sClient.List(test.Ctx, remoteCRBList, listOpts...)
			if err != nil || len(remoteCRBList.Items) > 0 {
				return false
			}
			return true
		}).Should(BeTrue(), "there should be no ClusterRoleBindings left to list on the remote cluster")

		// check that all RoleBindings are eventually deleted on the remote cluster
		remoteRBList := &rbacv1.RoleBindingList{}
		Eventually(func() bool {
			err := remoteK8sClient.List(test.Ctx, remoteRBList, listOpts...)
			if err != nil || len(remoteRBList.Items) > 0 {
				return false
			}
			return true
		}).Should(BeTrue(), "there should be no RoleBindings left to list on the remote cluster")

		// get and delete all Roles on the central cluster
		rList := &extensionsgreenhousev1alpha1.RoleList{}
		err = k8sClient.List(test.Ctx, rList)
		Expect(err).NotTo(HaveOccurred())
		for _, r := range rList.Items {
			r := r
			err := k8sClient.Delete(test.Ctx, &r)
			Expect(err).NotTo(HaveOccurred())
		}
		Eventually(func() bool {
			err := k8sClient.List(test.Ctx, rList)
			if err != nil || len(rList.Items) > 0 {
				return false
			}
			return true
		}).Should(BeTrue(), "there should be no Roles left to list on the central cluster")

		// check that all ClusterRoles are eventually deleted on the remote cluster
		remoteList := &rbacv1.ClusterRoleList{}
		listOpts = []client.ListOption{
			client.HasLabels{extentionsgreenhouse.LabelKeyRole},
		}
		Eventually(func() bool {
			err := remoteK8sClient.List(test.Ctx, remoteList, listOpts...)
			if err != nil || len(remoteList.Items) > 0 {
				return false
			}
			return true
		}).Should(BeTrue(), "there should be no ClusterRoles left to list on the remote cluster")
	})

	Context("When creating a Greenhouse RoleBinding with namespaces on the central cluster", func() {
		It("Should create a ClusterRole and RoleBinding on the remote cluster", func() {
			By("creating a RoleBinding on the central cluster")
			err := k8sClient.Create(test.Ctx, testRoleBinding.DeepCopy())
			Expect(err).NotTo(HaveOccurred())

			By("validating the RoleBinding created on the remote cluster")
			remoteRoleBinding := &rbacv1.RoleBinding{}
			remoteRoleBindingName := types.NamespacedName{
				Name:      testRoleBinding.Name,
				Namespace: testRoleBinding.Namespace,
			}
			Eventually(func() bool {
				err = remoteK8sClient.Get(context.TODO(), remoteRoleBindingName, remoteRoleBinding)
				if err != nil {
					return false
				}
				return !remoteRoleBinding.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the RoleBinding")
			Expect(remoteRoleBinding.RoleRef.Name).To(Equal(testRole.Name))

			By("validating the ClusterRole created on the remote cluster")
			remoteClusterRole := &rbacv1.ClusterRole{}
			remoteClusterRoleName := types.NamespacedName{
				Name: testRole.Name,
			}
			Eventually(func() bool {
				err = remoteK8sClient.Get(test.Ctx, remoteClusterRoleName, remoteClusterRole)
				if err != nil {
					return false
				}
				return !remoteClusterRole.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the ClusterRole")
			Expect(remoteClusterRole.Name).To(Equal(testRole.Name))
			Expect(remoteClusterRole.Rules).To(Equal(testRole.Spec.Rules))
		})
	})

	Context("When creating a Greenhouse RoleBinding without namespaces on the central cluster", func() {
		It("Should create a ClusterRole and ClusterRoleBinding on the remote cluster", func() {
			By("creating a RoleBinding without Namespaces on the central cluster")
			err := k8sClient.Create(test.Ctx, testClusterRoleBinding.DeepCopy())
			Expect(err).NotTo(HaveOccurred())

			By("validating the ClusterRoleBinding created on the remote cluster")
			remoteClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
			remoteRoleBindingName := types.NamespacedName{
				Name:      testClusterRoleBinding.Name,
				Namespace: testClusterRoleBinding.Namespace,
			}
			Eventually(func() bool {
				err = remoteK8sClient.Get(context.TODO(), remoteRoleBindingName, remoteClusterRoleBinding)
				if err != nil {
					return false
				}
				return !remoteClusterRoleBinding.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the ClusterRoleBinding")
			Expect(remoteClusterRoleBinding.RoleRef.Name).To(Equal(testRole.Name))

			By("validating the ClusterRole created on the remote cluster")
			remoteClusterRole := &rbacv1.ClusterRole{}
			remoteClusterRoleName := types.NamespacedName{
				Name: testRole.Name,
			}
			Eventually(func() bool {
				err = remoteK8sClient.Get(test.Ctx, remoteClusterRoleName, remoteClusterRole)
				if err != nil {
					return false
				}
				return !remoteClusterRole.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the ClusterRole")
			Expect(remoteClusterRole.Name).To(Equal(testRole.Name))
			Expect(remoteClusterRole.Rules).To(Equal(testRole.Spec.Rules))
		})
	})

	Context("When creating a Greenhouse RoleBinding with and without namespaces on the central cluster", func() {
		It("Should create a ClusterRole, ClusterRoleBinding and RoleBinding on the remote cluster", func() {
			By("creating a RoleBinding without Namespaces on the central cluster")
			err := k8sClient.Create(test.Ctx, testClusterRoleBinding.DeepCopy())
			Expect(err).NotTo(HaveOccurred())

			By("validating the ClusterRoleBinding created on the remote cluster")
			remoteClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
			remoteClusterRoleBindingName := types.NamespacedName{
				Name:      testClusterRoleBinding.Name,
				Namespace: testClusterRoleBinding.Namespace,
			}
			Eventually(func() bool {
				err = remoteK8sClient.Get(context.TODO(), remoteClusterRoleBindingName, remoteClusterRoleBinding)
				if err != nil {
					return false
				}
				return !remoteClusterRoleBinding.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the ClusterRoleBinding")
			Expect(remoteClusterRoleBinding.RoleRef.Name).To(Equal(testRole.Name))

			By("validating the ClusterRole created on the remote cluster")
			remoteClusterRole := &rbacv1.ClusterRole{}
			remoteClusterRoleName := types.NamespacedName{
				Name: testRole.Name,
			}
			Eventually(func() bool {
				err = remoteK8sClient.Get(test.Ctx, remoteClusterRoleName, remoteClusterRole)
				if err != nil {
					return false
				}
				return !remoteClusterRole.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the ClusterRole")
			Expect(remoteClusterRole.Name).To(Equal(testRole.Name))
			Expect(remoteClusterRole.Rules).To(Equal(testRole.Spec.Rules))

			By("creating a RoleBinding on the central cluster")
			err = k8sClient.Create(test.Ctx, testRoleBinding.DeepCopy())
			Expect(err).NotTo(HaveOccurred())

			By("validating the RoleBinding created on the remote cluster")
			remoteRoleBinding := &rbacv1.RoleBinding{}
			remoteRoleBindingName := types.NamespacedName{
				Name:      testRoleBinding.Name,
				Namespace: testRoleBinding.Namespace,
			}
			Eventually(func() bool {
				err = remoteK8sClient.Get(context.TODO(), remoteRoleBindingName, remoteRoleBinding)
				if err != nil {
					return false
				}
				return !remoteRoleBinding.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the RoleBinding")
			Expect(remoteRoleBinding.RoleRef.Name).To(Equal(testRole.Name))
		})
	})

	Context("When updating Greenhouse Role & RoleBinding w/wo Namespaces on the central cluster", func() {
		It("Should reconcile the ClusterRole, ClusterRoleBinding and RoleBinding on the remote cluster", func() {
			clusterRoleBindingUT := testClusterRoleBinding.DeepCopy()
			By("creating a RoleBinding without Namespaces on the central cluster")
			err := k8sClient.Create(test.Ctx, clusterRoleBindingUT)
			Expect(err).NotTo(HaveOccurred())

			By("validating the ClusterRoleBinding created on the remote cluster")
			remoteClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
			remoteClusterRoleBindingName := types.NamespacedName{
				Name:      testClusterRoleBinding.Name,
				Namespace: testClusterRoleBinding.Namespace,
			}
			Eventually(func() bool {
				err = remoteK8sClient.Get(context.TODO(), remoteClusterRoleBindingName, remoteClusterRoleBinding)
				if err != nil {
					return false
				}
				return !remoteClusterRoleBinding.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the ClusterRoleBinding")
			Expect(remoteClusterRoleBinding.RoleRef.Name).To(Equal(testRole.Name))

			By("validating the ClusterRole created on the remote cluster")
			remoteClusterRole := &rbacv1.ClusterRole{}
			remoteClusterRoleName := types.NamespacedName{
				Name: testRole.Name,
			}
			Eventually(func() bool {
				err = remoteK8sClient.Get(test.Ctx, remoteClusterRoleName, remoteClusterRole)
				if err != nil {
					return false
				}
				return !remoteClusterRole.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the ClusterRole")
			Expect(remoteClusterRole.Name).To(Equal(testRole.Name))
			Expect(remoteClusterRole.Rules).To(Equal(testRole.Spec.Rules))

			By("creating a RoleBinding on the central cluster")
			roleBindingUT := testRoleBinding.DeepCopy()
			err = k8sClient.Create(test.Ctx, roleBindingUT)
			Expect(err).NotTo(HaveOccurred())

			By("validating the RoleBinding created on the remote cluster")
			remoteRoleBinding := &rbacv1.RoleBinding{}
			remoteRoleBindingName := types.NamespacedName{
				Name:      testRoleBinding.Name,
				Namespace: testRoleBinding.Namespace,
			}
			Eventually(func() bool {
				err = remoteK8sClient.Get(context.TODO(), remoteRoleBindingName, remoteRoleBinding)
				if err != nil {
					return false
				}
				return !remoteRoleBinding.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the RoleBinding")
			Expect(remoteRoleBinding.RoleRef.Name).To(Equal(testRole.Name))

			By("updating the Greenhouse Role on the central cluster")
			roleUT.Spec.Rules[0].Verbs = []string{"get"}
			err = k8sClient.Update(test.Ctx, roleUT)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func(g Gomega) bool {
				err = remoteK8sClient.Get(test.Ctx, remoteClusterRoleName, remoteClusterRole)
				if err != nil {
					return false
				}
				return g.Expect(remoteClusterRole.Rules).To(Equal(roleUT.Spec.Rules))
			}).Should(BeTrue(), "there should be no error getting the ClusterRole from the remote cluster")
		})
	})
	Context("When tampering with a RoleBinding on the Remote Cluster", func() {
		It("should reconcile the remote RoleBinding", func() {
			By("creating a RoleBinding without Namespaces on the central cluster")
			roleBindingUT := testClusterRoleBinding.DeepCopy()
			err := k8sClient.Create(test.Ctx, roleBindingUT)
			Expect(err).NotTo(HaveOccurred())

			By("validating the ClusterRoleBinding created on the remote cluster")
			remoteClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
			remoteClusterRoleBindingName := types.NamespacedName{
				Name:      roleBindingUT.Name,
				Namespace: roleBindingUT.Namespace,
			}
			Eventually(func() bool {
				err = remoteK8sClient.Get(context.TODO(), remoteClusterRoleBindingName, remoteClusterRoleBinding)
				if err != nil {
					return false
				}
				return !remoteClusterRoleBinding.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the ClusterRoleBinding")
			Expect(remoteClusterRoleBinding.RoleRef.Name).To(Equal(testRole.Name))

			By("validating the ClusterRole created on the remote cluster")
			remoteClusterRole := &rbacv1.ClusterRole{}
			remoteClusterRoleName := types.NamespacedName{
				Name: testRole.Name,
			}
			Eventually(func() bool {
				err = remoteK8sClient.Get(test.Ctx, remoteClusterRoleName, remoteClusterRole)
				if err != nil {
					return false
				}
				return !remoteClusterRole.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the ClusterRole")
			Expect(remoteClusterRole.Name).To(Equal(testRole.Name))
			Expect(remoteClusterRole.Rules).To(Equal(testRole.Spec.Rules))

			By("altering the RoleBinding on the remote cluster")
			err = remoteK8sClient.Get(test.Ctx, remoteClusterRoleBindingName, remoteClusterRoleBinding)
			Expect(err).NotTo(HaveOccurred(), "there should be no error getting the ClusterRoleBinding from the remote cluster")
			expected := remoteClusterRoleBinding.DeepCopy().Subjects
			remoteClusterRoleBinding.Subjects = append(remoteClusterRoleBinding.Subjects, rbacv1.Subject{Kind: "User", Name: "foobar", APIGroup: "rbac.authorization.k8s.io"})
			err = remoteK8sClient.Update(test.Ctx, remoteClusterRoleBinding)
			Expect(err).NotTo(HaveOccurred(), "there should be no error updating the ClusterRoleBinding on the remote cluster")

			By("triggering the reconcile of the central cluster RoleBinding with a noop update")
			err = k8sClient.Get(test.Ctx, types.NamespacedName{Name: roleBindingUT.Name, Namespace: roleBindingUT.Namespace}, roleBindingUT)
			Expect(err).NotTo(HaveOccurred(), "there should be no error getting the RoleBinding from the central cluster")
			// changing the labels to trigger the reconciliation in this test.
			roleBindingUT.SetLabels(map[string]string{"foo": "bar"})
			err = k8sClient.Update(test.Ctx, roleBindingUT)
			Expect(err).NotTo(HaveOccurred(), "there should be no error updating the RoleBinding on the central cluster")

			Eventually(func(g Gomega) bool {
				err = remoteK8sClient.Get(test.Ctx, remoteClusterRoleBindingName, remoteClusterRoleBinding)
				if err != nil {
					return false
				}
				return g.Expect(remoteClusterRoleBinding.Subjects).To(Equal(expected))
			}).Should(BeTrue(), "the remote RoleBinding should eventually be reconciled")
		})
	})
})
