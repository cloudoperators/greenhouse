// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package ocimirror

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
			BaseDomain: "global.registry.com",
			SubPath:    "ghcr-mirror",
		},
		"docker.io": {
			BaseDomain: "global.registry.com",
			SubPath:    "dockerhub-mirror",
		},
	},
}

func newTestImageMirror(config *RegistryMirrorConfig, fetcher manifestFetcherFunc) *ImageMirror {
	return &ImageMirror{
		config:          config,
		auth:            authn.Anonymous,
		manifestFetcher: fetcher,
	}
}

var _ = Describe("EnsureChartReplicated", func() {
	It("should replicate via primaryMirror when ref is on baseDomain", func() {
		var fetchedRef string
		mirror := newTestImageMirror(mirrorConfig, func(ref string, opts ...crane.Option) ([]byte, error) {
			fetchedRef = ref
			return []byte(`{"test":"manifest"}`), nil
		})

		replicatedRef, manifest, err := mirror.EnsureChartReplicated(context.Background(), "global.registry.com/ghcr-mirror/cloudoperators/greenhouse:v1.0")
		Expect(err).NotTo(HaveOccurred())
		Expect(replicatedRef).To(Equal("global.registry.com/ghcr-mirror/cloudoperators/greenhouse:v1.0"))
		Expect(fetchedRef).To(Equal("primary.registry.com/ghcr-mirror/cloudoperators/greenhouse:v1.0"))
		Expect(manifest).NotTo(BeEmpty())
	})

	It("should replicate directly when ref is already on primaryMirror", func() {
		var fetchedRef string
		mirror := newTestImageMirror(mirrorConfig, func(ref string, opts ...crane.Option) ([]byte, error) {
			fetchedRef = ref
			return []byte(`{"test":"manifest"}`), nil
		})

		replicatedRef, manifest, err := mirror.EnsureChartReplicated(context.Background(), "primary.registry.com/ghcr-mirror/cloudoperators/greenhouse:v1.0")
		Expect(err).NotTo(HaveOccurred())
		Expect(replicatedRef).To(Equal("primary.registry.com/ghcr-mirror/cloudoperators/greenhouse:v1.0"))
		Expect(fetchedRef).To(Equal("primary.registry.com/ghcr-mirror/cloudoperators/greenhouse:v1.0"))
		Expect(manifest).NotTo(BeEmpty())
	})

	It("should return empty for upstream refs (charts should not be upstream)", func() {
		fetchCount := 0
		mirror := newTestImageMirror(mirrorConfig, func(ref string, opts ...crane.Option) ([]byte, error) {
			fetchCount++
			return []byte("{}"), nil
		})

		replicatedRef, manifest, err := mirror.EnsureChartReplicated(context.Background(), "ghcr.io/cloudoperators/greenhouse:main")
		Expect(err).NotTo(HaveOccurred())
		Expect(replicatedRef).To(BeEmpty())
		Expect(manifest).To(BeNil())
		Expect(fetchCount).To(Equal(0))
	})

	It("should return empty when no mirror relationship exists", func() {
		fetchCount := 0
		mirror := newTestImageMirror(mirrorConfig, func(ref string, opts ...crane.Option) ([]byte, error) {
			fetchCount++
			return []byte("{}"), nil
		})

		replicatedRef, manifest, err := mirror.EnsureChartReplicated(context.Background(), "registry.k8s.io/pause:3.9")
		Expect(err).NotTo(HaveOccurred())
		Expect(replicatedRef).To(BeEmpty())
		Expect(manifest).To(BeNil())
		Expect(fetchCount).To(Equal(0))
	})

	It("should propagate replication errors", func() {
		mirror := newTestImageMirror(mirrorConfig, func(ref string, opts ...crane.Option) ([]byte, error) {
			return nil, errors.New("connection refused")
		})

		_, _, err := mirror.EnsureChartReplicated(context.Background(), "primary.registry.com/ghcr-mirror/cloudoperators/greenhouse:v1.0")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("connection refused"))
	})
})

