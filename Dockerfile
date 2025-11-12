# Build the manager binary
FROM --platform=$BUILDPLATFORM golang:1.25 AS builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# Cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY api/ api/
COPY client-go/ client-go/
COPY cmd/ cmd/
COPY pkg/ pkg/

ARG TARGETOS
ARG TARGETARCH

RUN mkdir bin


FROM builder AS apiserver-builder
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH GO111MODULE=on go build -ldflags="-s -w" -a -o bin/arc-apiserver ./cmd/arc-apiserver


FROM builder AS manager-builder
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH GO111MODULE=on go build -ldflags="-s -w" -a -o bin/arc-controller-manager ./cmd/arc-controller-manager


FROM builder AS arcctl-builder
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH GO111MODULE=on go build -ldflags="-s -w" -a -o bin/arcctl ./cmd/arcctl

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot AS apiserver
WORKDIR /
COPY --from=apiserver-builder /workspace/bin/arc-apiserver .
USER 65532:65532
ENTRYPOINT ["/arc-apiserver"]

FROM gcr.io/distroless/static:nonroot AS manager
WORKDIR /
COPY --from=manager-builder /workspace/bin/arc-controller-manager .
USER 65532:65532
ENTRYPOINT ["/arc-controller-manager"]

FROM gcr.io/distroless/static:nonroot AS arcctl
WORKDIR /
COPY --from=arcctl-builder /workspace/bin/arcctl .
USER 65532:65532
ENTRYPOINT ["/arcctl"]
