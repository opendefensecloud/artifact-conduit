#!/usr/bin/env bash

# Sideload envtest binaries from canonical upstream sources (dl.k8s.io and
# etcd-io GitHub releases) for K8s versions that controller-tools hasn't
# packaged into its envtest-releases index yet. envtest's release cadence
# lags K8s releases sporadically, so a freshly-pinned ENVTEST_K8S_VERSION may
# not be downloadable via `setup-envtest use` until controller-tools catches
# up. This populates setup-envtest's cache directly from upstream instead.
#
# dl.k8s.io has all versions but only Linux builds. controller-tools has
# Darwin builds, but not for all versions of Kubernetes and etcd, so running
# a specific version under Darwin might not be possible.
#
# So it should:
# - check K8S_VERSION, use setup-envtest binaries if already cached
# - if not, fall back to dl.k8s.io (which has no Darwin builds) on Linux,
#   or defer to `setup-envtest use` on non-Linux hosts
#
# Idempotent — exits 0 immediately if setup-envtest already has the version
# cached, so this is safe as a `test` prerequisite that runs every invocation.
#
# Required env / args:
#   $1            K8s version (e.g. 1.36.0)
#   SETUP_ENVTEST path to setup-envtest binary
#   BIN_DIR       cache directory (matches the Makefile's $(LOCALBIN))
#
# Optional env:
#   YQ            path to yq (defaults to `yq` on PATH)

set -o errexit
set -o nounset
set -o pipefail

K8S_VERSION="${1:?Usage: $0 <k8s-version>}"

# Accept either "1.32.1" or "v1.32.1" — strip an optional leading "v" so
# URL construction (which adds its own "v") doesn't end up with "vv1.32.1".
K8S_VERSION="${K8S_VERSION#v}"

: "${SETUP_ENVTEST:?SETUP_ENVTEST must be set}"
: "${BIN_DIR:?BIN_DIR must be set}"

YQ="${YQ:-yq}"

# Idempotency — bail out if setup-envtest already finds this version in cache.
if "$SETUP_ENVTEST" use "$K8S_VERSION" --bin-dir "$BIN_DIR" -i -p path >/dev/null 2>&1; then
	exit 0
fi

# Detect host. dl.k8s.io publishes kube-apiserver only for linux server
# platforms (darwin/windows return 404). controller-tools cross-compiles its
# own darwin/windows envtest archives from K8s source — we can't replicate
# that from a shell script, so on non-linux hosts we defer to vanilla
# `setup-envtest use` (which downloads from controller-tools' index). It
# succeeds whenever the requested version is in the index; only when the
# index has no entry do we have to give up and ask the dev to pin.
os=$(uname -s | tr '[:upper:]' '[:lower:]')

if [ "$os" != "linux" ]; then
	if "$SETUP_ENVTEST" use "$K8S_VERSION" --bin-dir "$BIN_DIR" -p path >/dev/null 2>&1; then
		exit 0
	fi

	cat >&2 <<EOF
envtest-sideload: ${os} is supported via controller-tools' pre-packaged
archives, but '${K8S_VERSION}' is not in their index and dl.k8s.io has
no kube-apiserver binary for non-linux platforms.

List available versions: ${SETUP_ENVTEST} list

Then pin ENVTEST_K8S_VERSION to one of those, or run on Linux where
this script can sideload the upstream K8s/etcd binaries directly.
EOF
	exit 1
fi

case "$(uname -m)" in
	x86_64) arch=amd64 ;;
	aarch64 | arm64) arch=arm64 ;;
	*) echo "envtest-sideload: unsupported arch $(uname -m)" >&2; exit 1 ;;
esac

# Wrap curl with retries + timeouts so transient network blips don't fail
# the whole sideload. --retry-all-errors covers HTTP 5xx (curl 7.71+, 2020).
# --max-time bounds a single request; bytes are ~150 MB max (kube-apiserver),
# 300 s is plenty even on slow runners.
fetch() {
	curl --fail --silent --show-error --location \
		--retry 3 --retry-delay 2 --retry-all-errors \
		--connect-timeout 10 --max-time 300 "$@"
}

