// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"errors"
	"github.com/cenkalti/backoff/v4"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var DefaultElapsedTime = 30 * time.Second

// WaitUntilSecretCreated - waits until a secret is created in the given namespace with a backoff strategy
func WaitUntilSecretCreated(ctx context.Context, k8sClient client.Client, name, namespace string) error {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 5 * time.Second
	b.MaxElapsedTime = DefaultElapsedTime
	return backoff.Retry(func() error {
		Logf("waiting for secret %s to be created...", name)
		secret := &v1.Secret{}
		err := k8sClient.Get(ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}, secret)
		if err != nil {
			return err
		}
		return nil
	}, b)
}

// WaitUntilJobSucceeds - waits until a job succeeds in the given namespace with a backoff strategy
func WaitUntilJobSucceeds(ctx context.Context, k8sClient client.Client, name, namespace string) error {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 5 * time.Second
	b.MaxElapsedTime = DefaultElapsedTime
	return backoff.Retry(func() error {
		Logf("waiting for job %s to succeed...", name)
		job := &batchv1.Job{}
		err := k8sClient.Get(ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}, job)
		if err != nil {
			return err
		}
		if job.Status.Failed > 0 {
			return errors.New("job failed")
		}

		if job.Status.Succeeded == 0 {
			return errors.New("job is not yet ready")
		}
		return nil
	}, b)
}
