// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"context"
	"fmt"

	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/flux"
)

const (
	genericAuthProvider   = "generic"
	githubAppAuthProvider = "github"
	githubAppIDKey        = "githubAppID"
)

func (r *CatalogReconciler) getSourceSecret(ctx context.Context, catalog *greenhousev1alpha1.Catalog) (*corev1.Secret, error) {
	secretName := catalog.Spec.Source.SecretName
	if secretName == nil {
		return nil, nil
	}
	secret := &corev1.Secret{}
	secret.SetName(*secretName)
	secret.SetNamespace(catalog.Namespace)
	err := r.Get(ctx, client.ObjectKeyFromObject(secret), secret)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s for catalog %s/%s: %w", *secretName, catalog.Namespace, catalog.Name, err)
	}
	return secret, nil
}

func (r *CatalogReconciler) ensureGitRepository(ctx context.Context, catalog *greenhousev1alpha1.Catalog) error {
	gitRef := catalog.Spec.Source.Ref
	gitReference := &sourcev1.GitRepositoryRef{}
	if gitRef != nil {
		// flux precedence 1
		if gitRef.SHA != nil {
			gitReference.Commit = *gitRef.SHA
		}
		// flux precedence 2
		if gitRef.Tag != nil {
			gitReference.Tag = *gitRef.Tag
		}
		// flux precedence 3
		if gitRef.Branch != nil {
			gitReference.Branch = *gitRef.Branch
		}
	}

	authProvider := genericAuthProvider
	secret, err := r.getSourceSecret(ctx, catalog)
	if err != nil {
		catalog.SetTrueCondition(greenhousev1alpha1.CatalogSourceSecretErrorCondition, greenhousev1alpha1.CatalogSecretErrorReason, err.Error())
		return err
	} else {
		catalog.RemoveCondition(greenhousev1alpha1.CatalogSourceSecretErrorCondition)
	}

	if secret != nil {
		if _, ok := secret.Data[githubAppIDKey]; ok {
			authProvider = githubAppAuthProvider
		}
	}

	gitRepository := &sourcev1.GitRepository{}
	gitRepository.SetName(catalog.Name)
	gitRepository.SetNamespace(catalog.Namespace)
	result, err := controllerutil.CreateOrPatch(ctx, r.Client, gitRepository, func() error {
		gitRepository.Spec = sourcev1.GitRepositorySpec{
			URL:       catalog.Spec.Source.Repository,
			Interval:  metav1.Duration{Duration: flux.DefaultInterval},
			Reference: gitReference,
			Provider:  authProvider,
		}
		if secret != nil {
			gitRepository.Spec.SecretRef = &fluxmeta.LocalObjectReference{Name: *catalog.Spec.Source.SecretName}
		}
		return controllerutil.SetControllerReference(catalog, gitRepository, r.Scheme)
	})
	if err != nil {
		return err
	}
	switch result {
	case controllerutil.OperationResultCreated:
		r.Log.Info("created git repository for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
	case controllerutil.OperationResultUpdated:
		r.Log.Info("updated git repository for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
	case controllerutil.OperationResultNone:
		r.Log.Info("No changes to catalog git repository", "name", gitRepository.Name, "namespace", gitRepository.Namespace)
	default:
		r.Log.Info("result is unknown for catalog git repository", "name", catalog.Name, "namespace", catalog.Namespace, "result", result)
	}
	return nil
}