# Look up the etcd version K8s ships with from its build/dependencies.yaml.
# Self-updating per K8s release.
deps_url="https://raw.githubusercontent.com/kubernetes/kubernetes/v${K8S_VERSION}/build/dependencies.yaml"

etcd_version=$(fetch "$deps_url" \
	| "$YQ" '.dependencies[] | select(.name == "etcd") | .version')

if [[ -z "$etcd_version" ]]; then
	echo "envtest-sideload: could not resolve etcd version from $deps_url" >&2
	exit 1
fi

echo "envtest-sideload: K8s ${K8S_VERSION} ships with etcd ${etcd_version} (${os}/${arch})"

stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT

# --- K8s binaries (kube-apiserver, kubectl) --------------------------------
#
# Direct per-binary downloads avoid pulling the ~600 MB server tarball when we
# only need two binaries (~150 MB + ~50 MB combined).
#
# NOTE: If a future K8s release ships a new server binary that envtest
# expects, envtest will fail to start the control plane and the test run will
# surface the missing binary. Add the new name to the loop below — the server
# tarball at dl.k8s.io/v${VER}/kubernetes-server-${os}-${arch}.tar.gz is the
# canonical index of what's available per release if you need to discover the
# exact filename.
k8s_base="https://dl.k8s.io/release/v${K8S_VERSION}/bin/${os}/${arch}"

mkdir -p "$stage/k8s-bin"

for bin in kube-apiserver kubectl; do
	echo "envtest-sideload: downloading ${k8s_base}/${bin}"
	fetch -o "$stage/k8s-bin/$bin" "$k8s_base/$bin"
	fetch -o "$stage/k8s-bin/$bin.sha256" "$k8s_base/$bin.sha256"

	# The .sha256 sibling is just the hex digest; pair with the filename ourselves.
	(cd "$stage/k8s-bin" && printf '%s %s\n' "$(cat "$bin.sha256")" "$bin" | sha256sum -c -)

	chmod +x "$stage/k8s-bin/$bin"
done

# --- etcd tarball (etcd binary) --------------------------------------------

etcd_dir="etcd-v${etcd_version}-${os}-${arch}"
etcd_tar="${etcd_dir}.tar.gz"
etcd_base="https://github.com/etcd-io/etcd/releases/download/v${etcd_version}"

echo "envtest-sideload: downloading ${etcd_base}/${etcd_tar}"

fetch -o "$stage/$etcd_tar" "$etcd_base/$etcd_tar"
fetch -o "$stage/SHA256SUMS" "$etcd_base/SHA256SUMS"

# etcd's SHA256SUMS lists `<hash> <filename>` for every platform tarball;
# grep our specific filename and pass to sha256sum -c.
(cd "$stage" && grep " ${etcd_tar}$" SHA256SUMS | sha256sum -c -)

tar -C "$stage" -xzf "$stage/$etcd_tar" "$etcd_dir/etcd"

# --- Assemble the sideload tarball with the three binaries at top level ---

sideload_tar="$stage/sideload.tar.gz"

tar -czf "$sideload_tar" \
	-C "$stage/k8s-bin" kube-apiserver kubectl \
	-C "$stage/$etcd_dir" etcd

# Feed it to setup-envtest. --os/--arch matter — sideload writes the cache
# entry keyed on platform.
echo "envtest-sideload: sideloading ${K8S_VERSION} into ${BIN_DIR}"

"$SETUP_ENVTEST" sideload "$K8S_VERSION" \
	--os "$os" --arch "$arch" --bin-dir "$BIN_DIR" \
	< "$sideload_tar"

# Sanity-check that setup-envtest can now find it.
"$SETUP_ENVTEST" use "$K8S_VERSION" --bin-dir "$BIN_DIR" -i -p path >/dev/null

echo "envtest-sideload: done"
