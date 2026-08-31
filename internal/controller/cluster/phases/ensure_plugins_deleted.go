// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

func (p *Phase) ensurePluginsDeleted(cluster *greenhousev1alpha1.Cluster) lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		pluginList := &greenhousev1alpha1.PluginList{}
		if err := p.Client.List(ctx, pluginList,
			client.InNamespace(cluster.GetNamespace()),
			client.MatchingLabels{greenhouseapis.LabelKeyCluster: cluster.GetName()},
		); err != nil {
			return lifecycle.Break(), err
		}

		if strings.EqualFold(cluster.Annotations[greenhouseapis.AnnotationKeyDeletionPolicy], greenhouseapis.DeletionPolicyRetain) {
			updatedCount := 0
			for _, plugin := range pluginList.Items {
				if plugin.Annotations[greenhouseapis.AnnotationKeyDeletionPolicy] == greenhouseapis.DeletionPolicyRetain {
					continue
				}
				result, err := controllerutil.CreateOrUpdate(ctx, p.Client, &plugin, func() error {
					if plugin.Annotations == nil {
						plugin.Annotations = make(map[string]string)
					}
					plugin.Annotations[greenhouseapis.AnnotationKeyDeletionPolicy] = greenhouseapis.DeletionPolicyRetain
					return nil
				})
				if client.IgnoreNotFound(err) != nil {
					return lifecycle.Break(), err
				}
				if result == controllerutil.OperationResultUpdated {
					updatedCount++
				}
			}
			if updatedCount > 0 {
				return lifecycle.Requeue(), nil
			}
		}

		deletedCount := 0
		for _, plugin := range pluginList.Items {
			if err := p.Client.Delete(ctx, &plugin); client.IgnoreNotFound(err) != nil {
				return lifecycle.Break(), err
			}
			deletedCount++
		}
		if deletedCount > 0 {
			return lifecycle.Requeue(), nil
		}
		return lifecycle.Continue(), nil
	}
}
