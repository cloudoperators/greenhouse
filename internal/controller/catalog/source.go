// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"context"
	"fmt"
	"strings"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	"github.com/fluxcd/pkg/apis/kustomize"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	sourcev2 "github.com/fluxcd/source-watcher/api/v2/v1beta1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/flux"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
	"github.com/cloudoperators/greenhouse/internal/rbac"
)

const (
	genericAuthProvider   = "generic"
	githubAppAuthProvider = "github"
	githubAppIDKey        = "githubAppID"
)

const (
	artifactAlias   = "catalog"
	artifactCopyKey = "artifact"
	artifactToDir   = "catalogs" // all resources in Catalog.Spec.Resources will be copied under this dir in the ExternalArtifact
)

const (
	gitRepoArtifactPrefix   = "repository"
	generatorArtifactPrefix = "generator"
	externalArtifactPrefix  = "artifact"
	kustomizeArtifactPrefix = "kustomize"
)

type sourceError struct {
	errorTxt error
	kind     string
	name     string
	groupKey string
}

func (o *sourceError) Error() string {
	return o.errorTxt.Error()
}

type source struct {
	client.Client
	scheme           *runtime.Scheme
	log              logr.Logger
	recorder         record.EventRecorder
	catalog          *greenhousev1alpha1.Catalog
	source           greenhousev1alpha1.CatalogSource
	sourceHash       string
	sourceGroup      string
	gitRepo          greenhousev1alpha1.SourceStatus
	generator        greenhousev1alpha1.SourceStatus
	externalArtifact greenhousev1alpha1.SourceStatus
	kustomize        greenhousev1alpha1.SourceStatus
}

// validateSources - ensures there are no duplicate sources in the catalog spec
// duplicates are determined by repository URL and ref (branch/tag/sha)
func (r *CatalogReconciler) validateSources(catalog *greenhousev1alpha1.Catalog) error {
	sourceHashes := make(map[string]bool)
	for _, source := range catalog.Spec.Sources {
		host, owner, repo, err := lifecycle.GetOwnerRepoInfo(source.Repository)
		if err != nil {
			return err
		}
		ref := source.GetRefValue()
		hash, err := lifecycle.HashValue(fmt.Sprintf("%s-%s-%s-%s-%s", catalog.Name, host, owner, repo, ref))
		if err != nil {
			return err
		}
		if _, exists := sourceHashes[hash]; exists {
			return fmt.Errorf("duplicate source found: repository %s with ref %s", source.Repository, ref)
		}
		sourceHashes[hash] = true
	}
	return nil
}

// newCatalogSource - prepares a source struct for Catalog.Spec.Sources entry with necessary metadata
func (r *CatalogReconciler) newCatalogSource(catalogSource greenhousev1alpha1.CatalogSource, catalog *greenhousev1alpha1.Catalog) (*source, error) {
	host, owner, repo, err := lifecycle.GetOwnerRepoInfo(catalogSource.Repository)
	if err != nil {
		return nil, err
	}
	ref := catalogSource.GetRefValue()
	hash, err := lifecycle.HashValue(fmt.Sprintf("%s-%s-%s-%s-%s", catalog.Name, host, owner, repo, ref))
	if err != nil {
		return nil, err
	}
	return &source{
		Client:           r.Client,
		scheme:           r.Scheme,
		log:              r.Log,
		recorder:         r.recorder,
		catalog:          catalog,
		source:           catalogSource,
		sourceHash:       hash,
		sourceGroup:      fmt.Sprintf("%s-%s-%s-%s", host, owner, repo, ref),
		gitRepo:          greenhousev1alpha1.SourceStatus{Kind: sourcev1.GitRepositoryKind, Name: gitRepoArtifactPrefix + "-" + hash},
		generator:        greenhousev1alpha1.SourceStatus{Kind: sourcev2.ArtifactGeneratorKind, Name: generatorArtifactPrefix + "-" + hash},
		externalArtifact: greenhousev1alpha1.SourceStatus{Kind: sourcev1.ExternalArtifactKind, Name: externalArtifactPrefix + "-" + hash},
		kustomize:        greenhousev1alpha1.SourceStatus{Kind: kustomizev1.KustomizationKind, Name: kustomizeArtifactPrefix + "-" + hash},
	}, nil
}

