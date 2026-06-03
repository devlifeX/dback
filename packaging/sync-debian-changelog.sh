#!/usr/bin/env bash
# Align debian/changelog with APP_VERSION (e.g. from git tag in CI).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

APP_VERSION="${APP_VERSION:?APP_VERSION is required}"
PPA_DIST="${PPA_DIST:-noble}"
export DEBFULLNAME="${DEBFULLNAME:-Dariush Vesal}"
export DEBEMAIL="${DEBEMAIL:-dvworkmail2017@gmail.com}"

ppa_dist_suffix() {
	case "$1" in
		noble) echo "ubuntu24.04.1" ;;
		jammy) echo "ubuntu22.04.1" ;;
		*)
			echo "ERROR: unknown PPA_DIST '${1}' (use noble or jammy)." >&2
			exit 1
			;;
	esac
}

main_version="$(sed -n 's/^var appVersion = "\(.*\)".*/\1/p' main.go)"
build_version="$(sed -n 's/^APP_VERSION="${APP_VERSION:-\([^}]*\)}".*/\1/p' build.sh)"

if [ "$main_version" != "$APP_VERSION" ]; then
	echo "ERROR: tag version ${APP_VERSION} != main.go (${main_version})" >&2
	exit 1
fi
if [ "$build_version" != "$APP_VERSION" ]; then
	echo "ERROR: tag version ${APP_VERSION} != build.sh (${build_version})" >&2
	exit 1
fi

suffix="$(ppa_dist_suffix "$PPA_DIST")"
deb_version="${APP_VERSION}-1~${suffix}"

current_version="$(dpkg-parsechangelog -SVersion)"
current_dist="$(dpkg-parsechangelog -SDistribution)"

if [ "$current_version" = "$deb_version" ] && [ "$current_dist" = "$PPA_DIST" ]; then
	echo "debian/changelog already at ${current_version} (${PPA_DIST})"
	exit 0
fi

echo "Updating debian/changelog -> ${deb_version} (${PPA_DIST})"
dch -v "${deb_version}" -D "${PPA_DIST}" --force-bad-version "Release ${APP_VERSION} for ${PPA_DIST}."
