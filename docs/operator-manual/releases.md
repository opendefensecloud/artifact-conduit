# Releases

You can find the most recent version in the [GitHub Releases](https://github.com/opendefensecloud/artifact-conduit/releases).

## Cutting a release

Releases are cut with the [Release workflow](https://github.com/opendefensecloud/artifact-conduit/blob/main/.github/workflows/release.yaml), which calls dev-kit's shared release workflow to compute the version and notes from [Conventional Commit](https://www.conventionalcommits.org/en/v1.0.0/) messages. There is no release PR and no manual tagging.

1. **Commits land on `main`.** The Release workflow runs on every push and creates or updates the pending **draft** release in place, including its signed artefacts. It never tags or publishes. If nothing releasable landed, it leaves the draft untouched.
2. **Run the Release workflow manually** when you want to inspect or refresh the draft. `dry_run` is checked by default and only renders the version and notes. Re-run with it unchecked to update the draft and its assets. `create_tag` is off by default, so publishing the draft creates the tag. Check it only if you need the tag during the run.
3. **Publish the draft** via the GitHub UI or `gh release edit <tag> --draft=false`. Publishing creates the tag if it does not exist yet, makes the release immutable, and triggers the Docker, Helm, and docs workflows.
4. **Publish-triggered workflows** build and publish images, charts, and versioned docs.

A draft is used because releases are immutable: assets must be attached before publishing. The workflow updates the same draft on each push; it does not create one draft per push.

## Release notes

Notes are generated from the merge commits on `main` and grouped by conventional type. Every PR must land as a merge commit whose subject is the PR title, and the PR title must be a conventional commit. Squash merges, rebase merges, and direct commits cannot be rendered and make the Release workflow fail unless `force` is used.

`test:`, `style:`, and `build:` merges are omitted. A breaking change is signalled with `!` in the PR title.

## Versioning

Versions are expressed as `x.y.z`, where `x` is the major version, `y` is the minor version, and `z` is the patch version, following Semantic Versioning terminology.

| Commit type | Version bump |
|---|---|
| `fix:`, `perf:` | patch |
| `feat:` | minor |
| `feat!:` (any type with `!`) | minor while the major is `0` |
| `chore:`, `docs:`, `ci:`, … | none on their own |

Breaking changes bump the minor while the major is `0`; from `1.0.0` onward they bump the major. Pass an explicit `version` input to override the computed version. Prereleases (versions containing `-`) are marked as prereleases automatically.

## Draft and tag rules

The workflow keeps one pending draft. A push updates it in place (title, notes, prerelease flag, and target commit); if the computed version moves, the draft is renamed.

A draft whose tag already exists is blocked: publish it to ship, or delete the draft to abandon it. Where release tags are protected, a tag cannot be moved or deleted; once it exists (for example after a run with `create_tag`), publish that draft or cut the next patch.

## Recovering a failed asset upload

If the draft was created but the artefacts failed to upload, run **Release** again with `retry_assets_for` set to the draft's tag and uncheck `dry_run`. The retry skips version resolution and notes and only rebuilds and re-attaches the artefacts. It refuses if the release is missing or already published. Uploads use `--clobber`, so re-runs are idempotent.

Leave `retry_assets_for` empty for every normal release.

## Verifying release artefacts

Every release artefact (each binary and `checksums.txt`) is keylessly signed with [cosign](https://docs.sigstore.dev/) during the release workflow. For each artefact `<file>`, a matching Sigstore bundle `<file>.sigstore.json` (containing the signature, certificate, and transparency-log proof) is published next to it on the GitHub Release page.

To verify a downloaded binary, install [cosign](https://docs.sigstore.dev/system_config/installation/) and run `cosign verify-blob`, pointing it at the artefact's `.sigstore.json` bundle:

```bash
cosign verify-blob \
  --bundle arc-apiserver-linux-amd64.sigstore.json \
  --certificate-identity-regexp 'https://github.com/opendefensecloud/artifact-conduit/\.github/workflows/release\.yaml@.*' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  arc-apiserver-linux-amd64
```

The same command works for `checksums.txt` and any other artefact — swap the file names accordingly. A successful run prints `Verified OK`.

Release binaries additionally carry a [build-provenance attestation](https://docs.github.com/en/actions/security-guides/using-artifact-attestations-to-establish-provenance-for-builds), which can be verified with the GitHub CLI:

```bash
gh attestation verify arc-apiserver-linux-amd64 --repo opendefensecloud/artifact-conduit
```
