# Build the manager binary
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.25.4 AS builder

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0

WORKDIR /workspace
COPY . .

# Build greenhouse operator and tooling.
RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	make action-build CGO_ENABLED=${CGO_ENABLED} GOOS=${TARGETOS} GOARCH=${TARGETARCH}

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM --platform=${BUILDPLATFORM:-linux/amd64} gcr.io/distroless/static:nonroot
LABEL source_repository="https://github.com/cloudoperators/greenhouse"
WORKDIR /
COPY --from=builder /workspace/bin/* .
USER 65532:65532

RUN ["/greenhouse", "--version"]
ENTRYPOINT ["/greenhouse"]
