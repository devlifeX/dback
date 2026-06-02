#!/usr/bin/env bash
# Import the Launchpad signing key in GitHub Actions (non-interactive).
set -euo pipefail

if [ -z "${PPA_GPG_PRIVATE_KEY:-}" ]; then
	cat >&2 <<'EOF'
ERROR: GitHub secret PPA_GPG_PRIVATE_KEY is missing or empty.

Fix:
  1. Open: https://github.com/devlifeX/dback/settings/secrets/actions
  2. New repository secret
  3. Name must be EXACTLY: PPA_GPG_PRIVATE_KEY
  4. Value = full armored private key:

       gpg --armor --export-secret-keys E00C906928B7599C

  5. Also add PPA_GPG_PASSPHRASE (use a single space if the key has no passphrase)

Notes:
  - Create under "Actions" secrets, not Dependabot/Codespaces.
  - Organization secrets must allow access to this repository.
  - This workflow runs on tag push only; secrets are not available on fork PRs.
EOF
	exit 1
fi

mkdir -p ~/.gnupg
chmod 700 ~/.gnupg

cat >> ~/.gnupg/gpg.conf <<'EOF'
default-key E8B25AD68688EC024359E03FE00C906928B7599C
pinentry-mode loopback
EOF

cat >> ~/.gnupg/gpg-agent.conf <<'EOF'
allow-loopback-pinentry
allow-preset-passphrase
EOF

gpgconf --kill gpg-agent 2>/dev/null || true

gpg --batch --yes --pinentry-mode loopback \
	--passphrase "${PPA_GPG_PASSPHRASE:-}" \
	--import <<< "$PPA_GPG_PRIVATE_KEY"

KEYGRIP="$(gpg --with-keygrip --list-secret-keys E8B25AD68688EC024359E03FE00C906928B7599C 2>/dev/null \
	| awk '/Keygrip =/ { print $3; exit }')"

if [ -n "${PPA_GPG_PASSPHRASE:-}" ] && [ -n "$KEYGRIP" ]; then
	if [ -x /usr/lib/gnupg2/gpg-preset-passphrases ]; then
		echo "$PPA_GPG_PASSPHRASE" | /usr/lib/gnupg2/gpg-preset-passphrases --passphrase-fd 0 --preset "$KEYGRIP"
	elif [ -x /usr/lib/gnupg/gpg-preset-passphrases ]; then
		echo "$PPA_GPG_PASSPHRASE" | /usr/lib/gnupg/gpg-preset-passphrases --passphrase-fd 0 --preset "$KEYGRIP"
	fi
fi

gpgconf --reload gpg-agent 2>/dev/null || true

if ! gpg --list-secret-keys --keyid-format LONG dvworkmail2017@gmail.com 2>/dev/null | grep -q '^sec'; then
	echo "ERROR: GPG secret key import failed." >&2
	exit 1
fi

echo "GPG signing key imported successfully."
