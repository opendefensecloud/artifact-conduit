# Build the manager binary
FROM --platform=$BUILDPLATFORM golang:1.26@sha256:3aff6657219a4d9c14e27fb1d8976c49c29fddb70ba835014f477e1c70636647 AS builder

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
FROM gcr.io/distroless/static:nonroot@sha256:f7f8f729987ad0fdf6b05eeeae94b26e6a0f613bdf46feea7fc40f7bd72953e6 AS apiserver
WORKDIR /
COPY --from=apiserver-builder /workspace/bin/arc-apiserver .
USER 65532:65532
ENTRYPOINT ["/arc-apiserver"]

FROM gcr.io/distroless/static:nonroot@sha256:f7f8f729987ad0fdf6b05eeeae94b26e6a0f613bdf46feea7fc40f7bd72953e6 AS manager
WORKDIR /
COPY --from=manager-builder /workspace/bin/arc-controller-manager .
USER 65532:65532
ENTRYPOINT ["/arc-controller-manager"]

# Let's create a custom image with relevant plugins for local serving of docs
FROM squidfunk/mkdocs-material@sha256:51b87149d227691486b5f08993d28c65ca7e4990010664b697265b8e6fcd5287 AS mkdocs

COPY ./docs/requirements.txt /requirements.txt

RUN pip install -r /requirements.txt

CMD ["serve", "--dev-addr=0.0.0.0:8000", "--livereload"]
