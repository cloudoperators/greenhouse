// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// Package informers provides a small wrapper over controller-runtime's
// cache.Cache informers so callers register typed event handlers and wait for
// sync without repeating the boilerplate.
package informers

import (
	"context"
	"errors"
	"fmt"

	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EventHandlers holds the callbacks invoked on informer events. Any field may
// be nil to ignore that event.
type EventHandlers struct {
	AddFunc    func(obj any)
	UpdateFunc func(oldObj, newObj any)
	DeleteFunc func(obj any)
}

// GenericInformer wraps a single cache.Informer.
type GenericInformer struct {
	informer cache.Informer
}

// New resolves the informer for obj from the cache and registers handlers.
// The cache must be started separately (cache.Start); this only wires handlers.
func New(ctx context.Context, c cache.Cache, obj client.Object, handlers EventHandlers) (*GenericInformer, error) {
	informer, err := c.GetInformer(ctx, obj)
	if err != nil {
		return nil, fmt.Errorf("failed to get informer for %T: %w", obj, err)
	}
	if _, err := informer.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
		AddFunc:    handlers.AddFunc,
		UpdateFunc: handlers.UpdateFunc,
		DeleteFunc: handlers.DeleteFunc,
	}); err != nil {
		return nil, fmt.Errorf("failed to add event handler for %T: %w", obj, err)
	}
	return &GenericInformer{informer: informer}, nil
}

// WaitForSync blocks until the informer has populated or ctx is cancelled.
func (g *GenericInformer) WaitForSync(ctx context.Context) error {
	if !toolscache.WaitForCacheSync(ctx.Done(), g.informer.HasSynced) {
		return errors.New("timed out waiting for cache to sync")
	}
	return nil
}

// HasSynced reports whether the informer has completed its initial list.
func (g *GenericInformer) HasSynced() bool {
	return g.informer.HasSynced()
}
