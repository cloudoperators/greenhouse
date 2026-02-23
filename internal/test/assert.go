// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

// EventuallyDeleted deletes the object and waits until it is gone. Early return if the delete fails with NotFound
func EventuallyDeleted(ctx context.Context, c client.Client, obj client.Object) {
	GinkgoHelper()

	// Prepare object for deletion
	cluster, ok := obj.(*greenhousev1alpha1.Cluster)
	if ok {
		UpdateClusterWithDeletionAnnotation(ctx, c, cluster)
	}
	pluginPreset, ok := obj.(*greenhousev1alpha1.PluginPreset)
	if ok {
		MustRemoveAnnotation(ctx, c, pluginPreset, greenhousev1alpha1.PreventDeletionAnnotation)
	}

	// Retry delete on conflict - the object may have been modified by controllers
	Eventually(func(g Gomega) {
		err := c.Get(ctx, client.ObjectKeyFromObject(obj), obj)
		if apierrors.IsNotFound(err) {
			return // Already deleted
		}
		g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the object for deletion")
		err = c.Delete(ctx, obj)
		if apierrors.IsNotFound(err) {
			return // Deleted between Get and Delete
		}
		g.Expect(err).ToNot(HaveOccurred(), "there should be no error deleting the object")
	}).Should(Succeed(), "deletion should succeed or object should already be deleted")

	// Wait for object to be gone (with extended timeout for objects with finalizers)
	// Use longer polling interval to reduce conflicts with controller status updates
	Eventually(func() bool {
		err := c.Get(ctx, client.ObjectKeyFromObject(obj), obj)
		return apierrors.IsNotFound(err)
	}).WithTimeout(2*time.Minute).WithPolling(500*time.Millisecond).Should(BeTrue(), "the object should be deleted eventually")
}

// EventuallyCreated verifies if the object is created
func EventuallyCreated(ctx context.Context, c client.Client, obj client.Object) {
	GinkgoHelper()
	Eventually(func() bool {
		return c.Get(ctx, client.ObjectKeyFromObject(obj), obj) == nil
	}).Should(BeTrue(), "there should be no error getting the object")
}
