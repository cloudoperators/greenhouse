// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"context"
	"fmt"
	"time"

	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
)

const gitCatalogSourceSuffix = "-git-repository"

func (r *CatalogReconciler) buildGitReference(gitSource *greenhousev1alpha1.GitSource) *sourcev1.GitRepositoryRef {
	if gitSource.Ref != nil {
		// flux precedence 1
		if gitSource.Ref.SHA != nil {
			return &sourcev1.GitRepositoryRef{
				Commit: *gitSource.Ref.SHA,
			}
		}
		// flux precedence 2
		if gitSource.Ref.Tag != nil {
			return &sourcev1.GitRepositoryRef{
				Tag: *gitSource.Ref.Tag,
			}
		}
		// flux precedence 3
		if gitSource.Ref.Branch != nil {
			return &sourcev1.GitRepositoryRef{
				Branch: *gitSource.Ref.Branch,
			}
		}
	}
	return nil
}

func (r *CatalogReconciler) ensureGitRepository(catalog *greenhousev1alpha1.PluginDefinitionCatalog) lifecycle.ReconcileRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		var err error
		gitSource := catalog.GetCatalogSource()
		if gitSource == nil {
			err = fmt.Errorf("catalog %s/%s has no git source defined", catalog.Namespace, catalog.Name)
			return lifecycle.Break(), err
		}
		gitReference := r.buildGitReference(gitSource)

		gitRepository := &sourcev1.GitRepository{}
		gitRepository.SetName(catalog.Name + gitCatalogSourceSuffix)
		gitRepository.SetNamespace(catalog.Namespace)
		// when flux resources is being updated by greenhouse controller and in parallel by flux controller, we need to retryOnConflict
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			result, err := ctrl.CreateOrUpdate(ctx, r.Client, gitRepository, func() error {
				gitRepository.Spec = sourcev1.GitRepositorySpec{
					URL:       gitSource.URL,
					Interval:  catalog.Interval(),
					Reference: gitReference,
					Suspend:   catalog.IsSuspended(),
					Timeout:   catalog.Timeout(),
				}
				return controllerutil.SetControllerReference(catalog, gitRepository, r.Scheme)
			})
			if err != nil {
				return err
			}
			switch result {
			case controllerutil.OperationResultCreated:
				r.log.Info("created git repository for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
			case controllerutil.OperationResultUpdated:
				r.log.Info("updated git repository for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
			case controllerutil.OperationResultNone:
				r.log.Info("No changes to catalog git repository", "name", gitRepository.Name, "namespace", gitRepository.Namespace)
			default:
				r.log.Info("result is unknown for catalog git repository", "name", catalog.Name, "namespace", catalog.Namespace, "result", result)
			}
			return nil
		})
		if err != nil {
			catalog.SetGitRepositoryReadyFalse(greenhousev1alpha1.CatalogRepositoryFailReason, err.Error())
			return lifecycle.Break(), err
		}
		catalog.SetGitRepositoryReadyUnknown("", "git repository for catalog is being created or updated")
		return lifecycle.Continue(), nil
	}
}

func (r *CatalogReconciler) ensureGitRepositoryIsReady(catalog *greenhousev1alpha1.PluginDefinitionCatalog) lifecycle.ReconcileRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		gitRepository := &sourcev1.GitRepository{}
		gitRepository.SetName(catalog.Name + gitCatalogSourceSuffix)
		gitRepository.SetNamespace(catalog.Namespace)
		err := r.Client.Get(ctx, client.ObjectKeyFromObject(gitRepository), gitRepository)
		if err != nil {
			r.log.Error(err, "failed to get git repository for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
			return lifecycle.Break(), fmt.Errorf("failed to get git repository %s/%s: %w", gitRepository.Namespace, gitRepository.Name, err)
		}
		r.gitRepository = gitRepository
		if catalog.IsSuspended() {
			r.log.Info("catalog is suspended, skipping readiness check for git repository", "name", catalog.Name, "namespace", catalog.Namespace)
			catalog.SetGitRepositoryReadyUnknown(greenhousev1alpha1.CatalogSuspendedReason, "git repository is suspended")
			return lifecycle.Continue(), nil
		}
		readyCond := getReadyCondition(gitRepository.Status.Conditions)
		if readyCond == nil {
			r.log.Info("catalog git repository resource is not yet ready", "name", gitRepository.Name, "namespace", gitRepository.Namespace)
			return lifecycle.Requeue(), nil
		}

		if readyCond.Status == metav1.ConditionUnknown {
			catalog.SetGitRepositoryReadyUnknown(greenhousemetav1alpha1.ConditionReason(readyCond.Reason), readyCond.Message)
			r.log.Info("catalog git repository resource is in an unknown state, retrying in 10 seconds", "name", gitRepository.Name, "namespace", gitRepository.Namespace)
			return lifecycle.RequeueAfter(10 * time.Second), nil
		}

		if readyCond.Status == metav1.ConditionFalse {
			catalog.SetGitRepositoryReadyFalse(greenhousemetav1alpha1.ConditionReason(readyCond.Reason), readyCond.Message)
			return lifecycle.Break(), fmt.Errorf("catalog git repository %s/%s is not ready: %s", gitRepository.Namespace, gitRepository.Name, readyCond.Message)
		}
		catalog.SetGitRepositoryReadyTrue(greenhousemetav1alpha1.ConditionReason(readyCond.Reason), readyCond.Message)
		catalog.Status.RepositoryArtifact = gitRepository.Status.Artifact
		r.log.Info("catalog git repository resource is ready", "name", gitRepository.Name, "namespace", gitRepository.Namespace)
		return lifecycle.Continue(), nil
	}
}
