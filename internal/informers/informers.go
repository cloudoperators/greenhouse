// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package informers

import (
	"context"
	"errors"
	"fmt"

	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type EventHandlers struct {
	AddFunc    func(obj any)
	UpdateFunc func(oldObj, newObj any)
	DeleteFunc func(obj any)
}

type GenericInformer struct {
	informer cache.Informer
}

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

func (g *GenericInformer) WaitForSync(ctx context.Context) error {
	if !toolscache.WaitForCacheSync(ctx.Done(), g.informer.HasSynced) {
		return errors.New("timed out waiting for cache to sync")
	}
	return nil
}

func (g *GenericInformer) HasSynced() bool {
	return g.informer.HasSynced()
}
