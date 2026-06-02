#!/usr/bin/env bash
# Build a Debian source package (.dsc + .changes) for Launchpad PPA upload.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

PPA="${PPA:-ppa:devlifex/dback}"
UPLOAD="${UPLOAD:-0}"
# SIGN=auto (default): sign when a secret key exists for the changelog maintainer email
# SIGN=1: require signing; SIGN=0 or UNSIGNED=1: build without signing
SIGN="${SIGN:-auto}"
# Optional override; otherwise fingerprint is detected from the maintainer email
GPG_KEY_ID="${GPG_KEY_ID:-}"

need_cmd() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "Missing command: $1" >&2
		echo "Install with: sudo apt install devscripts debhelper build-essential" >&2
		exit 1
	fi
}

gpg_help() {
	cat <<'EOF'
Launchpad requires a signed .changes file. See ppa.md for full setup.

  export DEBFULLNAME="Dariush Vesal"
  export DEBEMAIL="dvworkmail2017@gmail.com"
  echo "default-key E00C906928B7599C" >> ~/.gnupg/gpg.conf
  export GPG_TTY=$(tty)

  UPLOAD=1 ./packaging/build-ppa.sh

Manual sign + upload:

  debsign -k E8B25AD68688EC024359E03FE00C906928B7599C ../dback_VERSION_source.changes
  dput ppa:devlifex/dback ../dback_VERSION_source.changes
EOF
}

signing_key_available() {
	local email="$1"
	gpg --list-secret-keys --keyid-format LONG "$email" 2>/dev/null | grep -q '^sec'
}

gpg_signing_fingerprint() {
	local email="$1"
	gpg --list-secret-keys --with-colons "$email" 2>/dev/null \
		| awk -F: '$1 == "fpr" && $10 != "" { print $10; exit }'
}

dput_with_retry() {
	local ppa="$1"
	local changes="$2"
	local attempt max_attempts=3
	for attempt in $(seq 1 "$max_attempts"); do
		if [ "$attempt" -gt 1 ]; then
			echo "Retrying dput (attempt ${attempt}/${max_attempts})..."
			sleep 5
		fi
		if dput "$ppa" "$changes"; then
			return 0
		fi
	done
	echo "ERROR: dput failed after ${max_attempts} attempts." >&2
	echo "Run manually: dput ${ppa} ${changes}" >&2
	return 1
}

export GPG_TTY="${GPG_TTY:-$(tty 2>/dev/null || true)}"

need_cmd debuild
need_cmd dpkg-parsechangelog

version="$(dpkg-parsechangelog -SVersion)"
maintainer="$(dpkg-parsechangelog --show-field Maintainer)"
maintainer_email="$(sed -nE 's/.*<([^>]+)>.*/\1/p' <<<"$maintainer")"

echo "Building source package for dback ${version}..."

# Avoid lintian "source-is-missing" when a local build artifact is present.
rm -f dback-linux dist/dback-linux

debuild_args=(-S -sa -d -us -uc)
will_sign=false
signing_key=""

case "$SIGN" in
	0 | false | no)
		echo "NOTE: UNSIGNED=1 — output will not be signed (not uploadable to PPA as-is)."
		;;
	1 | true | yes)
		if ! signing_key_available "$maintainer_email"; then
			echo "ERROR: SIGN=1 but no GPG secret key for ${maintainer_email}." >&2
			echo "" >&2
			gpg_help >&2
			exit 1
		fi
		will_sign=true
		signing_key="${GPG_KEY_ID:-$(gpg_signing_fingerprint "$maintainer_email")}"
		;;
	auto | *)
		if signing_key_available "$maintainer_email"; then
			will_sign=true
			signing_key="${GPG_KEY_ID:-$(gpg_signing_fingerprint "$maintainer_email")}"
			echo "GPG key found for ${maintainer_email}; will sign after build."
		else
			echo "WARNING: No GPG secret key for ${maintainer_email}."
			echo "Building unsigned. PPA upload needs signing — see ppa.md"
		fi
		;;
esac

debuild "${debuild_args[@]}"

parent="$(dirname "$ROOT")"
changes="$(ls -1t "${parent}"/dback_"${version}"_source.changes 2>/dev/null | head -n 1 || true)"

if [ -z "$changes" ]; then
	echo "ERROR: could not find ../dback_${version}_source.changes" >&2
	exit 1
fi

if [ "$will_sign" = true ]; then
	need_cmd debsign
	if [ -z "$signing_key" ]; then
		echo "ERROR: could not determine GPG fingerprint for ${maintainer_email}." >&2
		gpg_help >&2
		exit 1
	fi
	echo "Signing with GPG fingerprint ${signing_key}..."
	debsign -k "$signing_key" "$changes"
fi

echo ""
echo "Source package ready:"
echo "  ${changes}"
if [ "$will_sign" = false ]; then
	echo ""
	echo "To sign before upload:"
	echo "  debsign -k E8B25AD68688EC024359E03FE00C906928B7599C ${changes}"
fi
echo ""
echo "Upload to Launchpad PPA:"
echo "  dput ppa:devlifex/dback ${changes}"
echo ""
echo "After the build succeeds, users install with:"
echo "  sudo add-apt-repository ppa:devlifex/dback"
echo "  sudo apt update"
echo "  sudo apt install dback"
echo ""
echo "Full guide: ppa.md"

if [ "$UPLOAD" = "1" ]; then
	need_cmd dput
	if [ "$will_sign" = false ]; then
		echo "" >&2
		echo "ERROR: UPLOAD=1 requires a signed .changes file." >&2
		gpg_help >&2
		exit 1
	fi
	echo ""
	echo "Uploading to ${PPA}..."
	dput_with_retry "$PPA" "$changes"
fi