var _ = Describe("EnsureImageReplicated", func() {
	It("should rewrite upstream ref to mirror and replicate", func() {
		var fetchedRef string
		mirror := newTestImageMirror(mirrorConfig, func(ref string, opts ...crane.Option) ([]byte, error) {
			fetchedRef = ref
			return []byte(`{"test":"manifest"}`), nil
		})

		replicatedRef, manifest, err := mirror.EnsureImageReplicated(context.Background(), "ghcr.io/cloudoperators/greenhouse:main")
		Expect(err).NotTo(HaveOccurred())
		Expect(replicatedRef).To(Equal("primary.registry.com/ghcr-mirror/cloudoperators/greenhouse:main"))
		Expect(fetchedRef).To(Equal("primary.registry.com/ghcr-mirror/cloudoperators/greenhouse:main"))
		Expect(manifest).NotTo(BeEmpty())
	})

	It("should replicate via primaryMirror when ref is on baseDomain", func() {
		var fetchedRef string
		mirror := newTestImageMirror(mirrorConfig, func(ref string, opts ...crane.Option) ([]byte, error) {
			fetchedRef = ref
			return []byte(`{"test":"manifest"}`), nil
		})

		replicatedRef, manifest, err := mirror.EnsureImageReplicated(context.Background(), "global.registry.com/ghcr-mirror/cloudoperators/greenhouse:v1.0")
		Expect(err).NotTo(HaveOccurred())
		Expect(replicatedRef).To(Equal("global.registry.com/ghcr-mirror/cloudoperators/greenhouse:v1.0"))
		Expect(fetchedRef).To(Equal("primary.registry.com/ghcr-mirror/cloudoperators/greenhouse:v1.0"))
		Expect(manifest).NotTo(BeEmpty())
	})

	It("should replicate directly when ref is already on primaryMirror", func() {
		var fetchedRef string
		mirror := newTestImageMirror(mirrorConfig, func(ref string, opts ...crane.Option) ([]byte, error) {
			fetchedRef = ref
			return []byte(`{"test":"manifest"}`), nil
		})

		replicatedRef, manifest, err := mirror.EnsureImageReplicated(context.Background(), "primary.registry.com/ghcr-mirror/cloudoperators/greenhouse:v1.0")
		Expect(err).NotTo(HaveOccurred())
		Expect(replicatedRef).To(Equal("primary.registry.com/ghcr-mirror/cloudoperators/greenhouse:v1.0"))
		Expect(fetchedRef).To(Equal("primary.registry.com/ghcr-mirror/cloudoperators/greenhouse:v1.0"))
		Expect(manifest).NotTo(BeEmpty())
	})

	It("should return empty when no mirror relationship exists", func() {
		fetchCount := 0
		mirror := newTestImageMirror(mirrorConfig, func(ref string, opts ...crane.Option) ([]byte, error) {
			fetchCount++
			return []byte("{}"), nil
		})

		replicatedRef, manifest, err := mirror.EnsureImageReplicated(context.Background(), "registry.k8s.io/pause:3.9")
		Expect(err).NotTo(HaveOccurred())
		Expect(replicatedRef).To(BeEmpty())
		Expect(manifest).To(BeNil())
		Expect(fetchCount).To(Equal(0))
	})

	It("should pass platform option to manifest fetcher", func() {
		var receivedOpts []crane.Option
		mirror := newTestImageMirror(mirrorConfig, func(ref string, opts ...crane.Option) ([]byte, error) {
			receivedOpts = opts
			return []byte(`{}`), nil
		})

		_, _, err := mirror.EnsureImageReplicated(context.Background(), "ghcr.io/cloudoperators/greenhouse:main")
		Expect(err).NotTo(HaveOccurred())
		Expect(len(receivedOpts)).To(BeNumerically(">=", 2))
	})

	It("should propagate replication errors", func() {
		mirror := newTestImageMirror(mirrorConfig, func(ref string, opts ...crane.Option) ([]byte, error) {
			return nil, errors.New("connection refused")
		})

		_, _, err := mirror.EnsureImageReplicated(context.Background(), "ghcr.io/cloudoperators/greenhouse:main")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("connection refused"))
	})
})

var _ = Describe("BuildImageTransformations", func() {
	It("should return transforms for upstream images", func() {
		mirror := newTestImageMirror(mirrorConfig, nil)

		manifests := `
containers:
- image: ghcr.io/cloudoperators/greenhouse:main
- image: docker.io/library/nginx:latest
`
		transforms := mirror.BuildImageTransformations(manifests)
		Expect(transforms).To(HaveLen(2))
		Expect(transforms).To(ContainElement(ImageTransform{
			Original: "docker.io/library/nginx",
			Mirrored: "global.registry.com/dockerhub-mirror/library/nginx",
		}))
		Expect(transforms).To(ContainElement(ImageTransform{
			Original: "ghcr.io/cloudoperators/greenhouse",
			Mirrored: "global.registry.com/ghcr-mirror/cloudoperators/greenhouse",
		}))
	})

	It("should skip images already on a mirror", func() {
		mirror := newTestImageMirror(mirrorConfig, nil)

		manifests := `
containers:
- image: global.registry.com/ghcr-mirror/cloudoperators/greenhouse:main
`
		transforms := mirror.BuildImageTransformations(manifests)
		Expect(transforms).To(BeEmpty())
	})

	It("should skip images without configured mirror", func() {
		mirror := newTestImageMirror(mirrorConfig, nil)

		manifests := `
containers:
- image: registry.k8s.io/pause:3.9
`
		transforms := mirror.BuildImageTransformations(manifests)
		Expect(transforms).To(BeEmpty())
	})

	It("should handle mixed upstream and already-mirrored images", func() {
		mirror := newTestImageMirror(mirrorConfig, nil)

		manifests := `
containers:
- image: ghcr.io/cloudoperators/greenhouse:main
- image: global.registry.com/ghcr-mirror/already-mirrored:v1
- image: registry.k8s.io/pause:3.9
`
		transforms := mirror.BuildImageTransformations(manifests)
		Expect(transforms).To(HaveLen(1))
		Expect(transforms[0].Original).To(Equal("ghcr.io/cloudoperators/greenhouse"))
	})
})

