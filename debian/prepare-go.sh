#!/bin/bash
# Verify system Go is available for offline PPA builds (no network downloads).
set -euo pipefail

export GOTOOLCHAIN=local

MIN_GO="1.22"

go_version() {
	local bin="$1"
	"$bin" env GOVERSION 2>/dev/null | sed 's/^go//'
}

resolve_go_bin() {
	local candidate ver
	for candidate in go /usr/lib/go-1.22/bin/go /usr/lib/go-1.21/bin/go; do
		if command -v "$candidate" >/dev/null 2>&1 || [ -x "$candidate" ]; then
			ver="$(go_version "$candidate")"
			if [ -n "$ver" ] && dpkg --compare-versions "$ver" ge "$MIN_GO" 2>/dev/null; then
				echo "$candidate"
				return 0
			fi
		fi
	done
	return 1
}

GO_BIN="$(resolve_go_bin || true)"
if [ -z "$GO_BIN" ]; then
	echo "prepare-go: no Go >= ${MIN_GO} found (install golang-1.22-go or golang-go from Build-Depends)." >&2
	exit 1
fi

export PATH="$(dirname "$GO_BIN"):${PATH}"

if [ ! -f vendor/modules.txt ]; then
	echo "prepare-go: vendor/modules.txt missing in source tree." >&2
	echo "prepare-go: run ./packaging/build-ppa.sh (vendors deps into tarball, not git)." >&2
	exit 1
fi

echo "prepare-go: using $(command -v go)"
go version
