# PPA publishing

See **[`ppa.md`](../ppa.md)** at the repository root for the full Launchpad PPA guide (GPG, build, upload, troubleshooting).

**Current app version:** `3.8.0` — must match `main.go`, `build.sh`, and the release git tag `v3.8.0`.

## Pre-release checklist

| File | Field |
|------|-------|
| `main.go` | `appVersion = "3.8.0"` |
| `build.sh` | `APP_VERSION="${APP_VERSION:-3.8.0}"` |
| `debian/changelog` | e.g. `dback (3.8.0-1~ubuntu24.04.1) noble` |

Sync changelog before a manual PPA upload:

```bash
PPA_DIST=noble APP_VERSION=3.8.0 ./packaging/sync-debian-changelog.sh
# jammy (separate upload):
PPA_DIST=jammy APP_VERSION=3.8.0 ./packaging/sync-debian-changelog.sh
```

Note: Launchpad builds use Go 1.22. `golang-1.22-go` installs `go` under
`/usr/lib/go-1.22/bin`, so `debian/rules` exports that directory in `PATH`.
If a PPA build fails with `/bin/sh: 1: go: not found`, see the troubleshooting
section in [`ppa.md`](../ppa.md).

## Quick publish

```bash
export GPG_TTY=$(tty)
UPLOAD=1 ./packaging/build-ppa.sh
```

## CI release (recommended)

```bash
git tag v3.8.0
git push origin master v3.8.0
```

GitHub Actions runs `go.yml` (GitHub Release) and `ppa.yml` (noble + jammy matrix).
