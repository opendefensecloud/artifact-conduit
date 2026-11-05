# ocm CLI + shell tooling (bash, sed, jq, curl) used by the OCM example workflows.
# Built and published by .github/workflows/docker.yaml as ghcr.io/opendefensecloud/ocm2-tools
FROM alpine:3@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a

# Version is pinned by .github/workflows/docker.yaml's ocm2-tools matrix entry — no local default, to avoid a second source of truth.
ARG OCM_VERSION
ARG TARGETARCH
RUN apk add --no-cache bash sed jq curl
# The legacy install.sh installs OCM v1. Download the v2 release explicitly.
RUN : "${OCM_VERSION:?OCM_VERSION build-arg is required}" \
    && curl -fsSL "https://github.com/open-component-model/open-component-model/releases/download/v${OCM_VERSION}/ocm-linux-${TARGETARCH}" -o /usr/local/bin/ocm \
    && chmod +x /usr/local/bin/ocm \
    && ocm version

ENTRYPOINT ["bash"]

