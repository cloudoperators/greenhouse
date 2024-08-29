// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EventuallyDeleted deletes the object and waits until it is gone. Early return if the delete fails with NotFound
func EventuallyDeleted(ctx context.Context, c client.Client, obj client.Object) {
	GinkgoHelper()
	err := c.Delete(ctx, obj)
	if apierrors.IsNotFound(err) {
		return
	}
	Expect(err).Should(Succeed(), "there should be no error deleting the object")
	Eventually(func() bool {
		return apierrors.IsNotFound(c.Get(ctx, client.ObjectKeyFromObject(obj), obj))
	}).Should(BeTrue(), "there should be no error deleting the object")
}

// EventuallyGet gets the object and retries until it is available.
func EventuallyCreated(ctx context.Context, c client.Client, obj client.Object) {
	GinkgoHelper()
	Eventually(func() bool {
		return c.Get(ctx, client.ObjectKeyFromObject(obj), obj) == nil
	}).Should(BeTrue(), "there should be no error getting the object")
}
