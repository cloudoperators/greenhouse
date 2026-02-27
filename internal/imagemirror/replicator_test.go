// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package imagemirror

import (
	"context"
	"errors"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

)

var mirrorConfig = &RegistryMirrorConfig{
	PrimaryMirror: "primary.registry.com",
	RegistryMirrors: map[string]RegistryMirror{
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

var _ = Describe("ReplicateImages", func() {
	It("should replicate images successfully", func() {
		fetchedRefs := make([]string, 0)
		replicator := &ImageReplicator{
			config: mirrorConfig,
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
		Expect(err).NotTo(HaveOccurred())
		Expect(replicated).To(HaveLen(2))
		Expect(fetchedRefs).To(HaveLen(2))
		Expect(fetchedRefs).To(ContainElement("primary.registry.com/dockerhub-mirror/library/nginx:latest"))
		Expect(fetchedRefs).To(ContainElement("primary.registry.com/ghcr-mirror/cloudoperators/greenhouse:main"))
	})

	It("should skip already replicated images", func() {
		fetchCount := 0
		replicator := &ImageReplicator{
			config: mirrorConfig,
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
		Expect(err).NotTo(HaveOccurred())
		Expect(replicated).To(HaveLen(2))
		Expect(fetchCount).To(Equal(1))
	})

	It("should skip images without configured mirror", func() {
		fetchCount := 0
		replicator := &ImageReplicator{
			config: mirrorConfig,
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
		Expect(err).NotTo(HaveOccurred())
		Expect(replicated).To(BeEmpty())
		Expect(fetchCount).To(Equal(0))
	})

	It("should return partial results and error on failure", func() {
		replicator := &ImageReplicator{
			config: mirrorConfig,
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
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("connection refused"))
		Expect(replicated).To(ContainElement("docker.io/library/nginx:latest"))
		Expect(replicated).NotTo(ContainElement("ghcr.io/cloudoperators/greenhouse:main"))
	})

	It("should return existing list when no images in manifests", func() {
		replicator := &ImageReplicator{
			config: mirrorConfig,
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
		Expect(err).NotTo(HaveOccurred())
		Expect(replicated).To(Equal(existing))
	})
})

var _ = Describe("buildMirroredImageRef", func() {
	replicator := &ImageReplicator{config: mirrorConfig, auth: authn.Anonymous}

	DescribeTable("should build correct mirrored refs",
		func(imageRef, expected string) {
			Expect(replicator.buildMirroredImageRef(imageRef)).To(Equal(expected))
		},
		Entry("ghcr.io image with tag",
			"ghcr.io/cloudoperators/greenhouse:main",
			"primary.registry.com/ghcr-mirror/cloudoperators/greenhouse:main"),
		Entry("docker.io image with tag",
			"docker.io/library/nginx:latest",
			"primary.registry.com/dockerhub-mirror/library/nginx:latest"),
		Entry("unqualified image defaults to docker.io",
			"nginx:latest",
			"primary.registry.com/dockerhub-mirror/library/nginx:latest"),
		Entry("image with digest",
			"ghcr.io/cloudoperators/greenhouse@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			"primary.registry.com/ghcr-mirror/cloudoperators/greenhouse@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"),
		Entry("no mirror configured",
			"registry.k8s.io/pause:3.9",
			""),
		Entry("nested path image",
			"ghcr.io/org/team/project/app:v1.0",
			"primary.registry.com/ghcr-mirror/org/team/project/app:v1.0"),
	)
})
