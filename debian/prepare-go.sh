#!/bin/bash
# Verify system Go is available for offline PPA builds (no network downloads).
set -euo pipefail

export GOTOOLCHAIN=local

MIN_GO="1.21"

if ! command -v go >/dev/null 2>&1; then
	echo "prepare-go: go not found; install golang-go from Build-Depends." >&2
	exit 1
fi

ver="$(go env GOVERSION 2>/dev/null | sed 's/^go//')"
if [ -z "$ver" ]; then
	echo "prepare-go: could not determine Go version." >&2
	exit 1
fi

if ! dpkg --compare-versions "$ver" ge "$MIN_GO" 2>/dev/null; then
	echo "prepare-go: Go ${ver} is older than required ${MIN_GO}." >&2
	exit 1
fi

if [ ! -f vendor/modules.txt ]; then
	echo "prepare-go: vendor/modules.txt missing; run 'go mod vendor' before building." >&2
	exit 1
fi

go version
