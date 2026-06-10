// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var _ = Describe("TokenRequest Webhook", func() {
	ctxWithRequest := func(saName, saNamespace string) admission.Request {
		return admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				Name:      saName,
				Namespace: saNamespace,
			},
		}
	}

	makeTokenRequest := func(expSeconds *int64) *authenticationv1.TokenRequest {
		return &authenticationv1.TokenRequest{
			Spec: authenticationv1.TokenRequestSpec{
				ExpirationSeconds: expSeconds,
			},
		}
	}

	ptr := func(i int64) *int64 { return &i }

	Describe("defaultTokenRequest (unit)", func() {
		It("should skip pod-bound token requests", func() {
			sa := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "team-a-sa",
					Namespace: "test-org",
					Labels:    map[string]string{greenhouseapis.LabelKeyOwnedBy: "team-a"},
				},
			}
			fakeClient := fake.NewClientBuilder().WithScheme(test.GreenhouseV1Alpha1Scheme()).WithObjects(sa).Build()

			tr := makeTokenRequest(ptr(99999999))
			tr.Spec.BoundObjectRef = &authenticationv1.BoundObjectReference{Name: "some-pod"}

			ctx := admission.NewContextWithRequest(test.Ctx, ctxWithRequest(sa.Name, sa.Namespace))
			Expect(defaultTokenRequest(ctx, fakeClient, tr)).To(Succeed())
			Expect(*tr.Spec.ExpirationSeconds).To(Equal(int64(99999999)), "pod-bound token should not be modified")
		})

		It("should skip token requests for SAs without the owned-by label", func() {
			sa := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{Name: "unmanaged-sa", Namespace: "test-org"},
			}
			fakeClient := fake.NewClientBuilder().WithScheme(test.GreenhouseV1Alpha1Scheme()).WithObjects(sa).Build()

			tr := makeTokenRequest(ptr(99999999))
			ctx := admission.NewContextWithRequest(test.Ctx, ctxWithRequest(sa.Name, sa.Namespace))
			Expect(defaultTokenRequest(ctx, fakeClient, tr)).To(Succeed())
			Expect(*tr.Spec.ExpirationSeconds).To(Equal(int64(99999999)), "unmanaged SA token should not be modified")
		})

		It("should cap expiration when it exceeds 90 days", func() {
			sa := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "team-a-sa",
					Namespace: "test-org",
					Labels:    map[string]string{greenhouseapis.LabelKeyOwnedBy: "team-a"},
				},
			}
			fakeClient := fake.NewClientBuilder().WithScheme(test.GreenhouseV1Alpha1Scheme()).WithObjects(sa).Build()

			tr := makeTokenRequest(ptr(99999999))
			ctx := admission.NewContextWithRequest(test.Ctx, ctxWithRequest(sa.Name, sa.Namespace))
			Expect(defaultTokenRequest(ctx, fakeClient, tr)).To(Succeed())
			Expect(*tr.Spec.ExpirationSeconds).To(Equal(maxTokenExpirationSeconds))
		})

		It("should cap expiration when it is nil", func() {
			sa := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "team-a-sa",
					Namespace: "test-org",
					Labels:    map[string]string{greenhouseapis.LabelKeyOwnedBy: "team-a"},
				},
			}
			fakeClient := fake.NewClientBuilder().WithScheme(test.GreenhouseV1Alpha1Scheme()).WithObjects(sa).Build()

			tr := makeTokenRequest(nil)
			ctx := admission.NewContextWithRequest(test.Ctx, ctxWithRequest(sa.Name, sa.Namespace))
			Expect(defaultTokenRequest(ctx, fakeClient, tr)).To(Succeed())
			Expect(tr.Spec.ExpirationSeconds).NotTo(BeNil())
			Expect(*tr.Spec.ExpirationSeconds).To(Equal(maxTokenExpirationSeconds))
		})

		It("should leave expiration unchanged when it is within 90 days", func() {
			sa := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "team-a-sa",
					Namespace: "test-org",
					Labels:    map[string]string{greenhouseapis.LabelKeyOwnedBy: "team-a"},
				},
			}
			fakeClient := fake.NewClientBuilder().WithScheme(test.GreenhouseV1Alpha1Scheme()).WithObjects(sa).Build()

			oneHour := int64(3600)
			tr := makeTokenRequest(&oneHour)
			ctx := admission.NewContextWithRequest(test.Ctx, ctxWithRequest(sa.Name, sa.Namespace))
			Expect(defaultTokenRequest(ctx, fakeClient, tr)).To(Succeed())
			Expect(*tr.Spec.ExpirationSeconds).To(Equal(oneHour))
		})

		It("should leave expiration unchanged when it is exactly 90 days", func() {
			sa := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "team-a-sa",
					Namespace: "test-org",
					Labels:    map[string]string{greenhouseapis.LabelKeyOwnedBy: "team-a"},
				},
			}
			fakeClient := fake.NewClientBuilder().WithScheme(test.GreenhouseV1Alpha1Scheme()).WithObjects(sa).Build()

			tr := makeTokenRequest(ptr(maxTokenExpirationSeconds))
			ctx := admission.NewContextWithRequest(test.Ctx, ctxWithRequest(sa.Name, sa.Namespace))
			Expect(defaultTokenRequest(ctx, fakeClient, tr)).To(Succeed())
			Expect(*tr.Spec.ExpirationSeconds).To(Equal(maxTokenExpirationSeconds))
		})
	})
})