func (s *source) getSourceGroupHash() string {
	return s.sourceGroup + "-" + s.sourceHash
}

func (s *source) getInventory() []greenhousev1alpha1.SourceStatus {
	return []greenhousev1alpha1.SourceStatus{s.gitRepo, s.generator, s.externalArtifact, s.kustomize}
}

func (s *source) getGitRepoName() string {
	return s.gitRepo.Name
}

func (s *source) getArtifactName() string {
	return s.externalArtifact.Name
}

func (s *source) getGeneratorName() string {
	return s.generator.Name
}

func (s *source) getKustomizationName() string {
	return s.kustomize.Name
}

func (s *source) setInventory(kind, name, msg string, ready metav1.ConditionStatus) {
	s.catalog.SetInventory(s.getSourceGroupHash(), kind, name, msg, ready)
}

// getSourceSecret - retrieves the Secret resource referenced in the Catalog.Spec.Sources[].SecretName if it exists
func (s *source) getSourceSecret(ctx context.Context) (*corev1.Secret, error) {
	if s.source.SecretName == nil {
		return nil, nil
	}
	secret := &corev1.Secret{}
	secret.SetName(*s.source.SecretName)
	secret.SetNamespace(s.catalog.Namespace)
	err := s.Get(ctx, client.ObjectKeyFromObject(secret), secret)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s/%s: %w", s.catalog.Namespace, *s.source.SecretName, err)
	}
	return secret, nil
}

// reconcileGitRepository - ensures the GitRepository resource is up to date
func (s *source) reconcileGitRepository(ctx context.Context) error {
	gitRepo := &sourcev1.GitRepository{}
	gitRepo.SetName(s.gitRepo.Name)
	gitRepo.SetNamespace(s.catalog.Namespace)
	secretRef, err := s.getSourceSecret(ctx)
	if err != nil {
		return &sourceError{
			errorTxt: err,
			kind:     sourcev1.GitRepositoryKind,
			name:     gitRepo.Name,
			groupKey: s.getSourceGroupHash(),
		}
	}
	spec := sourcev1.GitRepositorySpec{
		URL:       s.source.Repository,
		Interval:  metav1.Duration{Duration: flux.DefaultInterval},
		Reference: s.source.GetGitRepositoryReference(),
		Provider:  genericAuthProvider,
	}
	spec.Provider = genericAuthProvider
	if secretRef != nil {
		spec.SecretRef = &fluxmeta.LocalObjectReference{Name: secretRef.Name}
		if _, ok := secretRef.Data[githubAppIDKey]; ok {
			spec.Provider = githubAppAuthProvider
		}
	}

	result, err := controllerutil.CreateOrPatch(ctx, s.Client, gitRepo, func() error {
		gitRepo.Spec = spec
		return controllerutil.SetControllerReference(s.catalog, gitRepo, s.scheme)
	})
	if err != nil {
		return &sourceError{
			errorTxt: err,
			kind:     sourcev1.GitRepositoryKind,
			name:     gitRepo.Name,
			groupKey: s.getSourceGroupHash(),
		}
	}
	switch result {
	case controllerutil.OperationResultCreated:
		s.log.Info("created git repository for catalog", "name", gitRepo.Name, "namespace", gitRepo.Namespace)
	case controllerutil.OperationResultUpdated:
		s.log.Info("updated git repository for catalog", "name", gitRepo.Name, "namespace", gitRepo.Namespace)
	case controllerutil.OperationResultNone:
		s.log.Info("No changes to catalog git repository", "name", gitRepo.Name, "namespace", gitRepo.Namespace)
	default:
		s.log.Info("result is unknown for catalog git repository", "name", gitRepo.Name, "namespace", gitRepo.Namespace, "result", result)
	}
	return nil
}

