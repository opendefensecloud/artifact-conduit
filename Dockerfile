# Build the manager binary
FROM --platform=$BUILDPLATFORM golang:1.27.3@sha256:2920bb124f1a2e12a1d39f0cab756ae0068441ee1da7655dec7f5e374944095b AS builder

WORKDIR /workspace
RUN go env -w GOMODCACHE=/root/.cache/go-build

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN --mount=type=cache,target=/root/.cache/go-build go mod download

# Copy the go source
COPY api/ api/
COPY client-go/ client-go/
COPY cmd/ cmd/
COPY pkg/ pkg/

ARG TARGETOS
ARG TARGETARCH
ARG GO_BUILD_FLAGS

RUN mkdir bin

FROM builder AS apiserver-builder
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH GO111MODULE=on go build -ldflags="-s -w" ${GO_BUILD_FLAGS} -o bin/arc-apiserver ./cmd/arc-apiserver


FROM builder AS manager-builder
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH GO111MODULE=on go build -ldflags="-s -w" ${GO_BUILD_FLAGS} -o bin/arc-controller-manager ./cmd/arc-controller-manager

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot@sha256:25ac8f0c3b2334a9eb05452266bd4697864a8c8ef0ef71e5e6480d7f92868329 AS apiserver
WORKDIR /
COPY --from=apiserver-builder /workspace/bin/arc-apiserver .
USER 65532:65532
ENTRYPOINT ["/arc-apiserver"]

FROM gcr.io/distroless/static:nonroot@sha256:25ac8f0c3b2334a9eb05452266bd4697864a8c8ef0ef71e5e6480d7f92868329 AS manager
WORKDIR /
COPY --from=manager-builder /workspace/bin/arc-controller-manager .
USER 65532:65532
ENTRYPOINT ["/arc-controller-manager"]

# Let's create a custom image with relevant plugins for local serving of docs
FROM squidfunk/mkdocs-material@sha256:dc3b5de76c6e30d18dc0922d4c4a7bed817e5e155f9ff0e594a53eb56e537e64 AS mkdocs

COPY ./docs/requirements.txt /requirements.txt

RUN pip install -r /requirements.txt

CMD ["serve", "--dev-addr=0.0.0.0:8000", "--livereload"]
