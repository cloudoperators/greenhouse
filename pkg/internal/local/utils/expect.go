// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"errors"
	"fmt"
	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
	"gopkg.in/yaml.v3"
	"time"

	"github.com/cenkalti/backoff/v4"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var DefaultElapsedTime = 30 * time.Second

// WaitUntilSecretCreated - waits until a secret is created in the given namespace with a backoff strategy
func WaitUntilSecretCreated(ctx context.Context, k8sClient client.Client, name, namespace string) error {
	b := backoff.NewExponentialBackOff(backoff.WithInitialInterval(5*time.Second), backoff.WithMaxElapsedTime(DefaultElapsedTime))
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
	b := backoff.NewExponentialBackOff(backoff.WithInitialInterval(5*time.Second), backoff.WithMaxElapsedTime(DefaultElapsedTime))
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

func WaitUntilNamespaceCreated(ctx context.Context, k8sClient client.Client, name string) error {
	b := backoff.NewExponentialBackOff(backoff.WithInitialInterval(5*time.Second), backoff.WithMaxElapsedTime(DefaultElapsedTime))
	return backoff.Retry(func() error {
		Logf("waiting for namespace %s to be created...", name)
		ns := &v1.Namespace{}
		err := k8sClient.Get(ctx, types.NamespacedName{
			Name: name,
		}, ns)
		if err != nil {
			return err
		}
		if ns.Status.Phase != v1.NamespaceActive {
			return errors.New("namespace is not yet ready")
		}
		return nil
	}, b)
}

// WaitUntilGreenhouseResourcesAreReady will fail the test in case not all resources of the workflow result in Ready state
func WaitUntilGreenhouseResourcesAreReady(ctx context.Context, k8sClient client.Client, namespace string, resources ...lifecycle.RuntimeObject) error {
	b := backoff.NewExponentialBackOff(backoff.WithInitialInterval(5*time.Second), backoff.WithMaxElapsedTime(DefaultElapsedTime))
	err := backoff.Retry(func() error {
		Log("waiting for resource to be created...")
		return waitUntilReady(ctx, k8sClient, namespace, resources...)
	}, b)
	if err != nil {
		var resourceStatus []error
		resourceStatus = append(resourceStatus, err)
		for _, resource := range resources {
			resource.SetManagedFields(nil)
			// transform resource to YAML
			data, _ := yaml.Marshal(resource)
			resourceStatus = append(resourceStatus, fmt.Errorf("resource: %s\n%s", resource.GetName(), string(data)))
		}
		err = errors.Join(resourceStatus...)
	}
	return err
}

func waitUntilReady(ctx context.Context, apiClient client.Client, namespace string, resources ...lifecycle.RuntimeObject) error {
	for _, resource := range resources {
		resource.SetNamespace(namespace)
		err := apiClient.Get(ctx, client.ObjectKeyFromObject(resource), resource)
		if err != nil {
			return err
		}
		conditions := resource.GetConditions()
		readyCondition := conditions.GetConditionByType(greenhouseapisv1alpha1.ReadyCondition)
		if !readyCondition.IsTrue() {
			return fmt.Errorf("resource %s is not yet ready", resource.GetName())
		}
	}
	return nil
}
