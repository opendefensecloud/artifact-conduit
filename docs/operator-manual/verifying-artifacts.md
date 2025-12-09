# Verifying Artifacts

!!! note

    Examples taking from PR. Therefore identity will alternate for releases.

## Container images

### Checking signatures

```bash
cosign verify --certificate-identity="https://github.com/opendefensecloud/artifact-conduit/.github/workflows/docker.yaml@refs/pull/137/merge" --certificate-oidc-issuer="https://token.actions.githubusercontent.com" ghcr.io/opendefensecloud/arc-controller-manager:pr-137
```

### Checking attestations

```bash
# CycloneDX BOM/SBOM
cosign verify-attestation \
  --certificate-identity https://github.com/opendefensecloud/artifact-conduit/.github/workflows/docker.yaml@refs/pull/137/merge \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --new-bundle-format \
  --type=https://cyclonedx.org/bom \
  ghcr.io/opendefensecloud/arc-controller-manager:pr-137 | jq -r '.payload | @base64d | fromjson'
# SLSA provenance
cosign verify-attestation \
  --certificate-identity https://github.com/opendefensecloud/artifact-conduit/.github/workflows/docker.yaml@refs/pull/137/merge \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --new-bundle-format \
  --type=https://slsa.dev/provenance/v1 \
  ghcr.io/opendefensecloud/arc-controller-manager:pr-137 | jq -r '.payload | @base64d | fromjson'
```
