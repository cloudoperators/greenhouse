// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var _ = Describe("ServiceAccount Webhook", func() {
	Describe("validateOwnedByLabelImmutable (unit)", func() {
		It("should allow update when the owned-by label was never set", func() {
			oldSA := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "sa", Namespace: "ns"}}
			newSA := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "sa", Namespace: "ns"}}
			Expect(validateOwnedByLabelImmutable(oldSA, newSA)).To(Succeed())
		})

		It("should allow update when the owned-by label is unchanged", func() {
			oldSA := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sa",
					Namespace: "ns",
					Labels:    map[string]string{greenhouseapis.LabelKeyOwnedBy: "team-a"},
				},
			}
			newSA := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sa",
					Namespace: "ns",
					Labels:    map[string]string{greenhouseapis.LabelKeyOwnedBy: "team-a"},
				},
			}
			Expect(validateOwnedByLabelImmutable(oldSA, newSA)).To(Succeed())
		})

		It("should deny update when the owned-by label is changed to a different value", func() {
			oldSA := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sa",
					Namespace: "ns",
					Labels:    map[string]string{greenhouseapis.LabelKeyOwnedBy: "team-a"},
				},
			}
			newSA := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sa",
					Namespace: "ns",
					Labels:    map[string]string{greenhouseapis.LabelKeyOwnedBy: "team-b"},
				},
			}
			err := validateOwnedByLabelImmutable(oldSA, newSA)
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsForbidden(err)).To(BeTrue(), "expected a Forbidden error")
		})

		It("should deny update when the owned-by label is removed", func() {
			oldSA := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sa",
					Namespace: "ns",
					Labels:    map[string]string{greenhouseapis.LabelKeyOwnedBy: "team-a"},
				},
			}
			newSA := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sa",
					Namespace: "ns",
				},
			}
			err := validateOwnedByLabelImmutable(oldSA, newSA)
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsForbidden(err)).To(BeTrue(), "expected a Forbidden error")
		})

		It("should allow update when owned-by label is added for the first time", func() {
			oldSA := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{Name: "sa", Namespace: "ns"},
			}
			newSA := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sa",
					Namespace: "ns",
					Labels:    map[string]string{greenhouseapis.LabelKeyOwnedBy: "team-a"},
				},
			}
			Expect(validateOwnedByLabelImmutable(oldSA, newSA)).To(Succeed())
		})
	})

	Describe("webhook integration (via API server)", func() {
		It("should allow creating a ServiceAccount with the owned-by label", func() {
			sa := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "webhook-test-sa-",
					Namespace:    test.TestNamespace,
					Labels:       map[string]string{greenhouseapis.LabelKeyOwnedBy: "team-a"},
				},
			}
			Expect(test.K8sClient.Create(test.Ctx, sa)).To(Succeed())
			DeferCleanup(func() {
				Expect(test.K8sClient.Delete(test.Ctx, sa)).To(Succeed())
			})
		})

		It("should allow creating a ServiceAccount without the owned-by label", func() {
			sa := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "webhook-test-sa-",
					Namespace:    test.TestNamespace,
				},
			}
			Expect(test.K8sClient.Create(test.Ctx, sa)).To(Succeed())
			DeferCleanup(func() {
				Expect(test.K8sClient.Delete(test.Ctx, sa)).To(Succeed())
			})
		})

		It("should allow updating a ServiceAccount when the owned-by label is unchanged", func() {
			sa := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "webhook-test-sa-",
					Namespace:    test.TestNamespace,
					Labels:       map[string]string{greenhouseapis.LabelKeyOwnedBy: "team-a"},
				},
			}
			Expect(test.K8sClient.Create(test.Ctx, sa)).To(Succeed())
			DeferCleanup(func() {
				Expect(test.K8sClient.Delete(test.Ctx, sa)).To(Succeed())
			})

			Expect(test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(sa), sa)).To(Succeed())
			// Add an annotation, keep owned-by label the same
			sa.Annotations = map[string]string{"example.com/test": "true"}
			Expect(test.K8sClient.Update(test.Ctx, sa)).To(Succeed())
		})

		It("should deny updating a ServiceAccount when the owned-by label is changed", func() {
			sa := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "webhook-test-sa-",
					Namespace:    test.TestNamespace,
					Labels:       map[string]string{greenhouseapis.LabelKeyOwnedBy: "team-a"},
				},
			}
			Expect(test.K8sClient.Create(test.Ctx, sa)).To(Succeed())
			DeferCleanup(func() {
				Expect(test.K8sClient.Delete(test.Ctx, sa)).To(Succeed())
			})

			Expect(test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(sa), sa)).To(Succeed())
			sa.Labels[greenhouseapis.LabelKeyOwnedBy] = "team-b"
			err := test.K8sClient.Update(test.Ctx, sa)
			Expect(err).To(HaveOccurred(), "changing the owned-by label should be rejected")
			Expect(apierrors.IsForbidden(err)).To(BeTrue(), "expected a Forbidden error")
		})

		It("should deny updating a ServiceAccount when the owned-by label is removed", func() {
			sa := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "webhook-test-sa-",
					Namespace:    test.TestNamespace,
					Labels:       map[string]string{greenhouseapis.LabelKeyOwnedBy: "team-a"},
				},
			}
			Expect(test.K8sClient.Create(test.Ctx, sa)).To(Succeed())
			DeferCleanup(func() {
				Expect(test.K8sClient.Delete(test.Ctx, sa)).To(Succeed())
			})

			Expect(test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(sa), sa)).To(Succeed())
			delete(sa.Labels, greenhouseapis.LabelKeyOwnedBy)
			err := test.K8sClient.Update(test.Ctx, sa)
			Expect(err).To(HaveOccurred(), "removing the owned-by label should be rejected")
			Expect(apierrors.IsForbidden(err)).To(BeTrue(), "expected a Forbidden error")
		})
	})
})
