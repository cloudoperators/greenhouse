// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package replication

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

	"github.com/cloudoperators/greenhouse/internal/common"
)

// ManifestFetcher matches the crane.Manifest signature.
type ManifestFetcher func(ref string, opts ...crane.Option) ([]byte, error)

// ImageReplicator ensures images are available in the mirror registry by fetching their manifests,
// triggering on-demand replication from the upstream source.
type ImageReplicator struct {
	config          *common.RegistryMirrorConfig
	auth            authn.Authenticator
	manifestFetcher ManifestFetcher
}

// NewImageReplicator creates an ImageReplicator with credentials from the configured Secret.
func NewImageReplicator(ctx context.Context, k8sClient client.Client, config *common.RegistryMirrorConfig, namespace string) (*ImageReplicator, error) {
	auth := authn.Anonymous
	if config.SecretName != "" {
		a, err := getAuthFromSecret(ctx, k8sClient, config.SecretName, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to read registry mirror credentials from secret %s/%s: %w", namespace, config.SecretName, err)
		}
		auth = a
	}

	return &ImageReplicator{
		config:          config,
		auth:            auth,
		manifestFetcher: crane.Manifest,
	}, nil
}

// ReplicateImages triggers replication for new images found in renderedManifests, skipping alreadyReplicated ones.
func (r *ImageReplicator) ReplicateImages(ctx context.Context, renderedManifests string, alreadyReplicated []string) ([]string, error) {
	imageRefs := common.ExtractUniqueImages(renderedManifests)
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
		mirroredRef := r.buildMirroredImageRef(imageRef)
		if mirroredRef == "" {
			continue
		}

		if _, ok := alreadySet[imageRef]; ok {
			continue
		}

		if err := r.triggerReplication(ctx, mirroredRef); err != nil {
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

func (r *ImageReplicator) buildMirroredImageRef(imageRef string) string {
	resolved := r.config.ResolveImage(imageRef)
	if resolved == nil {
		return ""
	}

	mirroredRef := fmt.Sprintf("%s/%s/%s", r.config.PrimaryMirror, resolved.Mirror.SubPath, resolved.Repository)
	if resolved.TagOrDigest != "" {
		mirroredRef += resolved.TagOrDigest
	}

	return mirroredRef
}

func (r *ImageReplicator) triggerReplication(ctx context.Context, mirroredRef string) error {
	log.FromContext(ctx).V(1).Info("triggering replication for image", "ref", mirroredRef)
	_, err := r.manifestFetcher(mirroredRef,
		crane.WithPlatform(&v1.Platform{OS: "linux", Architecture: "amd64"}),
		crane.WithAuth(r.auth),
	)
	if err != nil {
		return fmt.Errorf("failed to fetch manifest for %s: %w", mirroredRef, err)
	}

	return nil
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
