// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package ocimirror

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// manifestFetcherFunc is a function type matching the crane.Manifest signature.
type manifestFetcherFunc func(ref string, opts ...crane.Option) ([]byte, error)

// OCIReplicator ensures OCI artifacts are available in the mirror registry by fetching their manifests,
// triggering on-demand replication from the upstream source.
type OCIReplicator struct {
	config          *RegistryMirrorConfig
	auth            authn.Authenticator
	manifestFetcher manifestFetcherFunc
}

// NewOCIReplicator creates an OCIReplicator with credentials from the configured Secret.
func NewOCIReplicator(ctx context.Context, k8sClient client.Client, config *RegistryMirrorConfig, namespace string) (*OCIReplicator, error) {
	auth := authn.Anonymous
	if config.SecretName != "" {
		a, err := getAuthFromSecret(ctx, k8sClient, config.SecretName, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to read registry mirror credentials from secret %s/%s: %w", namespace, config.SecretName, err)
		}
		auth = a
	}

	return &OCIReplicator{
		config:          config,
		auth:            auth,
		manifestFetcher: crane.Manifest,
	}, nil
}

// ReplicateOCIArtifacts triggers replication for new OCI artifacts found in renderedManifests, skipping alreadyReplicated ones.
func (r *OCIReplicator) ReplicateOCIArtifacts(ctx context.Context, renderedManifests string, alreadyReplicated []string) ([]string, error) {
	imageRefs := ExtractUniqueOCIRefs(renderedManifests)
	if len(imageRefs) == 0 {
		return alreadyReplicated, nil
	}

	alreadySet := make(map[string]struct{}, len(alreadyReplicated))
	for _, img := range alreadyReplicated {
		alreadySet[img] = struct{}{}
	}

	replicated := slices.Clone(alreadyReplicated)

	var replicationErrors []string
	for _, imageRef := range imageRefs {
		mirroredRef := r.BuildMirroredOCIRef(imageRef)
		if mirroredRef == "" {
			continue
		}

		if _, ok := alreadySet[imageRef]; ok {
			continue
		}

		if _, err := r.TriggerReplication(ctx, mirroredRef, crane.WithPlatform(&v1.Platform{OS: "linux", Architecture: "amd64"})); err != nil {
			replicationErrors = append(replicationErrors, fmt.Sprintf("%s: %v", imageRef, err))
			continue
		}

		replicated = append(replicated, imageRef)
	}

	if len(replicationErrors) > 0 {
		return replicated, fmt.Errorf("failed to replicate images: %s", strings.Join(replicationErrors, "; "))
	}

	return replicated, nil
}

// BuildMirroredOCIRef resolves imageRef against the mirror config and returns the mirrored reference.
func (r *OCIReplicator) BuildMirroredOCIRef(imageRef string) string {
	resolved := r.config.ResolveOCIRef(imageRef)
	if resolved == nil {
		return ""
	}

	mirroredRef := fmt.Sprintf("%s/%s/%s", resolved.Mirror.BaseDomain, resolved.Mirror.SubPath, resolved.Repository)
	if resolved.TagOrDigest != "" {
		mirroredRef += resolved.TagOrDigest
	}

	return mirroredRef
}

// TriggerReplication fetches the manifest for mirroredRef to warm the pull-through cache.
// Returns the raw manifest bytes on success for digest computation.
func (r *OCIReplicator) TriggerReplication(ctx context.Context, mirroredRef string, extraOpts ...crane.Option) ([]byte, error) {
	log.FromContext(ctx).V(1).Info("triggering replication", "ref", mirroredRef)
	opts := append([]crane.Option{crane.WithAuth(r.auth)}, extraOpts...)
	manifest, err := r.manifestFetcher(mirroredRef, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest for %s: %w", mirroredRef, err)
	}

	return manifest, nil
}

func getAuthFromSecret(ctx context.Context, k8sClient client.Client, secretName, namespace string) (authn.Authenticator, error) {
	secret := &corev1.Secret{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, secret); err != nil {
		return nil, fmt.Errorf("failed to get secret %s/%s: %w", namespace, secretName, err)
	}

	username := string(secret.Data["username"])
	password := string(secret.Data["password"])

	if username == "" || password == "" {
		return nil, fmt.Errorf("secret %s/%s missing username or password", namespace, secretName)
	}

	return &authn.Basic{
		Username: username,
		Password: password,
	}, nil
}
