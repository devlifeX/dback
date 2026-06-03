# PPA publishing

See **[`ppa.md`](../ppa.md)** at the repository root for the full Launchpad PPA guide (GPG, build, upload, troubleshooting).

Note: Launchpad builds use Go 1.22. `golang-1.22-go` installs `go` under
`/usr/lib/go-1.22/bin`, so `debian/rules` exports that directory in `PATH`.
If a PPA build fails with `/bin/sh: 1: go: not found`, see the troubleshooting
section in [`ppa.md`](../ppa.md).

Quick publish:

```bash
export GPG_TTY=$(tty)
UPLOAD=1 ./packaging/build-ppa.sh
```
