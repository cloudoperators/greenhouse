// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package flux

import (
	"errors"
	"time"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	fluxkust "github.com/fluxcd/pkg/apis/kustomize"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

// greenhouse ready status condition cel expression
const defaultRetryInterval = 10 * time.Second

// TODO: health check expressions for Greenhouse PluginDefinition resources (remove this)
const (
	readyConditionExpr = `has(status.statusConditions) && status.statusConditions.conditions(c, c.type == 'Ready').all(c, c.status == 'True')`
	failConditionExpr  = `has(status.statusConditions) && status.statusConditions.conditions(c, c.type == 'Ready').any(c, c.status == 'False')`
)

type KustomizeBuilder struct {
	log  logr.Logger
	spec kustomizev1.KustomizationSpec
}

func NewKustomizationSpecBuilder(logger logr.Logger) *KustomizeBuilder {
	return &KustomizeBuilder{
		log: logger.WithName("kustomization-builder"),
		spec: kustomizev1.KustomizationSpec{
			SourceRef: kustomizev1.CrossNamespaceSourceReference{},
		},
	}
}

func (k *KustomizeBuilder) WithCommonMetadata(annotations, labels map[string]string) *KustomizeBuilder {
	metadata := &kustomizev1.CommonMetadata{}
	if annotations != nil {
		metadata.Annotations = annotations
	}
	if labels != nil {
		metadata.Labels = labels
	}
	k.spec.CommonMetadata = metadata
	return k
}
func (k *KustomizeBuilder) WithDependsOn(deps []fluxmeta.NamespacedObjectReference) *KustomizeBuilder {
	if len(deps) > 0 {
		k.spec.DependsOn = deps
	}
	return k
}

func (k *KustomizeBuilder) WithInterval(duration metav1.Duration) *KustomizeBuilder {
	k.spec.Interval = duration
	return k
}

func (k *KustomizeBuilder) WithRetryInterval(duration *metav1.Duration) *KustomizeBuilder {
	if duration != nil {
		k.spec.RetryInterval = duration
	} else {
		k.spec.RetryInterval = &metav1.Duration{Duration: defaultRetryInterval}
	}
	return k
}

func (k *KustomizeBuilder) WithPath(path string) *KustomizeBuilder {
	if path != "" {
		k.spec.Path = path
	}
	return k
}

func (k *KustomizeBuilder) WithServiceAccountName(serviceAccountName string) *KustomizeBuilder {
	if serviceAccountName != "" {
		k.spec.ServiceAccountName = serviceAccountName
	}
	return k
}

func (k *KustomizeBuilder) WithSourceRef(apiVersion, kind, name, namespace string) *KustomizeBuilder {
	ref := kustomizev1.CrossNamespaceSourceReference{
		APIVersion: apiVersion,
		Kind:       kind,
		Name:       name,
		Namespace:  namespace,
	}
	k.spec.SourceRef = ref
	return k
}

// WithSuspend - sets the suspend flag, if set flux will not reconcile the Kustomization resource
func (k *KustomizeBuilder) WithSuspend(suspend bool) *KustomizeBuilder {
	k.spec.Suspend = suspend
	return k
}

// WithTargetNamespace - sets the target namespace on kustomize resources
func (k *KustomizeBuilder) WithTargetNamespace(namespace string) *KustomizeBuilder {
	if namespace != "" {
		k.spec.TargetNamespace = namespace
	}
	return k
}

func (k *KustomizeBuilder) WithTimeout(duration *metav1.Duration) *KustomizeBuilder {
	if duration != nil {
		k.spec.Timeout = duration
	} else {
		k.spec.Timeout = &metav1.Duration{Duration: DefaultTimeout} // default timeout
	}
	return k
}

// WithPrune - when set to true, flux will prune resources that are no longer defined in the Kustomization
// immediate garbage collection
func (k *KustomizeBuilder) WithPrune(prune bool) *KustomizeBuilder {
	k.spec.Prune = prune
	return k
}

// WithForce - when set to true combined with Prune=true, flux will replace previous applied resources with the new ones
func (k *KustomizeBuilder) WithForce(force bool) *KustomizeBuilder {
	k.spec.Force = force
	return k
}

// WithWait - when combined with health checks / health check expressions, flux will wait for the resource to be ready before marking the Kustomization as ready
func (k *KustomizeBuilder) WithWait(wait bool) *KustomizeBuilder {
	k.spec.Wait = wait
	return k
}

// WithGreenhouseReadyHealthExpression - sets the health check expressions for a Greenhouse resource
// TODO: remove this
func (k *KustomizeBuilder) WithGreenhouseReadyHealthExpression(version, kind, name, namespace string) *KustomizeBuilder {
	k.spec.HealthChecks = []fluxmeta.NamespacedObjectKindReference{
		{
			APIVersion: version,
			Kind:       kind,
			Name:       name,
			Namespace:  namespace,
		},
	}
	k.spec.HealthCheckExprs = []fluxkust.CustomHealthCheck{
		{
			APIVersion: greenhouseapisv1alpha1.GroupVersion.String(),
			Kind:       kind,
			HealthCheckExpressions: fluxkust.HealthCheckExpressions{
				Current: readyConditionExpr,
				Failed:  failConditionExpr,
			},
		},
	}
	return k
}

func (k *KustomizeBuilder) Build() (kustomizev1.KustomizationSpec, error) {
	if k.spec.SourceRef.Kind == "" {
		return kustomizev1.KustomizationSpec{}, errors.New("source reference kind is required")
	}
	if k.spec.SourceRef.Name == "" {
		return kustomizev1.KustomizationSpec{}, errors.New("source reference name is required")
	}
	if k.spec.Interval.Duration <= 0 {
		return kustomizev1.KustomizationSpec{}, errors.New("interval must be greater than zero")
	}
	return k.spec, nil
}
