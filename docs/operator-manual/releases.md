# Releases

You can find the most recent version in the [GitHub Releases](https://github.com/opendefensecloud/artifact-conduit/releases).

## Versioning

Versions are expressed as `x.y.z`, where `x` is the major version, `y` is the minor version, and `z` is the patch version, following Semantic Versioning terminology.

## Verifying release artefacts

Every release artefact (each binary and `checksums.txt`) is keylessly signed with [cosign](https://docs.sigstore.dev/) during the release workflow. For each artefact `<file>`, a matching `<file>.sig` (signature) and `<file>.pem` (certificate) is published next to it on the GitHub Release page.

To verify a downloaded binary, install [cosign](https://docs.sigstore.dev/system_config/installation/) and run `cosign verify-blob`, pointing it at the artefact's `.pem` and `.sig`:

```bash
cosign verify-blob \
  --certificate arc-apiserver-linux-amd64.pem \
  --signature  arc-apiserver-linux-amd64.sig \
  --certificate-identity-regexp 'https://github.com/opendefensecloud/artifact-conduit/\.github/workflows/release\.yaml@.*' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  arc-apiserver-linux-amd64
```

The same command works for `checksums.txt` and any other artefact — swap the file names accordingly. A successful run prints `Verified OK`.

Release binaries additionally carry a [build-provenance attestation](https://docs.github.com/en/actions/security-guides/using-artifact-attestations-to-establish-provenance-for-builds), which can be verified with the GitHub CLI:

```bash
gh attestation verify arc-apiserver-linux-amd64 --repo opendefensecloud/artifact-conduit
```
