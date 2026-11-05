# ocm CLI + shell tooling (bash, sed, jq, curl) used by the OCM example workflows.
# Built and published by .github/workflows/docker.yaml as ghcr.io/opendefensecloud/ocm-tools
FROM alpine:3@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf

# Version is pinned by .github/workflows/docker.yaml's ocm2-tools matrix entry — no local default, to avoid a second source of truth.
ARG OCM_VERSION
ARG TARGETARCH
RUN apk add --no-cache bash sed jq curl
# The legacy install.sh installs OCM v1. Download the v2 release explicitly.
RUN : "${OCM_VERSION:?OCM_VERSION build-arg is required}" \
    && TARGETARCH="${TARGETARCH:-$(uname -m | sed 's/^x86_64$/amd64/; s/^aarch64$/arm64/')}" \
    && curl -fsSL "https://github.com/open-component-model/open-component-model/releases/download/v${OCM_VERSION}/ocm-linux-${TARGETARCH}" -o /usr/local/bin/ocm \
    && chmod +x /usr/local/bin/ocm \
    && ocm version

ENTRYPOINT ["bash"]
