# Build the manager binary
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.23 AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
COPY . .
# Build greenhouse operator and tooling.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -v -a -o greenhouse ./cmd/greenhouse

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
LABEL source_repository="https://github.com/cloudoperators/greenhouse"
WORKDIR /
COPY --from=builder /workspace/greenhouse .
USER 65532:65532

RUN ["/greenhouse", "--version"]
ENTRYPOINT ["/greenhouse"]
