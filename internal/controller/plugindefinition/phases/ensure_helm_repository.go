// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"
	"time"

	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudoperators/greenhouse/internal/flux"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

func (p *Phase) ensureHelmRepository() lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		pluginDefSpec := p.PluginDef.GetPluginDefinitionSpec()
		repositoryURL := pluginDefSpec.HelmChart.Repository
		helmRepository := &sourcev1.HelmRepository{}
		helmRepository.SetName(flux.ChartURLToName(repositoryURL))
		helmRepository.SetNamespace(p.NamespaceName)

		result, err := controllerutil.CreateOrUpdate(ctx, p.Client, helmRepository, func() error {
			helmRepository.Spec.Type = flux.GetSourceRepositoryType(repositoryURL)
			helmRepository.Spec.Interval = metav1.Duration{Duration: 24 * time.Hour}
			helmRepository.Spec.URL = repositoryURL
			if flux.CheckIfLocalRegistry(repositoryURL) {
				helmRepository.Spec.Insecure = true
			}
			return controllerutil.SetOwnerReference(p.PluginDef, helmRepository, p.Client.Scheme())
		})
		if err != nil {
			log.FromContext(ctx).Error(err, "Failed to create or update HelmRepository", "namespace", p.NamespaceName, "name", helmRepository.Name)
			return lifecycle.Break(), err
		}
		switch result {
		case controllerutil.OperationResultCreated:
			log.FromContext(ctx).Info("Created helmRepository", "namespace", p.NamespaceName, "name", helmRepository.Name)
			p.Recorder.Eventf(p.PluginDef, helmRepository, corev1.EventTypeNormal, "Created", "reconciling (Cluster-)PluginDefinition", "Created HelmRepository %s", helmRepository.Name)
		case controllerutil.OperationResultUpdated:
			log.FromContext(ctx).Info("Updated helmRepository", "namespace", p.NamespaceName, "name", helmRepository.Name)
			p.Recorder.Eventf(p.PluginDef, helmRepository, corev1.EventTypeNormal, "Updated", "reconciling (Cluster-)PluginDefinition", "Updated HelmRepository %s", helmRepository.Name)
		case controllerutil.OperationResultNone:
			log.FromContext(ctx).Info("No changes to helmRepository", "namespace", p.NamespaceName, "name", helmRepository.Name)
		}
		p.helmRepo = helmRepository
		return lifecycle.Continue(), nil
	}
}
