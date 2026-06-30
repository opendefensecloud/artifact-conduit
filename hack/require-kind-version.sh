#!/usr/bin/env bash
# TEMPORARY WORKAROUND — delete this script (and its call in the Makefile
# 'test-e2e' target) once devenv/flake.nix ships kind >= $KIND_MIN_VERSION.
#
# kindest/node images for K8s 1.36+ ship a containerd whose config is version 4,
# which only kind >= 0.32.0 can load images into. Older kind fails late and
# cryptically ("unknown containerd config version: 4 (supported versions: 2 and
# 3)") during `kind load`. Cluster *creation* works on older kind, and CI pulls
# images from a registry (E2E_IMAGE_SOURCE=ghcr, no `kind load`), so callers
# guard this on the local image-load path only and fail early with actionable
# instructions.
#
# Optional env:
#   KIND               path to kind binary (default: kind)
#   KIND_MIN_VERSION   minimum required kind version (default: 0.32.0)

set -euo pipefail

KIND="${KIND:-kind}"
KIND_MIN_VERSION="${KIND_MIN_VERSION:-0.32.0}"

kind_ver=$("$KIND" version 2>/dev/null | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' | head -1 | tr -d v)

# Can't determine the version (unexpected kind build) → don't block.
[ -n "$kind_ver" ] || exit 0

# sort -V ascending: if the smallest line isn't the minimum, the installed
# version is older than the minimum.
if [ "$(printf '%s\n%s\n' "$KIND_MIN_VERSION" "$kind_ver" | sort -V | head -1)" != "$KIND_MIN_VERSION" ]; then
  cat >&2 <<EOF
ERROR: kind $kind_ver is too old to load images into a K8s 1.36+ node.
       kind >= $KIND_MIN_VERSION is required; older versions fail 'kind load'
       with "unknown containerd config version: 4".

       Install a new-enough kind and point make at it, e.g.:

         go install sigs.k8s.io/kind@v$KIND_MIN_VERSION
         make test-e2e KIND=\$(go env GOPATH)/bin/kind
EOF
  exit 1
fi