// getArtifactSource - returns the SourceReference for the ArtifactGenerator
// pointing to the GitRepository resource where Catalog.Spec.Resources need to be extracted from
func (s *source) getArtifactSource() []sourcev2.SourceReference {
	return []sourcev2.SourceReference{
		{
			Alias:     artifactAlias,
			Name:      s.gitRepo.Name,
			Namespace: s.catalog.Namespace,
			Kind:      sourcev1.GitRepositoryKind,
		},
	}
}

// getArtifacts - returns the OutputArtifacts for the ArtifactGenerator
// list of resources that needs to be bundled from the Catalog.Spec.Resources
func (s *source) getArtifacts() []sourcev2.OutputArtifact {
	resources := s.source.Resources
	if len(resources) == 0 {
		return nil
	}
	copyOps := make([]sourcev2.CopyOperation, 0, len(resources))
	for _, res := range resources {
		res = strings.TrimPrefix(res, "/")
		copyOps = append(copyOps, sourcev2.CopyOperation{
			From: fmt.Sprintf("@%s/%s", artifactAlias, res),
			To:   fmt.Sprintf("@%s/%s/%s", artifactCopyKey, artifactToDir, res),
		})
	}
	return []sourcev2.OutputArtifact{
		{
			Name: s.externalArtifact.Name,
			Copy: copyOps,
		},
	}
}

// reconcileArtifactGeneration - ensures the ArtifactGenerator and ExternalArtifact resources are up to date
func (s *source) reconcileArtifactGeneration(ctx context.Context) (*sourcev1.ExternalArtifact, error) {
	generator := &sourcev2.ArtifactGenerator{}
	generator.SetName(s.generator.Name)
	generator.SetNamespace(s.catalog.Namespace)
	artifactSources := s.getArtifactSource()
	artifacts := s.getArtifacts()
	if len(artifacts) == 0 {
		return nil, &sourceError{
			errorTxt: fmt.Errorf("no resources defined in source %s/%s to generate artifact", generator.Namespace, s.getGitRepoName()),
			kind:     sourcev2.ArtifactGeneratorKind,
			name:     generator.Name,
			groupKey: s.getSourceGroupHash(),
		}
	}
	result, err := controllerutil.CreateOrPatch(ctx, s.Client, generator, func() error {
		generator.Spec.Sources = artifactSources
		generator.Spec.OutputArtifacts = artifacts
		return controllerutil.SetControllerReference(s.catalog, generator, s.scheme)
	})
	if err != nil {
		return nil, &sourceError{
			errorTxt: err,
			kind:     sourcev2.ArtifactGeneratorKind,
			name:     generator.Name,
			groupKey: s.getSourceGroupHash(),
		}
	}
	switch result {
	case controllerutil.OperationResultCreated:
		s.log.Info("created artifact generator for catalog", "name", generator.Name, "namespace", generator.Namespace)
	case controllerutil.OperationResultUpdated:
		s.log.Info("updated artifact generator for catalog", "name", generator.Name, "namespace", generator.Namespace)
	case controllerutil.OperationResultNone:
		s.log.Info("no changes to catalog artifact generator", "name", generator.Name, "namespace", generator.Namespace)
	default:
		s.log.Info("result is unknown for catalog artifact generator", "name", generator.Name, "namespace", generator.Namespace, "result", result)
	}
	extArtifact := &sourcev1.ExternalArtifact{}
	extArtifact.SetName(s.externalArtifact.Name)
	extArtifact.SetNamespace(s.catalog.Namespace)
	err = s.Get(ctx, client.ObjectKeyFromObject(extArtifact), extArtifact)
	if err != nil {
		return nil, &sourceError{
			errorTxt: err,
			kind:     sourcev1.ExternalArtifactKind,
			name:     extArtifact.Name,
			groupKey: s.getSourceGroupHash(),
		}
	}
	return extArtifact, nil
}

