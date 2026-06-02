#!/usr/bin/env bash
# Build a Debian source package (.dsc + .changes) for Launchpad PPA upload.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

PPA="${PPA:-ppa:devlifex/dback}"
UPLOAD="${UPLOAD:-0}"
SIGN="${SIGN:-auto}"
GPG_KEY_ID="${GPG_KEY_ID:-E8B25AD68688EC024359E03FE00C906928B7599C}"
LOCK_FILE="${ROOT}/.ppa-build.lock"
STAGING_DIR=""
STAGED_CHANGES=""

need_cmd() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "Missing command: $1" >&2
		echo "Install with: sudo apt install devscripts debhelper build-essential" >&2
		exit 1
	fi
}

gpg_help() {
	cat <<'EOF'
Launchpad requires signed .dsc and .changes files. See ppa.md.

  export DEBFULLNAME="Dariush Vesal"
  export DEBEMAIL="dvworkmail2017@gmail.com"
  export GPG_TTY=$(tty)
  echo "default-key E8B25AD68688EC024359E03FE00C906928B7599C" >> ~/.gnupg/gpg.conf

  UPLOAD=1 ./packaging/build-ppa.sh
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

clean_workspace() {
	echo "Cleaning local build artifacts..."
	rm -f dback-linux dist/dback-linux
	rm -rf debian/dback debian/.debhelper debian/debhelper-build-stamp debian/files
	if [ -f debian/rules ]; then
		fakeroot debian/rules clean >/dev/null 2>&1 || true
	fi
}

cleanup_staging() {
	if [ -n "${STAGING_DIR:-}" ] && [ -d "$STAGING_DIR" ]; then
		rm -rf "$STAGING_DIR"
		STAGING_DIR=""
	fi
}

# List basenames of files referenced in the Files: section of a .changes file.
files_from_changes() {
	local changes="$1" line
	while IFS= read -r line; do
		[[ "$line" =~ ^[[:space:]]+[0-9a-f]{32}[[:space:]]+[0-9]+[[:space:]]+[^[:space:]]+[[:space:]]+[^[:space:]]+[[:space:]]+([^[:space:]]+) ]] || continue
		echo "${BASH_REMATCH[1]}"
	done < <(awk '/^Files:/,/^$/' "$changes" | tail -n +2)
}

# Copy signed upload artifacts into an isolated directory so dput cannot read
# files that might be overwritten in the parent directory during concurrent builds.
stage_upload_artifacts() {
	local changes="$1"
	local src_dir="$2"
	local f

	STAGING_DIR="$(mktemp -d "${src_dir}/.dback-upload-stage.XXXXXX")"

	for f in $(files_from_changes "$changes"); do
		if [ ! -f "${src_dir}/${f}" ]; then
			echo "ERROR: missing file for staging: ${src_dir}/${f}" >&2
			return 1
		fi
		cp -a "${src_dir}/${f}" "${STAGING_DIR}/${f}"
	done
	cp -a "$changes" "${STAGING_DIR}/$(basename "$changes")"
	STAGED_CHANGES="${STAGING_DIR}/$(basename "$changes")"
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
	return 1
}

verify_changes_checksums() {
	local changes="$1"
	local dir failed=0
	dir="$(dirname "$changes")"
	echo "Verifying file checksums against ${changes}..."
	while IFS= read -r line; do
		[[ "$line" =~ ^[[:space:]]+([0-9a-f]{32})[[:space:]]+([0-9]+)[[:space:]]+[^[:space:]]+[[:space:]]+[^[:space:]]+[[:space:]]+([^[:space:]]+) ]] || continue
		local expect="${BASH_REMATCH[1]}"
		local file="${BASH_REMATCH[3]}"
		[[ "$file" == "$(basename "$changes")" ]] && continue
		local path="${dir}/${file}"
		if [ ! -f "$path" ]; then
			echo "ERROR: missing upload file: ${path}" >&2
			failed=1
			continue
		fi
		local actual
		actual="$(md5sum "$path" | awk '{print $1}')"
		if [ "$actual" != "$expect" ]; then
			echo "ERROR: MD5 mismatch for ${file}" >&2
			echo "  .changes expects: ${expect}" >&2
			echo "  file on disk:     ${actual}" >&2
			failed=1
		fi
	done < <(awk '/^Files:/,/^$/' "$changes" | tail -n +2)
	if [ "$failed" -ne 0 ]; then
		return 1
	fi
	echo "Checksum verification passed."
}

verify_gpg_signatures() {
	local changes="$1"
	local dsc="${changes/_source.changes/.dsc}"
	local failed=0

	for f in "$dsc" "$changes"; do
		if [ ! -f "$f" ]; then
			echo "ERROR: missing signed file: ${f}" >&2
			failed=1
			continue
		fi
		if ! grep -q 'BEGIN PGP SIGNATURE' "$f"; then
			echo "ERROR: no embedded signature in ${f}" >&2
			failed=1
			continue
		fi
		if ! gpg --verify "$f" >/dev/null 2>&1; then
			echo "ERROR: gpg --verify failed for ${f}" >&2
			gpg --verify "$f" 2>&1 || true
			failed=1
		fi
	done

	if [ "$failed" -ne 0 ]; then
		echo "Signing failed — do not upload. Fix GPG setup (see ppa.md)." >&2
		return 1
	fi
	echo "GPG signature verification passed (.dsc + .changes)."
}

run_build() {
	local version="$1"
	local maintainer_email="$2"
	local parent="$3"
	local changes will_sign=false signing_key=""

	echo "Building source package for dback ${version}..."

	echo "Cleaning previous ../dback_${version}* artifacts..."
	rm -f "${parent}"/dback_"${version}"*
	rm -f "${parent}"/dback_"${version}"*.upload 2>/dev/null || true
	rm -rf "${parent}"/.dback-upload-stage.* 2>/dev/null || true
	clean_workspace

	local debuild_args=(-S -sa -d)

	case "$SIGN" in
		0 | false | no)
			debuild_args+=(-us -uc)
			echo "NOTE: UNSIGNED=1 — not uploadable to PPA."
			;;
		1 | true | yes)
			if ! signing_key_available "$maintainer_email"; then
				echo "ERROR: SIGN=1 but no GPG secret key for ${maintainer_email}." >&2
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
				echo "GPG key found; debuild will sign with ${signing_key}."
			else
				debuild_args+=(-us -uc)
				echo "WARNING: No GPG key — building unsigned."
			fi
			;;
	esac

	if [ "$will_sign" = true ]; then
		debuild_args+=(-k"$signing_key")
	fi

	# --no-tgz-check / --no-lintian are debuild options (must precede -S -sa -d).
	# Do not pipe yes/printf to debuild: GitHub Actions has no TTY and stdin is ignored.
	if ! debuild --no-lintian --no-tgz-check "${debuild_args[@]}"; then
		echo "ERROR: debuild failed." >&2
		exit 1
	fi

	changes="$(ls -1t "${parent}"/dback_"${version}"_source.changes 2>/dev/null | head -n 1 || true)"
	if [ -z "$changes" ]; then
		echo "ERROR: could not find ../dback_${version}_source.changes" >&2
		exit 1
	fi

	if [ "$will_sign" = true ]; then
		verify_gpg_signatures "$changes"
		verify_changes_checksums "$changes"
	fi

	echo ""
	echo "Source package ready:"
	echo "  ${changes}"
	echo ""
	echo "Upload: dput ppa:devlifex/dback ${changes}"
	echo "Guide:  ppa.md"

	if [ "$UPLOAD" = "1" ]; then
		if [ "$will_sign" = false ]; then
			echo "ERROR: UPLOAD=1 requires signing." >&2
			exit 1
		fi
		upload_from_staging "$changes" "$parent"
	fi
}

