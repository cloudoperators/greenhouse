// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package replication

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudoperators/greenhouse/internal/common"
)

func TestReplicateImages(t *testing.T) {
	config := &common.RegistryMirrorConfig{
		PrimaryMirror: "primary.registry.com",
		RegistryMirrors: map[string]common.RegistryMirror{
			"ghcr.io": {
				BaseDomain: "primary.registry.com",
				SubPath:    "ghcr-mirror",
			},
			"docker.io": {
				BaseDomain: "primary.registry.com",
				SubPath:    "dockerhub-mirror",
			},
		},
	}

	t.Run("replicates images successfully", func(t *testing.T) {
		fetchedRefs := make([]string, 0)
		replicator := &ImageReplicator{
			config: config,
			auth:   authn.Anonymous,
			manifestFetcher: func(ref string, opts ...crane.Option) ([]byte, error) {
				fetchedRefs = append(fetchedRefs, ref)
				return []byte("{}"), nil
			},
		}

		manifests := `
containers:
- image: ghcr.io/cloudoperators/greenhouse:main
- image: docker.io/library/nginx:latest
`
		replicated, err := replicator.ReplicateImages(context.Background(), manifests, nil)
		require.NoError(t, err)
		assert.Len(t, replicated, 2)
		assert.Len(t, fetchedRefs, 2)
		assert.Contains(t, fetchedRefs, "primary.registry.com/dockerhub-mirror/library/nginx:latest")
		assert.Contains(t, fetchedRefs, "primary.registry.com/ghcr-mirror/cloudoperators/greenhouse:main")
	})

	t.Run("skips already replicated images", func(t *testing.T) {
		fetchCount := 0
		replicator := &ImageReplicator{
			config: config,
			auth:   authn.Anonymous,
			manifestFetcher: func(ref string, opts ...crane.Option) ([]byte, error) {
				fetchCount++
				return []byte("{}"), nil
			},
		}

		manifests := `
containers:
- image: ghcr.io/cloudoperators/greenhouse:main
- image: docker.io/library/nginx:latest
`
		alreadyReplicated := []string{"ghcr.io/cloudoperators/greenhouse:main"}
		replicated, err := replicator.ReplicateImages(context.Background(), manifests, alreadyReplicated)
		require.NoError(t, err)
		assert.Len(t, replicated, 2)
		assert.Equal(t, 1, fetchCount)
	})

	t.Run("skips images without configured mirror", func(t *testing.T) {
		fetchCount := 0
		replicator := &ImageReplicator{
			config: config,
			auth:   authn.Anonymous,
			manifestFetcher: func(ref string, opts ...crane.Option) ([]byte, error) {
				fetchCount++
				return []byte("{}"), nil
			},
		}

		manifests := `
containers:
- image: registry.k8s.io/pause:3.9
`
		replicated, err := replicator.ReplicateImages(context.Background(), manifests, nil)
		require.NoError(t, err)
		assert.Empty(t, replicated)
		assert.Equal(t, 0, fetchCount)
	})

	t.Run("returns partial results and error on failure", func(t *testing.T) {
		replicator := &ImageReplicator{
			config: config,
			auth:   authn.Anonymous,
			manifestFetcher: func(ref string, opts ...crane.Option) ([]byte, error) {
				if ref == "primary.registry.com/ghcr-mirror/cloudoperators/greenhouse:main" {
					return nil, errors.New("connection refused")
				}
				return []byte("{}"), nil
			},
		}

		manifests := `
containers:
- image: ghcr.io/cloudoperators/greenhouse:main
- image: docker.io/library/nginx:latest
`
		replicated, err := replicator.ReplicateImages(context.Background(), manifests, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "connection refused")
		assert.Contains(t, replicated, "docker.io/library/nginx:latest")
		assert.NotContains(t, replicated, "ghcr.io/cloudoperators/greenhouse:main")
	})

	t.Run("returns existing list when no images in manifests", func(t *testing.T) {
		replicator := &ImageReplicator{
			config: config,
			auth:   authn.Anonymous,
			manifestFetcher: func(ref string, opts ...crane.Option) ([]byte, error) {
				return []byte("{}"), nil
			},
		}

		manifests := `
apiVersion: v1
kind: ConfigMap
`
		existing := []string{"some-image:latest"}
		replicated, err := replicator.ReplicateImages(context.Background(), manifests, existing)
		require.NoError(t, err)
		assert.Equal(t, existing, replicated)
	})
}

func TestBuildMirroredImageRef(t *testing.T) {
	config := &common.RegistryMirrorConfig{
		PrimaryMirror: "primary.registry.com",
		RegistryMirrors: map[string]common.RegistryMirror{
			"ghcr.io": {
				BaseDomain: "primary.registry.com",
				SubPath:    "ghcr-mirror",
			},
			"docker.io": {
				BaseDomain: "primary.registry.com",
				SubPath:    "dockerhub-mirror",
			},
		},
	}

	replicator := &ImageReplicator{config: config, auth: authn.Anonymous}

	tests := []struct {
		name     string
		imageRef string
		expected string
	}{
		{
			name:     "ghcr.io image with tag",
			imageRef: "ghcr.io/cloudoperators/greenhouse:main",
			expected: "primary.registry.com/ghcr-mirror/cloudoperators/greenhouse:main",
		},
		{
			name:     "docker.io image with tag",
			imageRef: "docker.io/library/nginx:latest",
			expected: "primary.registry.com/dockerhub-mirror/library/nginx:latest",
		},
		{
			name:     "unqualified image defaults to docker.io",
			imageRef: "nginx:latest",
			expected: "primary.registry.com/dockerhub-mirror/library/nginx:latest",
		},
		{
			name:     "image with digest",
			imageRef: "ghcr.io/cloudoperators/greenhouse@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			expected: "primary.registry.com/ghcr-mirror/cloudoperators/greenhouse@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "no mirror configured",
			imageRef: "registry.k8s.io/pause:3.9",
			expected: "",
		},
		{
			name:     "nested path image",
			imageRef: "ghcr.io/org/team/project/app:v1.0",
			expected: "primary.registry.com/ghcr-mirror/org/team/project/app:v1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replicator.buildMirroredImageRef(tt.imageRef)
			assert.Equal(t, tt.expected, result)
		})
	}
}
