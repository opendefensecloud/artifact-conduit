# ocm CLI + shell tooling (bash, sed, jq, curl) used by the OCM example workflows.
# Built and published by .github/workflows/docker.yaml as ghcr.io/opendefensecloud/ocm-tools
FROM alpine:3@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a

# Version is pinned by .github/workflows/docker.yaml's ocm-tools matrix entry — no local default, to avoid a second source of truth.
ARG OCM_VERSION
RUN apk add --no-cache bash sed jq curl
RUN : "${OCM_VERSION:?OCM_VERSION build-arg is required}" && curl -fsSL https://ocm.software/install.sh | bash

ENTRYPOINT ["bash"]

