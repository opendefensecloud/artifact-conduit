# ocm CLI + shell tooling (bash, sed, jq, curl) used by the OCM example workflows.
# Built and published by .github/workflows/docker.yaml as ghcr.io/opendefensecloud/ocm-tools
FROM alpine:3@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

# Version is pinned by .github/workflows/docker.yaml's ocm-tools matrix entry — no local default, to avoid a second source of truth.
ARG OCM_VERSION
RUN apk add --no-cache bash sed jq curl
RUN : "${OCM_VERSION:?OCM_VERSION build-arg is required}" && curl -fsSL https://ocm.software/install.sh | bash

ENTRYPOINT ["bash"]

