#!/usr/bin/env bash
# Align debian/changelog with APP_VERSION (e.g. from git tag in CI).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

APP_VERSION="${APP_VERSION:?APP_VERSION is required}"
export DEBFULLNAME="${DEBFULLNAME:-Dariush Vesal}"
export DEBEMAIL="${DEBEMAIL:-dvworkmail2017@gmail.com}"

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

current_version="$(dpkg-parsechangelog -SVersion | sed 's/-.*//')"
if [ "$current_version" = "$APP_VERSION" ]; then
	echo "debian/changelog already at ${APP_VERSION}"
	exit 0
fi

echo "Updating debian/changelog: ${current_version} -> ${APP_VERSION}"
dch -v "${APP_VERSION}" -D ubuntu --force-bad-version "Release ${APP_VERSION}."
