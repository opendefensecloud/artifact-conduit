# ocm CLI + shell tooling (bash, sed, jq, curl) used by the OCM example workflows.
# Built and published by .github/workflows/docker.yaml as ghcr.io/opendefensecloud/ocm-tools
FROM alpine:3@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf

# Version is pinned by .github/workflows/docker.yaml's ocm-tools matrix entry — no local default, to avoid a second source of truth.
ARG OCM_VERSION
RUN apk add --no-cache bash sed jq curl
RUN : "${OCM_VERSION:?OCM_VERSION build-arg is required}" && curl -fsSL https://ocm.software/install.sh | bash

ENTRYPOINT ["bash"]

