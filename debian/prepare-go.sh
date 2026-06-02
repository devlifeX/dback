#!/bin/bash
# Ensure Go >= 1.25 for module builds on Launchpad (older Ubuntu releases ship older Go).
set -euo pipefail

REQUIRED_GO="1.25"
GO_TARBALL_VERSION="${GO_TARBALL_VERSION:-1.25.0}"
GO_DIR="${DEBIAN_GO_DIR:-debian/go}"

go_version_ok() {
	if ! command -v go >/dev/null 2>&1; then
		return 1
	fi
	local ver="${GOVERSION:-$(go env GOVERSION 2>/dev/null | sed 's/^go//')}"
	if [ -z "$ver" ]; then
		return 1
	fi
	dpkg --compare-versions "$ver" ge "$REQUIRED_GO"
}

if go_version_ok; then
	go version
	exit 0
fi

arch="$(dpkg-architecture -qDEB_BUILD_GNU_CPU 2>/dev/null || uname -m)"
case "$arch" in
	x86_64 | amd64) goarch="amd64" ;;
	aarch64 | arm64) goarch="arm64" ;;
	*)
		echo "prepare-go: unsupported architecture: $arch" >&2
		exit 1
		;;
esac

url="https://go.dev/dl/go${GO_TARBALL_VERSION}.linux-${goarch}.tar.gz"
echo "prepare-go: bootstrapping Go ${GO_TARBALL_VERSION} (${goarch}) from ${url}" >&2

rm -rf "$GO_DIR"
mkdir -p "$GO_DIR"
curl -fsSL "$url" | tar -xz -C "$GO_DIR"

export GOROOT="${GO_DIR}/go"
export PATH="${GOROOT}/bin:${PATH}"
export GOPATH="${GO_DIR}/gopath"
export GOCACHE="${GO_DIR}/cache"
mkdir -p "$GOPATH" "$GOCACHE"

go version
