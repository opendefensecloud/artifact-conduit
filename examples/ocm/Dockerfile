# ocm CLI + shell tooling (bash, sed, jq, curl) used by the OCM example workflows.
# Built and published by .github/workflows/docker.yaml as ghcr.io/opendefensecloud/ocm-tools
FROM alpine:3@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6

# Version is pinned by .github/workflows/docker.yaml's ocm-tools matrix entry — no local default, to avoid a second source of truth.
ARG OCM_VERSION
RUN apk add --no-cache bash sed jq curl
RUN : "${OCM_VERSION:?OCM_VERSION build-arg is required}" && curl -fsSL https://ocm.software/install.sh | bash

ENTRYPOINT ["bash"]