// reconcileKustomization - ensures the Kustomization resource is up to date
func (s *source) reconcileKustomization(ctx context.Context, extArtifact *sourcev1.ExternalArtifact) error {
	kustomization := &kustomizev1.Kustomization{}
	kustomization.SetName(s.kustomize.Name)
	kustomization.SetNamespace(s.catalog.Namespace)
	var err error
	var patches []kustomize.Patch
	if len(s.source.Overrides) > 0 {
		if patches, err = flux.PrepareKustomizePatches(s.source.Overrides, greenhousev1alpha1.GroupVersion.Group); err != nil {
			return &sourceError{
				errorTxt: err,
				kind:     kustomizev1.KustomizationKind,
				name:     kustomization.Name,
				groupKey: s.getSourceGroupHash(),
			}
		}
	}
	ggvk := extArtifact.GroupVersionKind()
	// ServiceAccount for the organization's PluginDefinitionCatalog operations
	serviceAccountName := rbac.OrgCatalogServiceAccountName(s.catalog.Namespace)
	spec, err := flux.NewKustomizationSpecBuilder(s.log).
		WithSourceRef(ggvk.String(), ggvk.Kind, extArtifact.Name, extArtifact.Namespace).
		WithServiceAccountName(serviceAccountName).
		WithPatches(patches).
		// this is necessary for kustomize to apply namespaced resources without errors,
		// as the resources in git will not have the namespace set.
		// namespace is ignored for Cluster scoped resources
		WithTargetNamespace(s.catalog.Namespace).
		// plugindefinitions applied can also have catalog source labels set on them
		// but on kustomize deletion the label stays behind since prune policy is to Retain.
		// WithCommonLabels(s.commonArtifactLabels).
		WithPath(artifactToDir).Build()
	if err != nil {
		return &sourceError{
			errorTxt: err,
			kind:     kustomizev1.KustomizationKind,
			name:     kustomization.Name,
			groupKey: s.getSourceGroupHash(),
		}
	}
	// when flux resources is being updated by greenhouse controller and in parallel by flux controller, we need to retryOnConflict
	result, err := controllerutil.CreateOrPatch(ctx, s.Client, kustomization, func() error {
		kustomization.Spec = spec
		return controllerutil.SetControllerReference(s.catalog, kustomization, s.scheme)
	})
	if err != nil {
		return &sourceError{
			errorTxt: err,
			kind:     kustomizev1.KustomizationKind,
			name:     kustomization.Name,
			groupKey: s.getSourceGroupHash(),
		}
	}
	switch result {
	case controllerutil.OperationResultCreated:
		s.log.Info("created kustomization for catalog", "name", kustomization.Name, "namespace", kustomization.Namespace)
	case controllerutil.OperationResultUpdated:
		s.log.Info("updated kustomization for catalog", "name", kustomization.Name, "namespace", kustomization.Namespace)
	case controllerutil.OperationResultNone:
		s.log.Info("No changes to catalog kustomization", "name", kustomization.Name, "namespace", kustomization.Namespace)
	default:
		s.log.Info("result is unknown for catalog kustomization", "name", kustomization.Name, "namespace", kustomization.Namespace, "result", result)
	}
	return nil
}

// objectReadiness - checks the Ready condition of a catalog object (GitRepository, ArtifactGenerator, ExternalArtifact, Kustomization)
// if not Ready, then the controller adds the Catalog object to requeue
func (s *source) objectReadiness(ctx context.Context, obj client.Object) (ready metav1.ConditionStatus, msg string) {
	ready = metav1.ConditionFalse
	key := client.ObjectKeyFromObject(obj)
	if err := s.Get(ctx, key, obj); err != nil {
		s.log.Error(err, "failed to get object", "key", key)
		msg = err.Error()
		return
	}

	kind := obj.GetObjectKind().GroupVersionKind().Kind
	cObj, ok := obj.(lifecycle.CatalogObject)
	if !ok {
		err := fmt.Errorf("failed to assert catalog object kind %s - %s/%s", kind, key.Namespace, key.Name)
		s.log.Error(err, "failed to assert catalog object", "key", key)
		msg = err.Error()
		return
	}

	conditions := cObj.GetConditions()
	readyCondition := meta.FindStatusCondition(conditions, fluxmeta.ReadyCondition)

	if readyCondition == nil {
		s.log.Info("Object not ready yet, requeue...", "kind", kind, "namespace", key.Namespace, "name", key.Name)
		msg = kind + " not ready yet"
		return
	}
	ready = readyCondition.Status
	msg = readyCondition.Message
	return
}