upload_from_staging() {
	local changes="$1"
	local parent="$2"
	local staged_changes

	need_cmd dput
	echo ""
	echo "Staging upload artifacts (isolated copy)..."
	stage_upload_artifacts "$changes" "$parent"
	trap cleanup_staging EXIT
	echo "  ${STAGED_CHANGES}"

	echo ""
	echo "Uploading to ${PPA} from staging directory..."
	rm -f "${STAGED_CHANGES%.changes}.ppa.upload"
	verify_gpg_signatures "$STAGED_CHANGES"
	verify_changes_checksums "$STAGED_CHANGES"
	dput_with_retry "$PPA" "$STAGED_CHANGES"
	echo "Upload complete; removing staging directory."
	cleanup_staging
	trap - EXIT
}

configure_gpg_for_environment() {
	# GitHub Actions has no TTY; GPG_TTY=/dev/tty makes debsign fail.
	if [ -n "${GITHUB_ACTIONS:-}" ] || [ -n "${CI:-}" ]; then
		unset GPG_TTY
		return
	fi
	local tty_path
	if tty_path="$(tty 2>/dev/null)" && [ -c "$tty_path" ]; then
		export GPG_TTY="$tty_path"
	else
		unset GPG_TTY
	fi
}

configure_gpg_for_environment
export DEBFULLNAME="${DEBFULLNAME:-Dariush Vesal}"
export DEBEMAIL="${DEBEMAIL:-dvworkmail2017@gmail.com}"
export DEBSIGN_KEYID="${DEBSIGN_KEYID:-$GPG_KEY_ID}"
export DEBIAN_FRONTEND=noninteractive

need_cmd debuild
need_cmd dpkg-parsechangelog
need_cmd gpg
need_cmd flock

version="$(dpkg-parsechangelog -SVersion)"
maintainer_email="$(dpkg-parsechangelog --show-field Maintainer | sed -nE 's/.*<([^>]+)>.*/\1/p')"
parent="$(dirname "$ROOT")"

exec 9>"$LOCK_FILE"
if ! flock -n 9; then
	echo "ERROR: Another PPA build/upload is already running (lock: ${LOCK_FILE})." >&2
	echo "Wait for it to finish, or remove the lock if no build is active." >&2
	exit 1
fi

run_build "$version" "$maintainer_email" "$parent"
