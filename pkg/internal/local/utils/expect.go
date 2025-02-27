// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"errors"
	"time"

	"github.com/cenkalti/backoff/v5"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	MaxRetries = 10
)

func StandardBackoff() *backoff.ExponentialBackOff {
	b := &backoff.ExponentialBackOff{
		InitialInterval:     500 * time.Millisecond, // Start with 500ms delay
		RandomizationFactor: 0.5,                    // Randomize interval by ±50%
		Multiplier:          2.0,                    // Double the interval each time
		MaxInterval:         30 * time.Second,       // Cap at 30s between retries
	}
	return b
}

// WaitUntilSecretCreated - waits until a secret is created in the given namespace with a backoff strategy
func WaitUntilSecretCreated(ctx context.Context, k8sClient client.Client, name, namespace string) error {
	b := StandardBackoff()
	b.Reset()
	retires := 0
	op := func() (op bool, err error) {
		if retires > MaxRetries {
			err = backoff.Permanent(errors.New("max retries reached"))
			return
		}
		Logf("waiting for secret %s to be created...", name)
		secret := &v1.Secret{}
		err = k8sClient.Get(ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}, secret)
		if err != nil {
			retires++
			return
		}
		op = true
		return
	}
	_, err := backoff.Retry(ctx, op, backoff.WithBackOff(b))
	return err
}

// WaitUntilJobSucceeds - waits until a job succeeds in the given namespace with a backoff strategy
func WaitUntilJobSucceeds(ctx context.Context, k8sClient client.Client, name, namespace string) error {
	b := StandardBackoff()
	b.Reset()
	retires := 0
	op := func() (op bool, err error) {
		if retires > MaxRetries {
			err = backoff.Permanent(errors.New("max retries reached"))
			return
		}
		Logf("waiting for job %s to succeed...", name)
		job := &batchv1.Job{}
		err = k8sClient.Get(ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}, job)
		if err != nil {
			retires++
			return
		}
		if job.Status.Failed > 0 {
			err = errors.New("job failed")
			return
		}
		if job.Status.Succeeded == 0 {
			retires++
			return
		}
		op = true
		return
	}
	_, err := backoff.Retry(ctx, op, backoff.WithBackOff(b))
	return err
}
