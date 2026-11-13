// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization

import (
	"context"
	"slices"
	"time"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

const defaultCleanupRequeueInterval = 60 * time.Second

type offboardingPhase struct {
	*OrganizationReconciler
	org             *greenhousev1alpha1.Organization
	groups          []string
	requeueInterval time.Duration
}

func (p *offboardingPhase) ensureDeletePhases() []lifecycle.SubRoutine {
	return []lifecycle.SubRoutine{
		p.ensureForeignResourcesDeleted(),
		p.ensureResourcesDeleted(&greenhousev1alpha2.TeamRoleBindingList{}),
		p.ensureClustersDeleted(),
		p.ensureResourcesDeleted(&greenhousev1alpha1.PluginPresetList{}),
		p.ensureResourcesDeleted(&greenhousev1alpha1.PluginList{}),
		p.ensureDexResourcesDeleted(),
		p.ensureNamespaceOrphaned(),
	}
}

func (p *offboardingPhase) ensureForeignResourcesDeleted() lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		crds := &apiextensionsv1.CustomResourceDefinitionList{}
		if err := p.apiReader.List(ctx, crds); err != nil {
			return lifecycle.Break(), err
		}
		return p.deleteAll(ctx, foreignResourceLists(crds, p.groups)...)
	}
}

func foreignResourceLists(crds *apiextensionsv1.CustomResourceDefinitionList, groups []string) []client.ObjectList {
	var lists []client.ObjectList
	for _, crd := range crds.Items {
		if crd.Spec.Scope != apiextensionsv1.NamespaceScoped || !slices.Contains(groups, crd.Spec.Group) {
			continue
		}
		for _, version := range crd.Spec.Versions {
			if !version.Served {
				continue
			}
			list := &unstructured.UnstructuredList{}
			list.SetGroupVersionKind(schema.GroupVersionKind{Group: crd.Spec.Group, Version: version.Name, Kind: crd.Spec.Names.ListKind})
			lists = append(lists, list)
			break
		}
	}
	return lists
}

func (p *offboardingPhase) ensureResourcesDeleted(list client.ObjectList) lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		return p.deleteAll(ctx, list)
	}
}

func (p *offboardingPhase) ensureClustersDeleted() lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		clusters := &greenhousev1alpha1.ClusterList{}
		if err := p.List(ctx, clusters, client.InNamespace(p.org.Name)); err != nil {
			return lifecycle.Break(), err
		}
		for _, cluster := range clusters.Items {
			secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: cluster.GetSecretName(), Namespace: cluster.GetNamespace()}}
			if err := p.Delete(ctx, secret); client.IgnoreNotFound(err) != nil {
				return lifecycle.Break(), err
			}
		}
		return p.deleteAll(ctx, clusters)
	}
}

func (p *offboardingPhase) ensureDexResourcesDeleted() lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		if p.org.Spec.Authentication == nil || p.org.Spec.Authentication.OIDCConfig == nil {
			return lifecycle.Continue(), nil
		}
		// delete org oauth redirects from default connector
		if err := p.removeAuthRedirectFromDefaultConnector(ctx, p.org); err != nil {
			return lifecycle.Break(), err
		}
		if err := p.deleteDexConnector(ctx, p.org); err != nil {
			return lifecycle.Break(), err
		}
		if err := p.deleteOAuth2Client(ctx, p.org.Name); err != nil {
			return lifecycle.Break(), err
		}
		if err := p.deleteOAuth2Client(ctx, oauth2ProxyClientName(p.org.Name)); err != nil {
			return lifecycle.Break(), err
		}
		return lifecycle.Continue(), nil
	}
}

func (p *offboardingPhase) ensureNamespaceOrphaned() lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		namespace := &corev1.Namespace{}
		if err := p.Get(ctx, types.NamespacedName{Name: p.org.Name}, namespace); err != nil {
			if apierrors.IsNotFound(err) {
				return lifecycle.Continue(), nil
			}
			return lifecycle.Break(), err
		}
		delete(namespace.Labels, greenhouseapis.LabelKeyOrganization)
		if metav1.IsControlledBy(namespace, p.org) {
			if err := controllerutil.RemoveControllerReference(p.org, namespace, p.Scheme()); err != nil {
				return lifecycle.Break(), err
			}
		}
		if err := p.Update(ctx, namespace); err != nil {
			return lifecycle.Break(), err
		}
		return lifecycle.Continue(), nil
	}
}

func (p *offboardingPhase) deleteAll(ctx context.Context, lists ...client.ObjectList) (lifecycle.Result, error) {
	remaining := 0
	for _, list := range lists {
		if err := p.List(ctx, list, client.InNamespace(p.org.Name)); err != nil {
			return lifecycle.Break(), err
		}
		err := meta.EachListItem(list, func(item runtime.Object) error {
			remaining++
			return client.IgnoreNotFound(p.Delete(ctx, item.(client.Object)))
		})
		if err != nil {
			return lifecycle.Break(), err
		}
	}
	if remaining > 0 {
		return lifecycle.RequeueAfter(p.requeueInterval), nil
	}
	return lifecycle.Continue(), nil
}