var _ = Describe("ReplicateOCIArtifacts", func() {
	It("should replicate images successfully", func() {
		fetchedRefs := make([]string, 0)
		mirror := newTestImageMirror(mirrorConfig, func(ref string, opts ...crane.Option) ([]byte, error) {
			fetchedRefs = append(fetchedRefs, ref)
			return []byte("{}"), nil
		})

		manifests := `
containers:
- image: ghcr.io/cloudoperators/greenhouse:main
- image: docker.io/library/nginx:latest
`
		replicated, err := mirror.ReplicateOCIArtifacts(context.Background(), nil, manifests)
		Expect(err).NotTo(HaveOccurred())
		Expect(replicated).To(HaveLen(2))
		Expect(fetchedRefs).To(HaveLen(2))
		Expect(fetchedRefs).To(ContainElement("primary.registry.com/dockerhub-mirror/library/nginx:latest"))
		Expect(fetchedRefs).To(ContainElement("primary.registry.com/ghcr-mirror/cloudoperators/greenhouse:main"))
	})

	It("should replicate via primaryMirror when ref is on baseDomain", func() {
		var fetchedRef string
		mirror := newTestImageMirror(mirrorConfig, func(ref string, opts ...crane.Option) ([]byte, error) {
			fetchedRef = ref
			return []byte("{}"), nil
		})

		manifests := `
containers:
- image: global.registry.com/ghcr-mirror/cloudoperators/greenhouse:main
`
		replicated, err := mirror.ReplicateOCIArtifacts(context.Background(), nil, manifests)
		Expect(err).NotTo(HaveOccurred())
		Expect(replicated).To(HaveLen(1))
		Expect(fetchedRef).To(Equal("primary.registry.com/ghcr-mirror/cloudoperators/greenhouse:main"))
	})

	It("should skip already replicated images", func() {
		fetchCount := 0
		mirror := newTestImageMirror(mirrorConfig, func(ref string, opts ...crane.Option) ([]byte, error) {
			fetchCount++
			return []byte("{}"), nil
		})

		manifests := `
containers:
- image: ghcr.io/cloudoperators/greenhouse:main
- image: docker.io/library/nginx:latest
`
		alreadyReplicated := []string{"ghcr.io/cloudoperators/greenhouse:main"}
		replicated, err := mirror.ReplicateOCIArtifacts(context.Background(), alreadyReplicated, manifests)
		Expect(err).NotTo(HaveOccurred())
		Expect(replicated).To(HaveLen(2))
		Expect(fetchCount).To(Equal(1))
	})

	It("should skip images without configured mirror", func() {
		fetchCount := 0
		mirror := newTestImageMirror(mirrorConfig, func(ref string, opts ...crane.Option) ([]byte, error) {
			fetchCount++
			return []byte("{}"), nil
		})

		manifests := `
containers:
- image: registry.k8s.io/pause:3.9
`
		replicated, err := mirror.ReplicateOCIArtifacts(context.Background(), nil, manifests)
		Expect(err).NotTo(HaveOccurred())
		Expect(replicated).To(BeEmpty())
		Expect(fetchCount).To(Equal(0))
	})

	It("should return partial results and error on failure", func() {
		mirror := newTestImageMirror(mirrorConfig, func(ref string, opts ...crane.Option) ([]byte, error) {
			if ref == "primary.registry.com/ghcr-mirror/cloudoperators/greenhouse:main" {
				return nil, errors.New("connection refused")
			}
			return []byte("{}"), nil
		})

		manifests := `
containers:
- image: ghcr.io/cloudoperators/greenhouse:main
- image: docker.io/library/nginx:latest
`
		replicated, err := mirror.ReplicateOCIArtifacts(context.Background(), nil, manifests)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("connection refused"))
		Expect(replicated).To(ContainElement("docker.io/library/nginx:latest"))
		Expect(replicated).NotTo(ContainElement("ghcr.io/cloudoperators/greenhouse:main"))
	})

	It("should return nil when no images in manifests", func() {
		mirror := newTestImageMirror(mirrorConfig, func(ref string, opts ...crane.Option) ([]byte, error) {
			return []byte("{}"), nil
		})

		manifests := `
apiVersion: v1
kind: ConfigMap
`
		replicated, err := mirror.ReplicateOCIArtifacts(context.Background(), []string{"some-image:latest"}, manifests)
		Expect(err).NotTo(HaveOccurred())
		Expect(replicated).To(BeNil())
	})
})

var _ = Describe("buildMirroredOCIRef", func() {
	mirror := newTestImageMirror(mirrorConfig, nil)

	DescribeTable("should build correct mirrored refs",
		func(imageRef, expected string) {
			Expect(mirror.buildMirroredOCIRef(imageRef)).To(Equal(expected))
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
